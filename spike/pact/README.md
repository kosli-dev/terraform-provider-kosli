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

## Step 2: Plugin framework integration check

**What was built:** Nothing — analysis only.

**What was observed:**
- Three options investigated:
  - (A) Parallel suite in separate package — pact tests import `pkg/client`, run via `go test ./spike/pact/...`, existing tests untouched
  - (B) Embed in Terraform `resource.Test` framework — **not viable**. The framework controls the provider lifecycle and HTTP client internally; no hook to inject Pact's mock server.
  - (C) Replace `httptest` mocks in `pkg/client/` with Pact mocks — technically possible but creates hard dependency on native FFI lib for all client unit tests
- `custom_attestation_types.go` uses `multipart/form-data` for create, not JSON — potential Pact matcher challenge for Step 4
- Existing unit tests use `httptest` features (request body inspection, call counting) that Pact doesn't replicate directly

**Dan's checkpoint decision:** Option A — parallel suite. Pact tests as an additive layer alongside existing tests.

**Open "what about" questions:**
- How does multipart/form-data work with Pact matchers? (relevant for custom_attestation_type create in Step 4)

## Step 3a: Consumer test for GetEnvironment (consumer side)

**What was built:**
- Consumer test in `spike/pact/environment_pact_test.go` — exercises `client.GetEnvironment()` against Pact mock server, generates pact file
- Pact file updated with real Kosli interaction: `GET /api/v2/environments/{org}/{name}`

