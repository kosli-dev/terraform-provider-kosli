# Pact spike — Claude Code edition

A learning-optimized spike to feel the shape of consumer-driven contract
testing using the Terraform provider as the test vehicle.

Claude Code does the typing. The human (Dan) supervises, learns, and
makes the decisions at every checkpoint.

The deliverable is a memo. The code is throwaway. The *learning* is
what makes the spike worth doing.

## Where this brief lives in the repo

This file should be placed in the Terraform provider repo
(`kosli-dev/terraform-provider-kosli`) at:

```
.claude/tasks/pact-spike.md
```

A pointer line should be added to the provider's root `CLAUDE.md` so
Claude Code finds it without being told:

```markdown
## Active workstreams
- Pact contract testing spike: see `.claude/tasks/pact-spike.md`
```

Related artifacts (created during the spike, not committed long-term):

- `.claude/tasks/pact-spike-notes.md` — the running notes file
  (see "The notes file" section below). Co-located with the brief so
  the spike materials stay together.
- `spike/pact/` — directory for the throwaway pact test code itself.
  Visually separated from production code under `internal/provider/`
  and `pkg/client/`, which makes the post-spike "delete or promote"
  decision easier. Claude Code should confirm this placement at
  step 0 rather than assume — the provider's `CONTRIBUTING.md` may
  suggest a different convention.

## How to run this brief

This brief is structured as a sequence of small steps. After each step,
Claude Code stops and surfaces what was built, what was observed, and
any open questions. Dan reviews, asks questions, and decides whether to
proceed, redo, or skip.

The default prompt to start each step with Claude Code:

> Work on step N from `.claude/tasks/pact-spike.md`. Take the smallest
> reasonable slice, stop, and show me what you built and what you
> observed before moving on.

Stop conditions Claude Code should honor:

- Before installing or pulling a new dependency, surface it and wait for approval
- Before adding code to the FastAPI app for provider state setup, stop and discuss the approach
- After any test runs (pass or fail), stop and show the output verbatim
- When hitting a "what about" question, write it down in the notes file
  and continue with the current task — don't try to answer it
- When something doesn't work, stop and surface it rather than working
  around it silently

If Claude Code is about to spend more than ~10 minutes on a single sub-step
without surfacing progress, that's a sign to stop and check in.

## Why we're spiking

(Same context as the regular spike brief — multi-consumer landscape,
infrastructure investment amortization question, etc. Read once at the
start, not repeated each step.)

The spike answers two questions:

A. Did Pact feel like a fit for the Terraform provider specifically?
B. Does the infrastructure investment plausibly amortize across the
   other consumers (CLI, MCP server, Backstage plugin, future operator)?

The Terraform provider is the spike vehicle because it's the
structurally most favorable consumer for Pact (bounded surface, CRUD
lifecycle, existing acceptance tests as starting material). If Pact
struggles here, it'll struggle elsewhere.

## Time-box

Two working days, hard ceiling, calendar time. The "Claude Code does
the typing" framing might tempt you to fit more in. Resist that. The
ceiling exists because the *learning* needs time to happen — Dan
needs to think between steps, not just approve them. A spike where
Dan approved everything without thinking has produced no learning,
regardless of what got built.

If past day two and step 4 (resource 2 CRUD lifecycle) isn't running
green, that's a signal against Pact. Stop, write the memo with what
you have.

## The notes file

Before step 1: Claude Code creates `.claude/tasks/pact-spike-notes.md`
(co-located with this brief). This file is updated after every step with:

- What was built
- What was observed (concrete, not editorial)
- Open "what about" questions
- Anything Dan flagged during the checkpoint conversation

This file is the raw material for the memo. It's appended to, not
rewritten.

---

## Step 0: Orient (≤30 min)

Claude Code reads the provider repo's `CLAUDE.md`, `CONTRIBUTING.md`,
`Makefile`, and the structure of `internal/provider/` and `pkg/client/`.
Surfaces back to Dan:

- Where the HTTP client code lives
- How acceptance tests are structured (`make testacc` entry point)
- How the existing tests authenticate against the API
- Any test environment setup the provider already requires
- Whether `spike/pact/` is the right home for the throwaway pact test
  code, or whether `CONTRIBUTING.md` suggests a different convention

**Checkpoint:** Dan reads the summary. Decides whether the orientation
is accurate before any code is written. If the structure is different
than expected, the rest of the brief needs adjustment.

