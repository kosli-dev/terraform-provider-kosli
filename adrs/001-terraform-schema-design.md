---
title: "ADR 001: Terraform Schema Design for Kosli Attestation Types API"
description: "Deciding on the Terraform schema design for managing Kosli Attestation Types via the Kosli API."
status: "Accepted"
date: "2025-01-07"
---

# ADR 001: Terraform Schema Design for Kosli Attestation Types API

## Context

The Kosli Terraform provider needs to manage custom attestation types through the Kosli API. The API uses a specific format for evaluation rules that differs from typical Terraform patterns:

**Kosli API Format:**
```json
{
  "name": "person-over-21",
  "description": "A person that is over 21 year",
  "evaluator": {
    "content_type": "jq",
    "rules": [".age > 21"]
  },
  "schema": {...}
}
```

We need to decide how to represent this in the Terraform provider schema - whether to mirror the API structure exactly or provide a simplified user experience.

## Decision Drivers

1. **User Experience** - Terraform should be easy to write and understand
2. **API Compatibility** - Must correctly communicate with Kosli API
3. **Future Extensibility** - May support other content types beyond "jq"
4. **Terraform Conventions** - Follow common patterns in Terraform providers

## Options Considered

### Option A: Simplified Schema with Transformation (Recommended)

**Rationale:** The Terraform provider should have a user-friendly schema that abstracts API details. Since `content_type` is always "jq" (only supported type), we can keep it simple.

**Terraform Schema:**
```hcl
resource "kosli_custom_attestation_type" "example" {
  name        = "person-over-21"
  description = "A person that is over 21 year"
  schema      = <<-EOT {...} EOT

  jq_rules = [  # User-friendly field name
    ".age > 21"
  ]
}
```

**Provider Implementation:**
- Accept `jq_rules` in Terraform schema
- Transform to `evaluator` object when calling API:
  ```go
  evaluator := map[string]any{
    "content_type": "jq",
    "rules": jqRules,
  }
  ```

**Pros:**
- Simpler user experience
- Less verbose
- No need to specify content_type (always "jq")
- Maintains backwards compatibility if we implement it now

**Cons:**
- Abstraction hides API structure
- If other content types are added later, requires schema change

### Option B: Mirror API Structure Exactly

**Terraform Schema:**
```hcl
resource "kosli_custom_attestation_type" "example" {
  name        = "person-over-21"
  description = "A person that is over 21 year"
  schema      = <<-EOT {...} EOT

  evaluator {
    content_type = "jq"
    rules = [
      ".age > 21"
    ]
  }
}
```

**Pros:**
- Matches API structure 1:1
- Easier to add new content types in future
- More explicit about API structure

**Cons:**
- More verbose for users
- Requires nested block
- Content type field is unnecessary (always "jq" currently)

---

## Decision

**Selected: Option A (Simplified Schema with Transformation)**

### Rationale

1. **User preference** - Explicit choice for simplified approach
2. **Content type uncertainty** - Future support unclear, so keeping flexible
3. **Better UX** - Simpler for 99% use case (jq-only)
4. **Terraform conventions** - Providers often abstract API complexity

### Trade-offs Accepted

- **Abstraction over explicitness**: We're hiding the evaluator structure
- **Future migration risk**: If other content types are added, may need schema changes
- **Mitigation**: Can add `content_type` parameter later with "jq" as default

## Consequences

### Positive

- **Simpler user experience** - Users write less code
- **Current examples remain valid** - No breaking changes needed
- **Follows Terraform patterns** - Similar to other providers that abstract API details
- **Easier to understand** - Less nesting, clearer intent

### Negative

- **API abstraction** - Users don't see actual API structure
- **Potential future migration** - If content types are added, may need schema evolution
- **Implementation complexity** - Provider must transform between formats

### Neutral

- **Documentation burden** - Need to clearly explain the transformation
- **Testing requirements** - Must test transformation logic thoroughly

## Implementation

### Provider Schema

```hcl
resource "kosli_custom_attestation_type" "example" {
  name        = "person-over-21"
  description = "A person that is over 21 year"
  schema      = <<-EOT {...} EOT
  jq_rules    = [".age > 21"]
}
```

### API Transformation

The provider transforms the `jq_rules` attribute to the API's `evaluator` format:

**Outgoing (Create/Update):**
```go
evaluator := map[string]any{
    "content_type": "jq",
    "rules":        jqRules,
}
```

**Incoming (Read):**
```go
jqRules := response["evaluator"]["rules"]
```

### Documentation

User-facing documentation explains that evaluation rules are jq expressions that must return `true` for compliance. The internal transformation to the `evaluator` format is noted but not emphasized, keeping the focus on the user experience.

### Schema Input Methods

The `schema` attribute accepts a JSON Schema string, which can be provided in multiple ways to suit different use cases:

#### 1. Inline Heredoc (Recommended for simple schemas)
```hcl
resource "kosli_custom_attestation_type" "example" {
  schema = <<-EOT
    {
      "type": "object",
      "properties": {"age": {"type": "integer"}},
      "required": ["age"]
    }
  EOT
}
```

#### 2. External File (Recommended for complex schemas)
```hcl
resource "kosli_custom_attestation_type" "example" {
  schema = file("${path.module}/schemas/coverage-schema.json")
}
```

#### 3. Schema Data Source (Recommended for reusable, type-safe schemas)

Following AWS provider patterns (e.g., `aws_iam_policy_document`), a `kosli_attestation_schema` data source provides type-safe schema definition:

```hcl
data "kosli_attestation_schema" "coverage_schema" {
  type = "object"
  properties = {
    line_coverage = {
      type    = "number"
      minimum = 0
      maximum = 100
    }
    branch_coverage = {
      type    = "number"
      minimum = 0
      maximum = 100
    }
  }
  required = ["line_coverage"]
}

resource "kosli_custom_attestation_type" "coverage_check" {
  name        = "coverage-check"
  description = "Validate test coverage"
  schema      = data.kosli_attestation_schema.coverage_schema.json
  jq_rules    = [".line_coverage >= 80"]
}
```

**Benefits of the data source approach:**
- **Reusability**: Share schemas across multiple attestation types
- **Type safety**: Terraform validates schema structure at plan time
- **Composability**: Build complex schemas from simpler components
- **Terraform conventions**: Follows established patterns from AWS and other major providers
- **No API changes required**: The data source generates a JSON string that's passed to the API like any other schema

**Implementation:**
- The data source generates valid JSON Schema from HCL structure
- All three methods produce the same API request format
- Users choose the method that best fits their use case
