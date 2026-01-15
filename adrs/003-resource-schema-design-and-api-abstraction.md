---
title: "ADR 003: Resource Schema Design and API Abstraction"
description: "Architectural decisions for Terraform resource schemas, field exposure, and API versioning abstraction"
status: "Proposed"
date: "2026-01-15"
---

# ADR 003: Resource Schema Design and API Abstraction

## Context

The Kosli Terraform provider manages custom attestation types through a resource (`kosli_custom_attestation_type`) and data source (`data.kosli_custom_attestation_type`). The Kosli API returns attestation types with a rich structure including version history, timestamps, creator information, and archive status.

Key questions emerged during development:

1. **`archived` field**: Should this API-level field be exposed in the Terraform resource schema?
2. **`org` field**: Should organization context be exposed as a resource attribute?
3. **Version history**: How should we handle the API's `versions[]` array abstraction?
4. **Field exposure**: Which API fields should be exposed to Terraform users vs hidden?

### API Response Structure

The Kosli API returns attestation types in this format:

```json
{
  "name": "code-coverage",
  "description": "Validates code coverage metrics",
  "archived": false,
  "org": "dangrondahl",
  "versions": [
    {
      "version": 1,
      "timestamp": 1768247330.112509,
      "type_schema": "{'type': 'object', 'properties': {...}}",
      "evaluator": {
        "content_type": "jq",
        "rules": [".line_coverage >= 80"]
      },
      "created_by": "Dan Grøndahl"
    }
  ]
}
```

### Current Implementation

The client layer transforms this to a user-facing format:

```go
type CustomAttestationType struct {
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Schema      string    `json:"-"`        // Extracted from latest version
    JqRules     []string  `json:"-"`        // Extracted from evaluator
    Versions    []Version `json:"versions"` // API format (hidden from users)
    Archived    bool      `json:"archived"`
    Org         string    `json:"org"`
}
```

The Terraform provider exposes these fields:
- `name` (Required, ForceNew)
- `description` (Optional)
- `schema` (Required, uses jsontypes.Normalized)
- `jq_rules` (Required)
- `archived` (Computed) ← **Question: Should this exist?**
- `org` (Computed) ← **Question: Known at plan time or apply time?**

## Decision Drivers

- **Terraform lifecycle patterns**: Resources have create/read/update/delete operations; state should reflect current resource state only
- **User experience**: Fields shown at plan time vs "known after apply" significantly impacts usability
- **Maintainability**: Simpler schemas are easier to maintain and evolve
- **Industry best practices**: AWS and Azure providers show account/subscription context at plan time and don't expose audit metadata
- **API abstraction philosophy**: Balance between transparency (showing what the API provides) and simplicity (showing only what users need)
- **Versioning semantics**: Terraform manages current state; version history is an audit/API concern

## Decisions

### Decision 1: Remove `archived` Field from Resource Schema

**Decision**: Remove the `archived` attribute from the resource schema. Keep it only in the data source schema for read-only queries.

#### Rationale

1. **Terraform lifecycle incompatibility**: In Terraform's lifecycle, the DELETE operation removes the resource from state entirely. The `archived` field would only be visible during the brief moment between the API archive call and state removal—effectively never visible to users in normal workflows.

2. **Not a user-configurable field**: Users cannot set `archived = true` to archive a resource. The only way to archive is through `terraform destroy`, which removes it from state anyway.

3. **API implementation detail**: Whether the API performs a "soft delete" (archive) or "hard delete" is an implementation detail that Terraform users don't need to manage. From a Terraform perspective, the resource either exists or doesn't exist.

4. **Clearer mental model**: Without `archived`, the resource behavior is simpler:
   - Resource exists → managed by Terraform
   - Resource destroyed → removed from state
   - No intermediate "archived but still in state" state to reason about

5. **Data source still provides visibility**: Users who need to query archived types can use the data source, which retains the `archived` field for read-only access.

#### Alternatives Considered

