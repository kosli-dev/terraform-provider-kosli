# Pact Contract Testing Spike — Memo

## 1. What was built

Consumer-driven contract tests for two Kosli API resources using pact-go v2, targeting the `pkg/client` HTTP client layer:

- **Environment** (read): 1 interaction (GET), consumer test + provider verification
- **Custom attestation type** (create/read/delete): 3 interactions (POST, GET, PUT), consumer tests + provider verification
- **Provider verification** against a local stub server with state handlers

All code is in `spike/pact/`. Running notes are in `spike/pact/README.md`.

## 2. Plugin framework integration shape

Pact tests run as a **parallel suite** alongside existing tests. They import `pkg/client`, point it at Pact's mock server, and exercise client methods directly. The existing unit tests and acceptance tests are untouched.

Embedding Pact in the Terraform `resource.Test` framework is **not viable** — the framework controls the provider lifecycle and HTTP client internally, with no hook to inject a mock server.

## 3. Authoring cost

**Per-interaction authoring time (with pattern established):**

| Interaction type | Time |
|---|---|
| Read (GET, JSON response) | ~5 min |
| Create (POST, multipart — request body not matchable) | ~10 min |
| Delete (PUT, no body) | ~3 min |
| Verification stub per interaction | ~2 min |

**Projected full-provider cost:**

The provider currently has ~15 distinct API interactions across 7 resources. At ~5-10 min per interaction plus verification stubs, that's roughly **2-3 hours of authoring** for consumer tests. The consumer test code is mechanical once the pattern is established.

The larger cost is **provider state handlers** when verifying against a real API. Each interaction needs a state handler that sets up preconditions. For resources with dependencies (e.g., logical environment needs physical environments to exist first), state handlers must orchestrate multi-step setup and teardown — work that Terraform's dependency graph does automatically in acceptance tests. We estimate ~15 lines of state handler code per self-contained interaction, ~30+ for interactions with dependencies.

**Rough projection for other consumers (with SDK model):**

If each language has its own Kosli Client SDK, the pact sits between the SDK and the API:

```
Terraform Provider ──┐
                     ├─→ Go SDK (pkg/client) ──→ Kosli API
CLI ─────────────────┘

Backstage plugin ────┐
                     ├─→ TypeScript SDK ────────→ Kosli API
MCP server ──────────┘
```

Each SDK writes one set of pact tests covering all its API methods. End consumers (Terraform provider, CLI, etc.) don't write pact tests — they trust the SDK. This means **2 pact suites** (Go SDK + TypeScript SDK), not 4+ (one per end consumer).

## 4. Provider state experience

Pact's provider state mechanism conflates two concerns:

- **The contract itself** — "given state X, the response has shape Y." This is the value add.
- **The state setup** — "create this environment so it exists when Pact replays the GET." This is functional test infrastructure, not contract testing.

With a stub server, state handlers are no-ops — the stub always returns the right shapes. The contract verification passes because shapes match, and no state setup is needed.

With a real API, state handlers become real code: creating resources, managing dependencies, cleaning up in reverse order. This is the same work the acceptance tests get for free via Terraform's dependency graph. The more resource dependencies we have, the wider this cost gap becomes.

**The full Pact value — catching real API drift — only materializes when verifying against the real API.** That's the most expensive verification mode.

| Verification target | Contract safety | State handler cost |
|---|---|---|
| Consumer tests only (no verification) | Shapes documented, not verified | None |
| Stub server | Same — we trust the stub matches reality | Minimal (no-ops) |
| Real API | Full — the real API proves it matches the contract | High (functional setup/teardown per interaction) |

## 5. Failure message quality

Three failure scenarios tested (wrong type, missing field, wrong status code). All produced clear, actionable messages:

- `$.type -> Expected 42 (Integer) to be the same type as 'K8S' (String)`
- `$ -> Actual map is missing the following keys: description`
- `expected 200 but was 404`

