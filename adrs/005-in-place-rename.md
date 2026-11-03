---
title: "ADR 005: In-Place Rename of Flows and Environments"
description: "Treating a change to the name of a kosli_flow, kosli_environment or kosli_logical_environment as an in-place rename through the API's rename endpoints instead of a destroy-and-create replacement."
status: "Accepted"
date: "2026-11-03"
---

# ADR 005: In-Place Rename of Flows and Environments

## Context

Until this decision, the `name` attribute of `kosli_flow`, `kosli_environment` and `kosli_logical_environment` carried the `RequiresReplace` plan modifier. Changing a name therefore planned as a replacement: `Delete` archived the resource under the old name and `Create` made a new one under the new name. The API client had no rename method, and the behaviour was never recorded as a deliberate choice.

The Kosli API has dedicated rename endpoints:

- `PUT /api/v2/flows/{org}/{name}/rename`
- `PUT /api/v2/environments/{org}/{name}/rename`

Both take `{"new_name": "..."}` and keep the resource's identity and history. Logical environments are environments to the API, so the environment endpoint renames them as well. The server also guarantees the alias behaviour documented on `kosli rename flow` and `kosli rename environment` (kosli-dev/cli#1322): after a rename, the old name still resolves to the renamed resource for reads and writes, so `kosli attest`, `kosli begin trail` and `kosli snapshot` using the old name land in the renamed resource. The old name stops resolving only once another resource is created with, or renamed to, that name. The only operation that requires the current name is a further rename.

Simulating a rename with archive-and-create loses all of that (issue #296):

- **Pipelines using the old name break.** After the archive, the old name belongs to an archived resource. The server refuses to auto-create a flow whose name has history, so pipelines that still use the old name get a `400` instead of landing in the renamed flow.
- **History is severed.** The new resource starts empty; trails and snapshots stay on the archived one.
- **Archived shells accumulate.** Nothing unarchives the old resource, so every rename leaves one behind.
- **Replace ordering adds a race.** Destroy-before-create leaves a window with no resource, and the provider already needs a post-create retry to cope with a related race (issue #121).

## Decision Drivers

1. **Reflect API semantics** - ADR 002 establishes the client as a thin wrapper that reflects what the API can do; the API can rename
2. **Preserve history and integrations** - A rename must not break pipelines or lose trails and snapshots
3. **Terraform conventions** - Treating a name as immutable identity is common, but only because most APIs cannot rename
4. **Failure handling** - A rename followed by further attribute updates must leave state consistent if one step fails

## Options Considered

### Option A: Keep `RequiresReplace` (status quo)

Keep the destroy-and-create behaviour and document it.

**Pros:**
- Matches the common Terraform convention that a name is identity
- No new client methods

**Cons:**
- All the problems listed in the context: broken pipelines, severed history, archived shells, replace race
- Hides a capability the API and CLI already expose

### Option B: In-place rename (Selected)

Remove `RequiresReplace` from `name`, add `RenameFlow` and `RenameEnvironment` to the client, and have `Update` call the rename endpoint first when the planned name differs from the prior state.

**Pros:**
- Keeps identity, history and the old-name alias exactly as the API guarantees
- No archived shells, no replace window
- Same semantics as `kosli rename flow` and `kosli rename environment`

**Cons:**
- Diverges from the "name is identity" convention; documented on the attribute
- `Update` has two API calls that must be ordered correctly

## Decision

**Selected: Option B (In-place rename)**

### Rationale

1. **The API supports it, so the provider should expose it.** ADR 002 asks the client to reflect API behaviour; refusing to rename when the API can is an artificial limitation.
2. **The alias behaviour is a server guarantee**, not something the provider has to emulate. The provider only needs to call the endpoint and document the guarantee.
3. **The convention exists for APIs that cannot rename.** Where an API can, an in-place update is the more faithful representation of what the user asked for.

## Consequences

### Positive

- Renaming a flow, environment or logical environment keeps its trails, snapshots, included environments, tags and identity
- Pipelines that still use the old name keep working until that name is taken by another resource
- No archived shells are left behind, and the destroy-before-create race (issue #121) cannot occur for a rename

### Negative

- A plan that changes `name` shows an update, not a replacement; users who expect the Terraform convention must read the attribute description
- A `name` change combined with a `type` change on `kosli_environment` still plans as a replacement, because `type` is immutable in the API

### Neutral

- The resource identity in state is still the name. After a rename, `Read` and `import` use the new name; the old name also resolves on the server, so a `Read` by a stale name refreshes state to the current name

## Implementation

### Client

`RenameFlow(ctx, name, newName)` and `RenameEnvironment(ctx, name, newName)` are thin wrappers over the rename endpoints (`pkg/client/flows.go`, `pkg/client/environments.go`). `kosli_logical_environment` uses `RenameEnvironment`.

### Update ordering

In `Update`, when the planned name differs from the prior-state name:

1. Call rename with the **prior-state** name. The server requires the current name, and for flows the subsequent `PUT` keyed on the new name would otherwise create a second flow instead of updating this one.
2. Persist the new name to state immediately, so a failure in the remaining update does not leave state pointing at the old name while the server already uses the new one.
3. Apply the remaining attribute updates against the new name.

If the rename itself fails (for example because the new name is taken), `Update` returns before any other call and the framework keeps the prior state, so the next plan shows the same rename again.

### Schema and documentation

`name` no longer has a plan modifier on any of the three resources. Its description and a note on each resource state that a change is an in-place rename and that the old name keeps resolving until another resource is created with, or renamed to, that name.

### Testing

- Client unit tests cover method, path, body, name escaping, and error mapping for both rename methods
- Provider unit tests for all three resources drive `Update` against a recording HTTP server and assert the call order (rename, then update, then read by the new name), the no-rename path, and state after a failed rename and after a failure following a successful rename
- Acceptance tests for all three resources assert that a name change plans as `Update`, preserves description, tags and included environments, leaves the old name resolving through the data source, works combined with other changes, and imports by the new name; a `type` change on `kosli_environment` still plans as `Replace`

## Related Decisions

- **ADR 002**: API Client Architecture - thin wrapper that reflects API behaviour
- **Issue #296**: Rename flows and environments in place instead of destroy-and-recreate
- **Issue #121**: Post-create retry for the label-rename race

## References

- [kosli rename flow](https://github.com/kosli-dev/cli/blob/main/cmd/kosli/renameFlow.go) and [kosli rename environment](https://github.com/kosli-dev/cli/blob/main/cmd/kosli/renameEnvironment.go) - CLI commands using the same endpoints
- [kosli-dev/cli#1322](https://github.com/kosli-dev/cli/pull/1322) - documents how the old name behaves after a rename
