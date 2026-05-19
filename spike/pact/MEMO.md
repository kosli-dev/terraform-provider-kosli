# Pact Contract Testing Spike -- Memo

## 1. What was built

Consumer-driven contract tests for two Kosli API resources using pact-go v2, targeting the `pkg/client` HTTP client layer:

- **Environment** (read): 1 interaction (GET), consumer test + provider verification
- **Custom attestation type** (create/read/delete): 3 interactions (POST, GET, PUT), consumer tests + provider verification
- **Provider verification** against a local stub server with state handlers

All code is in `spike/pact/`. Running notes are in `spike/pact/README.md`.

## 2. Integration shape

Pact tests run as a **parallel suite** alongside existing tests. They import `pkg/client`, point it at Pact's mock server, and exercise client methods directly. The existing unit tests and acceptance tests are untouched.

Embedding Pact in the Terraform `resource.Test` framework is **not viable** -- the framework controls the provider lifecycle and HTTP client internally, with no hook to inject a mock server.

### Three test layers

Pact doesn't replace any existing tests -- it fills a gap between them:

| Layer | Tests what | Catches | Speed |
|---|---|---|---|
| **Unit tests** | Client code against httptest mocks | Parsing bugs, error handling, edge cases | Milliseconds |
| **Contract tests** | HTTP interface between client and API | Field renames, type changes, missing fields, status code changes | Seconds |
| **Acceptance tests** | Full Terraform plan/apply/destroy against real API | Business logic, state management, dependency orchestration | Minutes |

### CI pipeline -- earliest feedback first

Contract tests slot in right after unit tests, catching integration drift before the slow e2e stage:

```
Push/PR -> Unit tests (~5s) -> Contract tests (~10s) -> E2E/acceptance (~15 min) -> Deploy
```

Contract tests need only the Pact FFI library -- no API credentials, no test org, no network.

## 3. Authoring cost

**Per-interaction authoring time (with pattern established):**

| Interaction type | Time |
|---|---|
| Read (GET, JSON response) | ~5 min |
| Create (POST, multipart -- request body not matchable) | ~10 min |
| Delete (PUT, no body) | ~3 min |
| Verification stub per interaction | ~2 min |

The provider has ~15 distinct API interactions across 7 resources. Projected full-provider authoring cost: **2-3 hours**. The consumer test code is mechanical once the pattern is established. With agent-supported development, authoring cost is largely automated -- making it negligible in the overall investment.

**Not all interactions are equal.** Create interactions (POST with multipart) have no request-body or response-body contract -- only method, path regex, status code, and Content-Type are verified. Read interactions (GET with JSON response) carry full field-level contracts. Roughly 30-40% of interactions in this provider are creates/deletes with thin contracts, which tempers the per-interaction amortization story.

The larger cost is **provider state handlers** when verifying against a real API. For resources with dependencies, state handlers must orchestrate multi-step setup and teardown.

### Without SDKs -- pact per consumer

Without a shared SDK, every consumer talks to the API directly and needs its own pact tests. The same API interactions get tested repeatedly across consumers -- violating DRY and multiplying maintenance.

### With SDKs -- pact per SDK

```
Terraform Provider --+
                     +--> Go SDK (pkg/client) --> Kosli API
CLI -----------------+

Backstage plugin ----+
                     +--> TypeScript SDK -------> Kosli API
MCP server ----------+
```

Each SDK writes one set of pact tests covering all its API methods. End consumers don't write pact tests -- they trust the SDK. This means **2 pact suites** (Go SDK + TypeScript SDK), not 4+ (one per end consumer).

## 4. Provider state experience

### What is provider state?

A contract says: "when I GET `/environments/org/production-k8s`, the API returns a JSON object with these fields." But that only works if the environment actually exists. **Provider state** is Pact's mechanism for expressing that precondition.

On the **consumer side**, the test declares a state as a plain-English string -- purely a label. The mock server doesn't enforce it.

On the **provider side**, during verification, Pact looks up that string in a map of state handlers -- functions that run before each interaction to make the precondition true (setup) and clean up after (teardown).

With a stub server (what we used in the spike), these handlers are no-ops. With a real API, they become real code that creates and tears down test data.

### Why this matters

Provider state conflates two concerns:

- **The contract itself** -- "given state X, the response has shape Y." This is the value add.
- **The state setup code** -- creating resources, managing dependencies, cleaning up. This is functional test infrastructure, not contract testing. It's also where the cost scales.

| Verification target | Contract safety | State handler cost |
|---|---|---|
| Consumer tests only (no verification) | Shapes documented, not verified | None |
| Stub server | Same -- we trust the stub matches reality | Minimal (no-ops) |
| Real API | Full -- the real API proves it matches the contract | High (functional setup/teardown per interaction) |

**Note:** The "free orchestration" advantage of acceptance tests is specific to the Terraform provider (dependency graph). SDK consumers don't have that, so this cost gap disappears at the SDK level.

## 5. Failure message quality

Three failure scenarios tested. All produced clear, actionable messages:

- `$.type -> Expected 42 (Integer) to be the same type as 'K8S' (String)`
- `$ -> Actual map is missing the following keys: description`
- `expected 200 but was 404`

JSONPath notation pinpoints the location. No stack traces or framework noise. If a contract breaks, the message tells us exactly what changed and where.

## 6. Cross-language relevance

### Languages in scope

Three languages are relevant for Kosli SDK consumers of the public API (`/api/v2/`):