JSONPath notation pinpoints the location. No stack traces or framework noise. If a contract breaks, the message tells us exactly what changed and where.

## 6. Cross-language relevance

The pact files are fully portable across languages. Matchers (`"match": "type"`, `"match": "regex"`) are Pact specification standard — pact-js, pact-python, pact-jvm all generate and consume the same format. Field names and shapes are language-neutral (snake_case from the Kosli API, standard JSON types).

Provider state strings are plain English, shareable across SDKs. In a real Pact setup, state handlers live on the **provider side** (the API team writes them once), and all SDK consumers reference the same state names.

The reusability bottleneck is not the pact files — it's the **per-SDK native library dependency**. pact-go wraps a 13MB Rust FFI binary (`libpact_ffi.dylib`) that requires `sudo` to install and `DYLD_LIBRARY_PATH` at runtime on macOS. pact-js has its own native dependency story. Every developer and CI runner needs this installed.

## 7. The "what about" list

Collected verbatim from spike notes:

- CI setup: every runner needs the FFI library installed + `DYLD_LIBRARY_PATH` (or `LD_LIBRARY_PATH`). What does this cost in CI config maintenance?
- Developer onboarding: new contributors need `sudo pact-go install` before pact tests work. How does that sit with the project's current zero-native-deps Go toolchain?
- How to express "this field can be null OR a number" (e.g., `last_reported_at`)? Pact V2 doesn't support union types; V3+ might.
- `matchers.EachLike(..., 0)` is not allowed — Pact forces min 1 element. Problem for fields like `policies` that can be empty arrays.
- Multipart/form-data (used by `CreateCustomAttestationType`) can't be matched on the request body in Pact V2. ~50% of CRUD interactions for that resource have no request body contract.
- `CreateCustomAttestationType` bypasses `doRequest()` and calls `c.httpClient.Do()` directly — should be refactored independently of Pact. Tracked in kosli-dev/terraform-provider-kosli#198.
- `JSONBody` with bare string literals causes base64 encoding issues during verification.
- Provider state handler count scales linearly with interactions. For full provider coverage (~30+ endpoints), that's significant plumbing.
- The hello-world interaction from Step 1 accumulates in the same pact file as real interactions. Need separate pact files or cleanup strategy.

## 8. Recommendation

### Question A: Did Pact feel like a fit for the Terraform provider specifically?

*Dan's call, based on the evidence above.*

What the spike showed:

- **Pact works mechanically.** Consumer tests generate contracts, verification checks them, failure messages are clear.
- **The authoring cost is moderate.** ~5-10 min per interaction once the pattern is established. The consumer test code is mechanical.
- **The value is narrow.** Pact catches integration drift — when the API changes response shapes without the SDK knowing. It doesn't test business logic, data persistence, or Terraform lifecycle behavior. The existing unit tests (httptest) and acceptance tests already cover those.
- **The infrastructure cost is real.** Native FFI dependency on every machine, `DYLD_LIBRARY_PATH` at runtime, multipart/form-data limitations, provider state handlers for real API verification.
- **The acceptance tests already handle dependency orchestration for free** via Terraform's dependency graph. Pact state handlers must manually replicate that work.

### Question B: Does the infrastructure investment plausibly amortize across other consumers?

*Dan's call, based on the evidence above.*

What the spike showed:

- **With an SDK-per-language model, amortization improves.** 2 pact suites (Go SDK + TS SDK) cover all current and future consumers in those languages. Each new consumer (e.g., a future operator) adds zero pact cost if it uses an existing SDK.
- **The provider verification side is shared.** The API team writes state handlers once, all SDKs reference the same state strings.
- **The per-SDK cost is not shared.** Each SDK needs its own native Pact library, its own consumer tests, its own CI setup with FFI library installation.
- **Full contract safety requires real API verification**, which is the most expensive mode (state handler plumbing). Stub-only verification documents shapes but doesn't prove the real API matches.