**Alternative A: Keep `archived` as Computed field (current implementation)**
- ❌ Field is never visible in practice (removed from state immediately after being set)
- ❌ Confusing to users: "Why is this field here if I can't use it?"
- ❌ Inconsistent with Terraform patterns

**Alternative B: Make `archived` Optional/Required to enable soft deletes**
- ❌ Changes Terraform semantics: `terraform destroy` wouldn't actually remove the resource
- ❌ Users would need to manually set `archived = false` to "undelete"
- ❌ Conflicts with Terraform's lifecycle management
- ✅ Would enable interesting use cases (archive instead of delete)
- **Verdict**: Interesting but violates Terraform conventions. Could revisit if strong user demand emerges.

#### Consequences

**Positive:**
- ✅ Cleaner schema that reflects actual Terraform usage patterns
- ✅ Simpler mental model for users
- ✅ Removes confusing "known after apply" field that's never actually observed
- ✅ Aligns with Terraform best practices

**Negative:**
- ⚠️ Users cannot see archive status through the resource (must use data source or API)

**Neutral:**
- The API still performs archive operations; only the Terraform schema changes
- Delete behavior remains unchanged (calls archive endpoint)

#### Migration Path

**Note**: Since the provider has not yet been released, there are no existing state files to migrate. This change can be implemented cleanly in the initial release without backward compatibility concerns.

---

### Decision 2: Remove `org` Field Entirely

**Decision**: Remove the `org` attribute from both resource and data source schemas. The organization context is provider-level configuration, not resource-level data.

#### Rationale

1. **Provider context, not resource data**: The `org` field is configured at the provider level and determines the scope of all operations. It's not resource-specific configuration.

2. **Not user-configurable per resource**: Users cannot set `org` on individual resources—it's always inherited from the provider configuration. Exposing it as an attribute implies it can be controlled, which is misleading.

3. **Not part of resource identity**: Resources are identified by `name`, not `org`. The org is implicit context from the provider configuration.

4. **Industry standard pattern**: Major cloud providers follow this pattern:
   - **AWS**: Resources don't have `account_id` attribute (even though APIs return it)
   - **Azure**: Resources don't have `subscription_id` attribute per resource
   - **Google Cloud**: Resources don't have `project_id` attribute per resource
   - Provider-level context (account, subscription, project, org) is implicit, not exposed per-resource

5. **Redundant information**: Users already know their org from the provider block configuration. Repeating it in every resource adds noise without value.

6. **Internal use only**: The client uses `org` to build API paths (`/custom-attestation-types/{org}/{name}`), but this is an implementation detail that doesn't need to be surfaced to users.

#### Alternatives Considered

**Alternative A: Keep as Computed, make it known at plan time**
- ✅ Would show org value at plan time
- ✅ More accurate than "known after apply"
- ❌ Still exposes provider context as resource attribute
- ❌ Doesn't match industry patterns (AWS, Azure, GCP)
- ❌ Adds field noise without clear value
- **Verdict**: Improvement over current, but still not optimal

**Alternative B: Keep as Computed-only (current implementation)**
- ❌ Shows "(known after apply)" for predictable value
- ❌ Misleading to users
- ❌ Doesn't match industry patterns
- **Verdict**: Worst option—neither useful nor accurate

**Alternative C: Make `org` Optional at resource level**
- ✅ Would allow per-resource org override
- ❌ Kosli API doesn't support cross-org operations from single credentials
- ❌ Adds significant complexity for unclear benefit
- **Verdict**: Over-engineering with no use case

**Alternative D: Remove entirely (CHOSEN)**
- ✅ Matches industry best practices (AWS, Azure, GCP)
- ✅ Cleaner schema focused on user-configurable data
- ✅ Removes redundant information
- ✅ Org remains available internally for API calls
- ✅ Provider configuration clearly documents the org context
- **Verdict**: Simplest and most aligned with Terraform conventions

#### Consequences

**Positive:**
- ✅ Cleaner, simpler schema
- ✅ Matches industry patterns (AWS, Azure, GCP)
- ✅ Removes redundant information from state
- ✅ Clearer separation: provider config = context, resource = configuration
- ✅ Users reference provider configuration if they need org value

