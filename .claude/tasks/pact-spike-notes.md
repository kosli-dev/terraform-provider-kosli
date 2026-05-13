# Pact Spike — Running Notes

## Step 0: Orient

**What was built:** Nothing — orientation only.

**What was observed:**
- Clean two-layer architecture: `pkg/client/` (HTTP client) and `internal/provider/` (Terraform layer)
- All HTTP calls go through `client.doRequest()` which adds auth headers and checks status codes
- Acceptance tests use `terraform-plugin-testing` framework with `resource.Test` — completely separate concern from API contract testing
- Unit tests in `pkg/client/` already use `httptest.NewServer()` to mock HTTP — Pact would replace that mock with a contract-generating mock
- Auth: Bearer token via `KOSLI_API_TOKEN` env var
- Pact's natural target is `pkg/client` layer, not the Terraform provider layer

**Dan's checkpoint decision:** Confirmed orientation is accurate. Agreed `pkg/client` is the right layer for Pact. Agreed `spike/pact/` is the throwaway code location.

**Open questions:** (none yet)

## Step 1: Hello-world pact-go

**What was built:**
- Added `pact-go v2.4.3` to `go.mod`
- Single hello-world consumer test in `spike/pact/hello_pact_test.go` — one interaction (GET /hello → 200 JSON)
- Pact file generated at `spike/pact/pacts/TerraformProviderKosli-KosliAPI.json`
- Matching rules docs: https://docs.pact.io/getting_started/matching

**What was observed:**
- pact-go v2 is a Go wrapper around a Rust FFI library (`libpact_ffi.dylib`, 13MB native binary). Not pure Go.
- `pact-go install` requires write access to `/usr/local/lib` (needed `sudo` on macOS)
- Even after install, `DYLD_LIBRARY_PATH=/usr/local/lib` is required at runtime — macOS `dyld` doesn't search `/usr/local/lib` by default (post-Big Sur security change). Linux would need equivalent `LD_LIBRARY_PATH`.
- The pact-go v2 API is fluent/builder-style (not error-returning) — minor surprise if you're used to Go conventions
- Cosmetic logger warning on every run (`can't set logger`) — harmless but noisy
- The generated pact file is readable JSON: consumer/provider names, interactions with request/response, matchingRules, metadata

**Open "what about" questions:**
- CI setup: every runner needs the FFI library installed + `DYLD_LIBRARY_PATH` (or `LD_LIBRARY_PATH`). What does this cost in CI config maintenance?
- Developer onboarding: new contributors need `sudo pact-go install` before tests work. How does that sit with the project's current zero-native-deps Go toolchain?
