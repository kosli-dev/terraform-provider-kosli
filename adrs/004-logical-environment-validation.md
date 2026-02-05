---
title: "ADR 004: Logical Environment Validation Strategy"
description: "Deciding whether to perform client-side validation for logical environment constraints or rely on API validation."
status: "Proposed"
date: "2026-02-05"
---

# ADR 004: Logical Environment Validation Strategy

## Context

Logical environments in Kosli aggregate physical environments for organizational purposes. They have a critical design constraint:

**Logical environments can ONLY contain physical environments**, not other logical environments.

This constraint prevents circular dependencies and simplifies the aggregation model. When implementing the `kosli_logical_environment` Terraform resource, we need to decide where to enforce this validation.

## Decision Drivers

1. **Consistency with Kosli CLI** - Follow established patterns in the Kosli ecosystem
2. **Thin wrapper philosophy** - ADR 002 establishes that the API client should be a thin wrapper that reflects API behavior
3. **Validation logic ownership** - Centralized vs. distributed validation
4. **User experience** - Clear error messages and immediate feedback
5. **Maintenance burden** - Keeping validation logic synchronized across clients

## Options Considered

### Option A: Client-Side Validation

Terraform provider queries each environment in `included_environments` to verify it's physical before creating/updating the logical environment.

**Implementation:**
```go
// In Create/Update operation
for _, envName := range includedEnvironments {
    env, err := client.GetEnvironment(ctx, envName)
    if err != nil {
        return diag.FromErr(err)
    }
    if env.Type == "logical" {
        return diag.Errorf("logical environment cannot include another logical environment: %s", envName)
    }
}
// Proceed with create/update
```

**Pros:**
- Immediate feedback before API call
- Custom error messages specific to Terraform users
- Reduces invalid API requests
- Can provide richer validation context

**Cons:**
- Additional API calls (N queries for N environments)
- Validation logic duplicated across clients
- Must stay synchronized with API rules
- Performance impact (especially for large lists)
- Violates thin wrapper principle from ADR 002

### Option B: API Validation (Recommended)

Send the request to the API and let the API enforce validation rules, surfacing API errors as Terraform diagnostics.

**Implementation:**
```go
// In Create/Update operation
err := client.CreateEnvironment(ctx, req)
if err != nil {
    // API will return error if logical env is included
    return diag.FromErr(err)
}
```

**Pros:**
- Single source of truth for validation (API)
- Follows thin wrapper principle (ADR 002)
- Consistent with Kosli CLI approach
- No additional API calls
- Automatic consistency when API rules change
- Simpler provider code

**Cons:**
- Error feedback after API call (not before)
- Error messages determined by API (less customization)
- Invalid requests reach the API

---

## Decision

**Selected: Option B (API Validation)**

### Rationale

1. **Kosli CLI precedent**: Investigation of `kosli-dev/cli` repository shows the CLI follows this approach:
   - `createEnvironment.go` accepts `--included-environments` without type validation
   - `joinEnvironment.go` joins environments without checking types
   - Both commands rely on API to enforce rules

2. **ADR 002 alignment**: The API client architecture decision (ADR 002) establishes a thin wrapper philosophy that "transparently reflects API behavior"

3. **Single source of truth**: Validation rules live in one place (API), reducing risk of desynchronization

4. **Maintenance efficiency**: API changes automatically propagate to all clients without client updates

### Trade-offs Accepted

- **Delayed feedback**: Users don't discover validation errors until API call (typically <1s delay)
- **Less customized errors**: Error messages come from API rather than Terraform-specific text
- **Mitigation**: API errors are still surfaced as clear Terraform diagnostics via `diag.FromErr()`

## Consequences

### Positive

- **Consistency**: All Kosli clients (CLI, Terraform, future clients) behave identically
- **Simplicity**: Provider code is simpler and easier to maintain
- **Performance**: No additional API calls for validation
- **Reliability**: Validation logic can't drift from API rules
- **Future-proof**: New constraints added to API automatically apply

### Negative

- **Error timing**: Validation errors appear during `terraform apply` rather than `terraform plan`
- **Error messages**: Cannot customize error text for Terraform-specific guidance
- **No pre-flight checks**: Invalid configurations aren't caught before API interaction

### Neutral

- **Testing**: Must test with real API or mock API errors (not custom validation logic)
- **Documentation**: Should clearly document the constraint even though provider doesn't enforce it

## Implementation

### Resource Schema

The `kosli_logical_environment` resource schema documents but doesn't enforce the constraint:

```hcl
resource "kosli_logical_environment" "prod" {
  name        = "production-aggregate"
  description = "All production environments"

  # IMPORTANT: Only physical environments allowed
  # Attempting to include logical environments will fail
  included_environments = [
    "prod-k8s",    # ✓ physical
    "prod-ecs",    # ✓ physical
    "prod-lambda", # ✓ physical
  ]
}
```

### Error Handling

The provider passes through API errors as Terraform diagnostics:

```go
func (r *LogicalEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // ... prepare request ...

    err := r.client.CreateEnvironment(ctx, createReq)
    if err != nil {
        // API error surfaced directly to user
        resp.Diagnostics.AddError(
            "Error creating logical environment",
            fmt.Sprintf("Could not create logical environment %s: %s", name, err),
        )
        return
    }
}
```

### Documentation

Documentation explicitly states the constraint:

> **Important**: Logical environments can only include physical environments. Attempting to include another logical environment will result in an error from the Kosli API.

### Testing

Acceptance tests verify the API enforces the constraint:

```go
func TestAccLogicalEnvironment_RejectsLogicalInclusion(t *testing.T) {
    // Test that API rejects logical-in-logical
    // Verifies error propagates correctly to Terraform
}
```

## Related Decisions

- **ADR 002**: API Client Architecture - Establishes thin wrapper principle
- **Issue #75**: Logical environment implementation
- **Issue #92**: Provider resource implementation

## References

- [Kosli CLI createEnvironment.go](https://github.com/kosli-dev/cli/blob/main/cmd/kosli/createEnvironment.go) - No client-side validation
- [Kosli CLI joinEnvironment.go](https://github.com/kosli-dev/cli/blob/main/cmd/kosli/joinEnvironment.go) - Relies on API `/join` endpoint
- Kosli API Documentation - Logical environment constraints