**Learning goal:** Dan understands the provider's existing testing
shape well enough to predict where pact-go will fit (or fight).

---

## Step 1: Hello-world pact-go (≤1 hour)

Claude Code adds `pact-go` to the provider as a dependency, sets up
the minimum needed to write one Pact consumer test (not yet against
Kosli — just the canonical "hello world" pact-go example, or a
deliberately tiny test against any mock endpoint).

Goal: prove that pact-go installs, runs, and produces a pact file in
this repo before doing anything Kosli-specific.

**Checkpoint:** Dan looks at:
- What got added to `go.mod`
- The single test file
- The pact file pact-go produced (the actual JSON on disk)
- The test output

This is the first place to write a "what about" item if the install
itself was annoying or required system-level dependencies.

**Learning goal:** Dan has seen a pact file with their own eyes and
knows what one looks like before any complexity is layered on.

**Stop-and-discuss prompt for Claude Code:** "Before we move on, here's
the pact file. Read through it with me — what's worth understanding in
this JSON before we use it for real?"

---

## Step 2: Plugin framework integration check (≤1 hour)

Before doing anything Kosli-specific, answer the structural question:
how do pact-go tests coexist with the Terraform Plugin Framework's
testing patterns?

Claude Code investigates and proposes one of:

a. Pact tests as a parallel suite alongside acceptance tests (separate
   files, separate `go test` invocations, possibly separate package)
b. Pact tests embedded in the existing `resource.Test` framework via
   hooks or custom steps
c. Something else discovered along the way

Surfaces the trade-offs Dan needs to decide between.

**Checkpoint:** Dan picks the integration shape. This decision affects
everything downstream, so it's worth slowing down here. If neither
option feels clean, that itself is a memo observation.

**Learning goal:** Dan understands the structural relationship between
the two test frameworks and has made a deliberate choice rather than
inheriting whatever Claude Code picked.

---

## Step 3: Resource 1 — data source read (≤2 hours)

Build a pact consumer test for one read-only data source. Recommend
`data.kosli_environment` because it's the simplest.

Sub-steps Claude Code should pause after:

3a. Write the consumer test, generate the pact file
    → Stop. Dan reviews the pact file (second time seeing one, now
    with a real Kosli interaction in it)

3b. Set up a running Kosli API instance for verification (local docker?
    dev environment? Claude Code surfaces options)
    → Stop. Dan picks the verification target.

3c. Run provider verification
    → Stop. Dan reads the output regardless of pass/fail.

3d. *Deliberately break it* — Claude Code introduces a small mismatch
    (rename a field expectation, change a response shape) and re-runs
    verification
    → Stop. Dan reads the failure message and judges whether it's
    actionable.

**Checkpoint:** Dan now has direct experience with the full Pact loop
on a simple case. Notes file gets:
- Authoring time for this resource
- Pact file readability assessment
- Failure message quality assessment

**Learning goal:** Dan has seen the whole pipeline work end-to-end
on the easiest case. From here, every additional complexity is
visible against this baseline.

---

## Step 4: Resource 2 — full CRUD lifecycle (≤4 hours)

Build pact tests for `kosli_custom_attestation_type` across
create → read → update → delete. This is the substantive step where
provider state becomes the central question.

Sub-steps Claude Code should pause after:

4a. Write the consumer test for the *create* interaction
    → Stop. Dan reviews. Note authoring time. This is the data point
    for "how long does each interaction take to write."

4b. Discuss provider state setup *before* writing any FastAPI code.
    Claude Code surfaces:
    - Does this interaction need preexisting state?
    - If yes, what's the minimum state, and how do we set it up?
    - Are we modifying the FastAPI app? Adding test-only endpoints?
      Using existing API calls?
    → Stop. Dan makes the call on approach. This is the page-the-Pact-
    docs-warned-about moment.

4c. Implement the chosen provider state approach
    → Stop. Dan reviews what code got added where.

4d. Run verification for the create interaction
    → Stop. Output review.

4e. Repeat 4a-4d for read, update, delete — but Claude Code should
    move faster through these now that the pattern is established.
    Only stop on something genuinely new (e.g., a matcher question
    for `updated_at`, or a state teardown issue).

4f. *Sniff-test for generalization:* Claude Code shows Dan the provider
    state code and asks "would this pattern work for the CLI's flow
    setup needs, or the Backstage plugin's environment fixtures?"
    Five-minute conversation, captured in notes.