**Negative:**
- ⚠️ Users cannot see org in resource outputs (must reference provider configuration)
- ⚠️ Cannot use `resource.org` in interpolations (but provider.org is available)

**Neutral:**
- Org is still used internally by the client for API paths
- Provider configuration clearly documents the org scope
- No impact on functionality—only schema representation

#### Implementation

**Remove from resource:**
1. Remove `Org` field from `customAttestationTypeResourceModel` struct
2. Remove `"org"` from resource schema definition
3. Remove lines that populate `data.Org` from API responses

**Remove from data source:**
1. Remove `Org` field from `customAttestationTypeDataSourceModel` struct
2. Remove `"org"` from data source schema definition
3. Remove lines that populate `data.Org` from API responses

**Client remains unchanged:**
- `CustomAttestationType` struct keeps `Org string` field (for API parsing)
- Client continues to use `c.Organization()` for building API paths
- `fromAPIFormat()` can ignore the org field (it's in the API response but we don't expose it)

---

### Decision 3: Hide Version History, Expose Only Latest Version

**Decision**: Continue the current abstraction strategy of hiding the API's `versions[]` array and exposing only the latest version's `schema` and `jq_rules` to Terraform users.

#### Rationale

1. **Terraform manages current state, not history**: Terraform's state management is about the current desired state of infrastructure, not historical states. Version history is fundamentally incompatible with Terraform's model.

2. **No actionable use cases**: Users cannot take action on version history through Terraform:
   - Cannot rollback to previous versions (would require API support for version selection)
   - Cannot query version metadata (timestamp, creator) for infrastructure management
   - Cannot configure based on version history

3. **Audit concern, not IaC concern**: Version history, timestamps, and creator information are audit trail data, not infrastructure configuration. These belong in logging/audit systems, not infrastructure-as-code state.

4. **Simpler mental model**: Users think about "what schema and rules do I want?" not "what's the version history of my attestation type?"

5. **API abstraction philosophy**: The client layer already extracts the latest version. Exposing the raw `versions[]` array would violate the abstraction layer and leak API implementation details.

#### What's Hidden from Users

- `versions[]` array
- `version` number (integer)
- `timestamp` (Unix timestamp)
- `created_by` (username string)
- Historical versions (previous schemas/rules)

#### What's Exposed to Users

- `schema` (string) - Extracted from `versions[0].type_schema`
- `jq_rules` (list of strings) - Extracted from `versions[0].evaluator.rules`

#### Alternatives Considered

**Alternative A: Expose read-only version metadata (number, timestamp, creator)**
- ✅ Provides audit information
- ❌ Not actionable in Terraform workflows
- ❌ Adds complexity for unclear benefit
- ❌ Timestamps and creators belong in audit logs, not IaC state
- **Verdict**: Over-engineering without clear use case

**Alternative B: Add version selection capability in data source**
- ✅ Would enable querying specific historical versions
- ✅ Could support rollback scenarios
- ❌ Complex implementation (new query parameter, state management)
- ❌ No clear use case for version rollback through Terraform
- ❌ Users can update schema/rules directly without version mechanics
- **Verdict**: Interesting but premature. Can add later if demand emerges.

**Alternative C: Expose full version history as a list attribute**
- ✅ Complete transparency of API data
- ❌ Most complex option
- ❌ Version history is immutable, adds noise to state diffs
- ❌ No actionable use cases
- ❌ Violates Terraform's "current state" philosophy
- **Verdict**: Maximum transparency, minimal value

#### Consequences

**Positive:**
- ✅ Simpler resource schema
- ✅ Users focus on "what" not "how" (desired state, not version mechanics)
- ✅ Clear separation of concerns: Terraform = IaC, Kosli API/CLI = audit trail
- ✅ Easier to maintain and evolve

**Negative:**
- ⚠️ Cannot query version history through Terraform
- ⚠️ Cannot rollback to previous versions through Terraform
- ⚠️ Users needing audit data must use Kosli CLI/API/Web UI