- **Go** -- Terraform provider, CLI. Spike tested with [pact-go](https://github.com/pact-foundation/pact-go) v2.
- **TypeScript** -- Backstage plugin, MCP server. Would use [pact-js](https://github.com/pact-foundation/pact-js).
- **Python** -- Future consumers, scripting, automation. Would use [pact-python](https://github.com/pact-foundation/pact-python).

### What about the frontend?

The Kosli frontend is TypeScript/React but server-rendered via HTMX + Jinja2 templates. It communicates exclusively with **internal Flask UI routes** (returning HTML fragments), not the public `/api/v2/` REST API. Contract testing between front- and backend would target a different API surface and is a separate question from this spike.

### Portability of pact files

Pact files are fully portable across all three languages. Matchers (`"match": "type"`, `"match": "regex"`) are Pact specification standard. Field names are language-neutral snake_case from the API.

Provider state strings are plain English, shareable across SDKs. In production, state handlers live on the **provider side** -- the API team writes them once.

The reusability bottleneck is not the pact files -- it's that each Pact library requires a native binary installed on every machine. CI setup on Linux is straightforward (`wget` + copy to `/usr/local/lib`), though no official GitHub Action exists. Local development on macOS requires `sudo` to install and `DYLD_LIBRARY_PATH` at runtime.

## 7. Open questions

Issues surfaced during the spike that would need answers before production adoption:

1. **CI runner setup:** Every Linux runner needs the FFI library installed (`wget` + copy to `/usr/local/lib`). No official GitHub Action exists. Local dev on macOS additionally requires `DYLD_LIBRARY_PATH`.
2. **Onboarding friction:** `sudo pact-go install` required for local development. This adds a native dependency to what is currently a zero-native-deps Go toolchain.
3. **Nullable types:** How to express "null OR number" (e.g., `last_reported_at`)? Pact V2 doesn't support union types. [V3 adds a null matcher and OR combinator](https://github.com/pact-foundation/pact-specification/tree/version-3).
4. **Empty arrays:** `matchers.EachLike(..., 0)` not allowed -- Pact forces [min 1 element](https://github.com/pact-foundation/pact-go/blob/master/docs/consumer.md). Problem for fields like `policies` that can legitimately be empty.
5. **Multipart/form-data:** Can't be matched on request body in Pact V2. [V3+ adds multipart support](https://github.com/pact-foundation/pact-go/blob/master/docs/consumer.md). ~50% of CRUD interactions for custom attestation type have no request body contract as a result.
6. **Map-typed fields with dynamic keys:** `matchers.Like(map[string]string{"env": "prod"})` constrains `tags` to always contain a literal key `env`. Pact V2 type matching on maps matches keys literally and values by type -- there is no way to express "map of unknown keys to strings." If the real API returns empty tags or different key sets, provider verification fails. Same shape of problem as empty arrays (item 4).
7. **Client bypass:** `CreateCustomAttestationType` bypasses `doRequest()` and calls `httpClient.Do()` directly -- skipping retry and auth logic. Should be refactored independently of Pact. Tracked in kosli-dev/terraform-provider-kosli#198.
8. **State handler scaling:** Provider state handler count scales linearly with interactions. For full API coverage (~30+ endpoints), that's significant plumbing -- especially for resources with dependencies that need multi-step setup.

## 8. Recommendation

### Verdict

**Wait with Pact on the consumer side until the first SDK exists.** Contracts are consumer-driven, so the most reasonable provider-side work is to get familiar with Pact until the first contract has been created. When an SDK is built, that's the trigger to write consumer pact tests and stand up provider verification.

Pact works mechanically and its failure messages are excellent. The value is narrow (shape drift only) and the infrastructure cost is real -- but the **SDK model makes amortization viable**, and shifting integration checks out of e2e into contract tests would give us faster, more focused feedback.

### Shifting integration feedback left

Today we rely on long-running e2e tests to catch conflicts between the API and clients. Contract testing would give us that signal earlier -- in CI, on every SDK commit -- with confidence that a given SDK version satisfies the API contract. This frees e2e tests to focus on what they're actually good at: testing functional behavior end-to-end, rather than also serving as integration compatibility checks.

### Key risks

1. **Will the API team pick up Pact?** Contract testing is two-sided. Pact files are inert unless the provider runs verification in CI. Without provider-side buy-in, Pact reduces to documentation with extra steps.

2. **Pact Broker infrastructure.** Required for the full workflow. [PactFlow](https://pactflow.io/pricing/) free tier limited to 2 integrations; Team plan ~$1,385/yr. Self-hosted needs Postgres + Docker. Operational cost not yet evaluated.

### Suggested next steps

1. **Agree on contract testing investment.** Align across teams on whether contract testing with Pact is the right approach going forward. This is a two-sided investment -- both consumer and provider teams need to commit for the full value to materialize.

2. **Create SDKs with consumer contract tests.** SDK development will happen regardless of this spike, but we should have at least one stable SDK (e.g., consumed by `terraform-provider-kosli`) before investing in consumer-side contract testing. When the SDK boundary is established, add pact consumer tests to the SDK's CI.

3. **Investigate contract publishing.** Evaluate Pact Broker options: PactFlow (SaaS, zero infra), self-hosted (Docker + Postgres), or a simpler artifact-sharing approach. The broker enables the "can I deploy?" workflow and tracks verification results across teams.

4. **Invest in provider-side verification.** Stand up `pact verify` in the API's CI pipeline: state handlers, verification against real API, and a process for handling contract failures. This is where the contract testing loop closes.