**Checkpoint:** This is the big one. Dan reviews the notes accumulated:
- Total authoring time across CRUD
- Provider state code volume
- Generalization sniff-test outcome
- Matcher decisions made
- Any "what about" items added

**Learning goal:** Dan has felt the substantial cost of Pact —
provider state plumbing, matcher discipline, lifecycle management —
on the most representative case the provider offers.

---

## Step 5: Resource 3 — dependent resource cost estimate (≤1 hour)

Claude Code does *not* try to make `kosli_logical_environment` work.
Instead:

5a. Lay out what would be needed to write the create interaction:
    - What resources need to exist as preconditions
    - How many provider states that translates to
    - Rough estimate of provider state code volume

5b. Compare against the resource 2 cost

5c. Make a judgment: is the cost roughly linear in dependency count,
    or does it grow worse?

**Checkpoint:** Dan reviews the estimate. Notes file gets the cost
projection.

**Learning goal:** Dan can extrapolate from the spike's measured costs
to predict full-provider coverage cost, and beyond.

---

## Step 6: Cross-language relevance check (≤30 min)

Claude Code looks at the pact files generated so far and asks: would
these be consumable by a TypeScript consumer (e.g., the Backstage
plugin or MCP server)?

This isn't building TypeScript tests. It's a 30-minute thought
experiment surfaced as concrete observations:

- Are any matchers Go-specific?
- Are field names and shapes language-neutral?
- Would the provider state machinery be reusable, or does it assume
  Go-specific test infrastructure?

**Checkpoint:** Dan reviews. Notes file gets the cross-language
assessment.

**Learning goal:** Dan can answer recommendation question B with
something more than a guess.

---

## Step 7: Write the memo (≤2 hours)

Claude Code drafts the memo from the notes file. Structure:

1. What was built (2-3 lines)
2. Plugin framework integration shape chosen (from step 2)
3. Authoring cost (concrete time per interaction, projected full-
   provider cost, rough projection for other consumers)
4. Provider state experience (from step 4)
5. Failure message quality (from step 3d)
6. Cross-language relevance (from step 6)
7. The "what about" list (verbatim from notes, not curated)
8. Recommendation:
   - *Question A:* Did Pact fit the Terraform provider?
   - *Question B:* Does it amortize across other consumers?

Dan edits the draft. The memo is Dan's, not Claude Code's — the draft
is just to save typing.

**Checkpoint:** Dan finalizes the memo. Decides next steps.

**Learning goal:** Dan has a concrete, defensible recommendation
grounded in observed evidence, not vibes.

---

## What Claude Code should NOT do

These are common Claude Code failure modes worth heading off:

- *Don't optimize for green tests.* If something fails, surface it.
  Don't silently work around it with mocks, skips, or matcher loosening.
- *Don't expand scope.* If a "what about" surfaces, it goes in the
  notes file. It doesn't become a new step.
- *Don't keep working after a checkpoint without explicit go-ahead.*
- *Don't make architectural decisions silently.* Provider state
  approach, matcher strategy, test framework integration shape — all
  surface for Dan to decide.
- *Don't claim the spike is "going well" or "going badly".* Report
  what was observed. Dan judges.
- *Don't pre-empt the recommendation.* The memo's recommendation is
  Dan's call, drafted from notes, not Claude Code's opinion.

## What success looks like

Not "all tests pass." Not "we have a working pact suite."

Success is: at the end of two days, Dan can say with concrete evidence
whether Pact is a fit for the multi-consumer landscape, and the memo
explains the reasoning to someone who wasn't there.

A "no" memo with concrete cost data is a success. A "yes" memo with
concrete cost data is a success. A pile of working tests with no
clear conclusion is a failure.

## Pre-mortem

The Claude-Code-does-the-typing framing has a specific failure mode:
Dan stops thinking because progress feels effortless. Approving each
step becomes mechanical. Checkpoints get skipped because the code
looks reasonable.

The protection against this: if Dan finds themselves approving steps
without questions or observations, that's the signal to stop and ask
"what am I actually learning right now?" If the answer is "not much,"
the spike has stopped earning its keep regardless of how much got
built.

The notes file is the artifact of learning. If it's thin at the end
of day one, the spike isn't working as intended. Slow down, ask more
questions at checkpoints, write more down.