**What was observed:**
- Key Pact discipline: the contract should only include fields the consumer actually reads. Initial version included `state`, `policies`, `require_provenance`, `org` — none of which `data_source_environment.go` uses. Trimmed to 7 fields: `name`, `type`, `description`, `include_scaling`, `last_modified_at`, `last_reported_at`, `tags`.
- Pact response matching is liberal (Postel's Law) — extra fields from the provider are allowed and ignored during verification. This means trimming doesn't lose safety, it gains precision.
- `matchers.EachLike(..., 0)` is not allowed — Pact forces min 1 element in array matchers. Would be a problem for fields like `policies` that can be empty arrays.
- Path matching with regex works: `matchers.Term(example, regex)` produces both an example for the mock and a regex for verification.
- Wiring up `pkg/client` to Pact's mock server was straightforward: `client.WithBaseURL(fmt.Sprintf("http://%s:%d", config.Host, config.Port))`

**Open "what about" questions:**
- How to express "this field can be null OR a number" (e.g., `last_reported_at`)? Current contract says it's always a number.
- The hello-world interaction from Step 1 accumulates in the same pact file. Need separate pact files or cleanup strategy?

## Step 3b/3c: Provider verification against stub (provider side)

**What was built:**
- Provider verification test in `spike/pact/verify_test.go` — replays pact file against a local stub server and checks responses match the contract
- Local stub HTTP server (`stubKosliAPI()`) mimicking Kosli API responses
- State handlers for both interactions (no-ops since stub is static)

**What was observed:**
- Verification passed with different response values than consumer test examples (e.g., different timestamps) — confirms type-based matching works as expected
- Extra fields in stub response (`org`, `state`, `require_provenance`, `policies`) were ignored by Pact — Postel's Law working correctly
- State handler called with `setup=true` before and `setup=false` (teardown) after each interaction
- Verification output is human-readable: lists each interaction, checks status, headers, body, with OK/FAIL per check
- Pact sends anonymous usage tracking by default (`PACT_DO_NOT_TRACK=true` to disable)

**Dan's checkpoint decision:** Option 2 (local stub) for verification — keeps spike self-contained, makes deliberate breakage easy to test

## Step 3d: Deliberate breakage — failure message quality (provider side)

**What was built:** Nothing permanent — three temporary mutations to the stub server, each reverted.

**What was observed — three failure scenarios tested:**

1. **Wrong type** (`"type": 42` instead of `"K8S"`):
   - Message: `$.type -> Expected 42 (Integer) to be the same type as 'K8S' (String)`
   - Verdict: Clear, actionable. Points to exact field and shows both actual and expected types.

2. **Missing field** (`description` removed from response):
   - Message: `$ -> Actual map is missing the following keys: description`
   - Verdict: Clear, actionable. Names the missing field.

3. **Wrong status code** (404 instead of 200):
   - Message: `expected 200 but was 404`
   - Verdict: Clear, actionable.

**Failure message quality assessment:** All three failures were immediately understandable. The JSONPath notation (`$.type`, `$`) pinpoints the location. No stack traces or internal framework noise — just the mismatch.

## Step 4: Custom attestation type CRUD lifecycle

### Step 4a-4e: Consumer tests + provider verification (consumer + provider side)

**What was built:**
- Consumer tests in `spike/pact/custom_attestation_type_pact_test.go` for create (POST), read (GET), delete/archive (PUT)
- Updated stub server and state handlers in `spike/pact/verify_test.go` for all three interactions
- Update (POST creating new version) skipped as a separate test — uses same endpoint/shape as create

**Authoring time per interaction:**
- Create: ~10 min (multipart issue discovery + workaround)
- Read: ~5 min (pattern established)
- Delete: ~3 min (simplest — no request body, no response body to match)
- Verification stub additions: ~5 min

**What was observed:**
- **Multipart/form-data limitation:** `CreateCustomAttestationType` uses `multipart/form-data`, which Pact V2 can't match on the request body. Consumer test only verifies method, path, and response status/headers. The request body shape is not captured in the contract.
- **Client bypass of doRequest():** `CreateCustomAttestationType` builds its own `http.Request` and calls `c.httpClient.Do()` directly, bypassing retry, auth header, and error handling in `doRequest()`. Should be refactored independently of Pact. Tracked in kosli-dev/terraform-provider-kosli#198.
- **JSON string response body:** The API returns `"OK"` (a bare JSON string) for create/archive. Pact's `JSONBody("OK")` base64-encoded this during verification (`T0s=`), causing a mismatch. Workaround: skip body matching for these responses since the consumer doesn't read them.
- **Nested response structure:** The read response has a `versions` array with nested `type_schema` and `evaluator`. `matchers.EachLike` with nested `matchers.Like` handled this cleanly. The contract reflects the API shape, not the client's transformed shape.
- **State handler count growing:** 6 state handlers now (2 from Step 1/3, 4 new). With a real API these would each need setup/teardown logic. Provider state code volume is directly proportional to interaction count.

**Open "what about" questions:**
- Multipart/form-data means ~50% of CRUD interactions for this resource have no request body contract. Is that acceptable?
- `JSONBody` with string literals causes base64 encoding issues. Is this a pact-go bug or expected behavior for non-object bodies?
- Provider state handler count scales linearly with interactions. For full provider coverage (~30+ endpoints), that's significant plumbing.

### Step 4f: Generalization sniff-test

**The contract boundary with SDKs:**

If each language has its own Kosli Client SDK, the pact sits between the SDK and the API — not between the end consumer and the API. The SDK is where HTTP happens.

```
Terraform Provider ──┐
                     ├─→ Go SDK (pkg/client) ──→ Kosli API
CLI ─────────────────┘

Backstage plugin ────┐
                     ├─→ TypeScript SDK ────────→ Kosli API
MCP server ──────────┘
```

Each SDK is a Pact consumer. Provider state handlers also live at the SDK level — the SDK pact tests set up and tear down API state, not the end consumers.

| Pact consumer | Pact provider | Pact file |
|--------------|---------------|-----------|
| `KosliGoSDK` | `KosliAPI` | `KosliGoSDK-KosliAPI.json` |
| `KosliTypeScriptSDK` | `KosliAPI` | `KosliTypeScriptSDK-KosliAPI.json` |

**What reuses across consumers, and what doesn't:**

| Artifact | Shared? | Notes |
|----------|---------|-------|
| Pact file format | Yes | JSON, language-neutral |
| Provider name (`KosliAPI`) | Yes | All SDKs talk to same provider |
| Provider verification infra | Yes | Runs once per SDK pact, not per end consumer |
| Provider state handlers | Per SDK | SDK pact tests own the state setup, not end consumers |
| SDK pact test code | Per SDK | Go SDK writes Go tests, TS SDK writes TS tests |
| Native Pact library dependency | Per SDK | pact-go, pact-js, etc. each have their own FFI story |

**Amortization picture:**
- The Go SDK pact covers *all* Go consumers (Terraform provider + CLI) with one contract
- The TypeScript SDK pact covers *all* TS consumers (Backstage + MCP server) with one contract
- 2 pact files instead of 4+ (one per SDK, not one per end consumer)
- Provider verification runs twice, not four+ times
- The SDK pact is comprehensive — covers every API method the SDK exposes, since any consumer might use any of them
- End consumers don't write pact tests. They trust the SDK.
