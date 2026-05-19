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

**Note:** This "free orchestration" advantage is specific to the Terraform provider. An SDK consumer doesn't have Terraform's dependency graph — the state handler cost for dependent resources would be comparable whether using Pact or writing integration tests manually. This is one reason the spike's findings point toward Pact at the SDK level rather than the Terraform provider level.

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

The reusability bottleneck is not the pact files — it's the **per-SDK native library dependency**. pact-go wraps a 13MB Rust FFI binary (`libpact_ffi`). CI setup on Linux is straightforward (`wget` + copy to `/usr/local/lib`), though no official GitHub Action exists. Local development on macOS requires `sudo` to install and `DYLD_LIBRARY_PATH` at runtime. pact-js has its own native dependency story. Every developer and CI runner needs this installed.

## 7. The "what about" list

Collected verbatim from spike notes:

- CI setup: every Linux runner needs the FFI library installed (`wget` + copy to `/usr/local/lib`). No official GitHub Action exists. Local dev on macOS additionally requires `DYLD_LIBRARY_PATH`.
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

**Verdict: Wait.** Pact works mechanically, but the Terraform provider is not the right place to start. The provider doesn't have its own SDK yet — it uses `pkg/client` directly. Contracts are consumer-driven, and the consumer should be an SDK, not an end application. Investing in Pact consumer tests before the first SDK exists would mean rework when the SDK boundary is established.

What the spike showed:

- **Pact works mechanically.** Consumer tests generate contracts, verification checks them, failure messages are clear.
- **The authoring cost is moderate.** ~5-10 min per interaction once the pattern is established. The consumer test code is mechanical.
- **The value is narrow.** Pact catches integration drift — when the API changes response shapes without the SDK knowing. It doesn't test business logic, data persistence, or Terraform lifecycle behavior. The existing unit tests (httptest) and acceptance tests already cover those.
- **The infrastructure cost is real.** Native FFI dependency on every machine, no official GitHub Action for CI setup, `DYLD_LIBRARY_PATH` needed for local dev on macOS, multipart/form-data limitations, provider state handlers for real API verification.
- **The acceptance tests already handle dependency orchestration for free** via Terraform's dependency graph. Pact state handlers must manually replicate that work. (Note: this advantage is Terraform-specific — SDK consumers don't have a dependency graph, so this argument disappears at the SDK level.)

### Question B: Does the infrastructure investment plausibly amortize across other consumers?

**Verdict: Yes, but timing matters.** The SDK model makes amortization viable — 2 pact suites cover all consumers. But the trigger should be the first SDK, not the first consumer. Until then, the provider side should get familiar with Pact so verification can be stood up quickly when the first contract arrives.

What the spike showed:

- **With an SDK-per-language model, amortization improves.** 2 pact suites (Go SDK + TS SDK) cover all current and future consumers in those languages. Each new consumer (e.g., a future operator) adds zero pact cost if it uses an existing SDK.
- **The provider verification side is shared.** The API team writes state handlers once, all SDKs reference the same state strings.
- **The per-SDK cost is not shared.** Each SDK needs its own native Pact library, its own consumer tests, its own CI setup with FFI library installation.
- **Full contract safety requires real API verification**, which is the most expensive mode (state handler plumbing). Stub-only verification documents shapes but doesn't prove the real API matches.

### Decision

**Wait with Pact on the consumer side until the first SDK exists.** Contracts are consumer-driven, so the most reasonable provider-side work is to get familiar with Pact until the first contract has been created. When an SDK is built, that's the trigger to write consumer pact tests and stand up provider verification.

### Suggested next steps

1. **Agree on contract testing investment.** Align across teams on whether contract testing with Pact is the right approach going forward. This is a two-sided investment — both consumer and provider teams need to commit for the full value to materialize.

2. **Create SDKs with consumer contract tests.** SDK development will happen regardless of this spike, but we should have at least one stable SDK (e.g., consumed by `terraform-provider-kosli`) before investing in consumer-side contract testing. When the SDK boundary is established, add pact consumer tests to the SDK's CI.

3. **Investigate contract publishing.** Evaluate Pact Broker options: PactFlow (SaaS, zero infra), self-hosted (Docker + Postgres), or a simpler artifact-sharing approach. The broker enables the "can I deploy?" workflow and tracks verification results across teams.

4. **Invest in provider-side verification.** Stand up `pact verify` in the API's CI pipeline: state handlers, verification against real API, and a process for handling contract failures. This is where the contract testing loop closes.

### Key risks and open questions

**1. Will the API provider side actually pick up Pact?**

Contract testing is a two-sided investment. The consumer side (SDK pact tests) only generates pact files — JSON documents that describe expected behavior. Those files are inert unless the provider side runs verification against them. If the API team doesn't run verification in their CI pipeline, the contracts are never checked against reality, and the entire system provides no more safety than the stub-based verification we built in this spike.

This is the largest adoption risk. It requires:
- The API team agreeing to run `pact verify` in their CI
- State handlers written in the API's language/framework
- A process for handling verification failures (who fixes what, and when)

Without provider-side buy-in, Pact reduces to documentation with extra steps.

**2. Pact Broker: setup and buy vs. build**

The pact files need to get from the consumer CI to the provider CI somehow. The Pact Broker is the standard mechanism — it stores pact files, tracks verification results, and enables "can I deploy?" checks.

Options:
- **PactFlow (SaaS)** — hosted Pact Broker by the Pact team. Paid service, zero infrastructure. Simplest path.
- **Self-hosted Pact Broker** — open-source Docker image. Needs a database (Postgres), hosting, and maintenance.
- **No broker** — commit pact files to a shared repo or artifact store. Loses the verification tracking and "can I deploy?" workflow, but avoids the infrastructure. This is roughly what our spike did (pact files on disk).

The broker is not optional for the full Pact workflow across teams. Without it, there's no mechanism for the provider CI to know which pact files to verify, or for the consumer CI to know whether the provider has verified its latest contract.

**3. What do we gain, and what does the ROI look like?**

What Pact catches that nothing else does:
- **API removes or renames a field the SDK depends on.** Unit tests pass (they mock the old shape). Acceptance tests may not cover the specific field. Pact verification fails on the provider side before the change ships.
- **API changes a field's type** (e.g., string to number, or non-null to nullable). Same story — caught at contract level.
- **A new SDK consumer starts using an endpoint differently** than the existing consumers. The new pact file makes the dependency explicit and verifiable.

What Pact does not catch:
- Business logic bugs, data persistence issues, authorization problems, race conditions, or anything that requires the API to actually process data correctly.

ROI framing:
- **Cost:** ~2-3 hours authoring per SDK, native library dependency per SDK, Pact Broker infrastructure, API team buy-in and state handler maintenance, ongoing maintenance as the API evolves.
- **Benefit:** Early detection of breaking API changes across SDK boundaries. The value scales with: (a) how often the API changes response shapes, (b) how many SDKs consume the API, and (c) how painful a shape-mismatch bug is to debug in production vs. catching it in CI.

If the API is stable and changes infrequently, the ROI is low — the contracts rarely catch anything, but the maintenance cost persists. If the API is evolving rapidly with multiple SDK consumers, the ROI improves because the contracts catch drift that would otherwise surface as runtime bugs across multiple codebases.