**Neutral:**
- Version mechanics still work on API side (POST creates new versions when payload changes)
- Client layer preserves `Versions` struct for internal use

#### Future Extensibility

If version selection becomes a critical use case, we can add:

```hcl
# New data source for version-specific queries
data "kosli_custom_attestation_type_version" "v2" {
  name    = "security-scan"
  version = 2  # Optional; defaults to latest
}
```

This would be a non-breaking addition that separates "current state management" (resource) from "historical queries" (specialized data source).

---

### Decision 4: Current Field Set is Sufficient

**Decision**: The current field set adequately covers user needs. Do not expose additional API fields at this time.

#### Current Field Set

**Resource Attributes:**
- `name` (string, Required, ForceNew) - Attestation type identifier
- `description` (string, Optional) - Human-readable description
- `schema` (string, Required, jsontypes.Normalized) - JSON Schema definition
- `jq_rules` (list(string), Required) - Evaluation rules

**Removed from Resource:**
- ~~`archived` (bool, Computed)~~ - Removed per Decision 1; kept in data source only
- ~~`org` (string, Computed)~~ - Removed per Decision 2; provider-level context, not resource data

#### Fields Not Exposed (and Why)

| API Field | Why Not Exposed |
|-----------|----------------|
| `org` | See Decision 2 - provider-level context, not resource data |
| `versions[]` | See Decision 3 - history not relevant to IaC |
| `version` number | See Decision 3 - part of hidden version history |
| `timestamp` | Audit data, not configuration data |
| `created_by` | Audit data, not configuration data |
| `evaluator.content_type` | Currently always "jq"; can add later if other types emerge |

#### Rationale

1. **Minimalism**: Expose only what users need to configure their infrastructure. Extra fields create cognitive overhead and maintenance burden.

2. **Current usage patterns**: All current Kosli users use "jq" as `content_type`. No need to expose what's effectively a constant.

3. **Future extensibility**: If the API adds support for other evaluator types (e.g., "rego", "cel"), we can add `content_type` as an Optional field with "jq" default. This would be a non-breaking change.

4. **Audit data belongs elsewhere**: Timestamps and creator information should be accessed through audit logs, not infrastructure state.

#### Alternatives Considered

**Alternative A: Add `evaluator.content_type` field now**
- ✅ Future-proofs against new evaluator types
- ❌ Adds field that's always "jq" today (noise)
- ❌ No user demand or API support for other types
- **Verdict**: YAGNI (You Aren't Gonna Need It). Add when actually needed.

**Alternative B: Add version metadata (number, timestamp, creator)**
- ❌ Already covered in Decision 3 - not relevant to IaC

#### Consequences

**Positive:**
- ✅ Clean, focused schema
- ✅ Easy to understand and use
- ✅ Low maintenance burden
- ✅ Can extend without breaking changes

**Negative:**
- (None identified—can always add fields later)

**Neutral:**
- Users wanting audit data use Kosli CLI/API/Web UI

## Overall Consequences

### Practical Impact: Before and After

Here's how these decisions affect the actual Terraform configuration and plan output:

#### Resource Definition

| Aspect | Before (with archived + org) | After (clean schema) |
|--------|------------------------------|----------------------|
| **Schema Fields** | `name`, `description`, `schema`, `jq_rules`, `archived`, `org` | `name`, `description`, `schema`, `jq_rules` |
| **User-Configurable** | 4 fields (`name`, `description`, `schema`, `jq_rules`) | 4 fields (same) |
| **Computed Fields** | 2 fields (`archived`, `org`) | 0 fields |
| **Field Count** | 6 total | 4 total |

#### Terraform Plan Output

**Before:**
```hcl
# kosli_custom_attestation_type.code_coverage will be created
+ resource "kosli_custom_attestation_type" "code_coverage" {
    + archived    = (known after apply)  # ❌ Confusing: never actually observed
    + description = "Validates code coverage metrics"
    + jq_rules    = [
        + ".line_coverage >= 80",
        + ".branch_coverage >= 70",
      ]
    + name        = "code-coverage"
    + org         = (known after apply)  # ❌ Redundant: always same as provider
    + schema      = jsonencode({
        + properties = { ... }
        + type       = "object"
      })
  }
```

**After:**
```hcl
# kosli_custom_attestation_type.code_coverage will be created
+ resource "kosli_custom_attestation_type" "code_coverage" {
    + description = "Validates code coverage metrics"
    + jq_rules    = [
        + ".line_coverage >= 80",
        + ".branch_coverage >= 70",
      ]
    + name        = "code-coverage"
    + schema      = jsonencode({
        + properties = { ... }
        + type       = "object"
      })
  }
```

✅ **Cleaner**: Only shows user-configurable fields
✅ **No confusion**: No "(known after apply)" for fields that add no value
✅ **Focused**: Users see exactly what they control

#### Terraform State

**Before:**
```json
{
  "mode": "managed",
  "type": "kosli_custom_attestation_type",
  "name": "code_coverage",
  "attributes": {
    "name": "code-coverage",
    "description": "Validates code coverage metrics",
    "schema": "{\"type\":\"object\",\"properties\":{...}}",
    "jq_rules": [".line_coverage >= 80"],
    "archived": false,
    "org": "my-org"
  }
}
```

**After:**
```json
{
  "mode": "managed",
  "type": "kosli_custom_attestation_type",
  "name": "code_coverage",
  "attributes": {
    "name": "code-coverage",
    "description": "Validates code coverage metrics",
    "schema": "{\"type\":\"object\",\"properties\":{...}}",
    "jq_rules": [".line_coverage >= 80"]
  }
}
```

✅ **33% smaller state** (4 fields vs 6)
✅ **No redundant data** (org implicit from provider)
✅ **No misleading fields** (archived never meaningful in state)

#### Data Source Query

**Before and After (data source unchanged):**
```hcl
data "kosli_custom_attestation_type" "security" {
  name = "security-scan"
}

# Outputs:
# - name
# - description
# - schema
# - jq_rules
# - archived    ← Still available for read-only queries
# - org         ← Removed (matches resource removal)
```

⚠️ **Note**: Data source also has `org` removed for consistency, but keeps `archived` for querying deleted types.

### Breaking Changes

**None** - Since the provider has not been released yet, these changes will be part of the initial v0.1.0 release. No migration path needed.

### Improvements

- ✅ **33% fewer fields** in state (4 vs 6 attributes)
- ✅ **Zero computed fields** that add noise
- ✅ **Removed `org` field** - matches industry patterns (AWS, Azure, GCP)
- ✅ **Removed `archived` field** - never visible in normal Terraform lifecycle
- ✅ **Cleaner plan output** - only shows user-configurable data
- ✅ **Clear abstraction boundaries**: provider config = context, resource = configuration, API = history/audit

### Technical Debt

- None introduced. These decisions simplify the codebase.

## Implementation Guide

### For Provider Developers

**Implementing Decision 1 (Remove `archived`)**:
1. Remove `Archived` from resource model struct
2. Remove `archived` from resource schema
3. Remove archived references in Create/Read/Update methods
4. Keep `archived` in data source schema (read-only queries)
5. Update tests

**Implementing Decision 2 (Remove `org`)**:
1. Remove `Org` field from resource and data source model structs
2. Remove `"org"` attribute from resource and data source schemas
3. Remove code that populates `data.Org` from API responses
4. Update tests to remove `org` field assertions
5. Client struct keeps `Org` for API parsing but doesn't expose to provider

## Related ADRs

- [ADR-002: API Client Architecture and Transformation Layer](./002-api-client-architecture.md) - Establishes the abstraction strategy for transforming API responses

## References

- [Terraform Plugin Framework - Computed Attributes](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes)
- [Terraform Plugin Framework - Plan Modifiers](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
- [AWS Provider - Account ID Handling](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity)
- [Azure Provider - Subscription ID Handling](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/subscription)
- [HashiCorp Provider Design Principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles)
