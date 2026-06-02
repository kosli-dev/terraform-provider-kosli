---
name: changelog-creator
description: Generate one CHANGELOG.md entry for this Terraform provider, following HashiCorp's Terraform plugin changelog conventions. Use when preparing a release entry from the commits between two tags.
---

# Terraform Provider Changelog Entry

> **Consumed by CI.** This file is loaded as the governing prompt by
> `.github/workflows/release.yaml` (during a release tag push) and by
> `scripts/changelog-ai.sh` (the local dry-run helper). Edits here change
> what those tools produce. Keep that in mind before broadening scope or
> changing format rules.

Generate a single CHANGELOG.md entry for this Terraform provider. The output is a Markdown fragment intended to be inserted at the top of `CHANGELOG.md`, immediately under the `# Changelog` title.

The canonical reference is HashiCorp's spec:
https://developer.hashicorp.com/terraform/plugin/best-practices/versioning#changelog-specification

## Audience

The CHANGELOG is read by end users running `terraform plan` / `terraform apply`. It is **not** a development log for contributors. If a commit does not change something a Terraform user can observe, it does not appear in the entry.

## Scope filter

INCLUDE commits that affect:
- Resources or data sources (new, removed, renamed, schema changes, attribute changes)
- Provider configuration (auth, endpoints, environment variables, defaults)
- Runtime behavior at plan/apply time (drift, retries, error messages, validation, performance characteristics a user would notice)
- User-facing docs or examples under `docs/` or `examples/`
- Dependency bumps that change minimum Terraform or Go versions, or otherwise change visible behavior

EXCLUDE commits that only touch:
- GitHub Actions workflows or CI configuration (anything under `.github/`, `ci.yaml`, `release.yaml`, `pr-quality.yaml`)
- Repo tooling, scripts, or contributor docs (`scripts/`, `CLAUDE.md`, README development sections, `adrs/`)
- Internal-only tests or test infrastructure (`*_test.go` that does not reflect a user-observable fix)
- Dependency bumps with no behavior change, formatting, lint configuration

If, after filtering, no commits remain, emit a minimal entry of exactly this shape:

```
## VERSION (DATE)

NOTES:

* No user-facing changes in this release.
```

## Format

### Version header

`## VERSION (DATE)` where `DATE` is formatted as `Month D, YYYY` with no zero-padded day. Examples: `May 11, 2026`, `June 1, 2026` (not `June 01, 2026`).

### Category headings

Use these category names, in this order when present:

1. `BREAKING CHANGES`
2. `NOTES`
3. `FEATURES`
4. `IMPROVEMENTS`
5. `BUG FIXES`

Each heading is left-aligned, in all caps, with a trailing colon and a blank line above and below.

### Entry line

```
* <subsystem>: <user-focused description> [GH-####]
```

- `<subsystem>` is one of:
  - `resource/<name>:` (e.g. `resource/kosli_environment:`)
  - `data_source/<name>:` (underscore, matching this repo's history; note this diverges from HashiCorp's hyphenated `data-source/` style, but is consistent across all prior entries here)
  - `provider:` (cross-cutting provider behavior)
  - `client:` (the underlying Kosli API client behavior, when user-visible)
  - `docs:` (user-facing documentation or examples)
- Do not use generic prefixes such as `ci:`, `chore:`, `internal/...:`, `test:`, or `refactor:`. If a commit would only carry one of these prefixes, it is almost always filtered out by the scope rules above.
- `<user-focused description>` is one sentence, starts with a capital, no trailing period required (match prior entries), and describes the user-visible outcome rather than the implementation. Prefer "Fixed X" / "Added Y" / "Improved Z" phrasing.
- `[GH-####]` references the merging PR or the resolved issue number when known. Omit the bracket entirely if no PR or issue is associated.

### Ordering within a category

1. `provider:` entries first (cross-cutting changes lead).
2. All other entries in lexicographic order by subsystem prefix.

## Style

- One entry per change. Do not combine multiple changes into a single bullet.
- Past tense for completed work ("Added", "Fixed", "Improved").
- Reference attribute and resource names in backticks.
- Do not include implementation jargon (struct names, function names, internal package paths) unless it is part of the public surface a Terraform user interacts with.
- No emojis. No section icons. No marketing language.
- Match the tone and density of the existing entries in `CHANGELOG.md`.

## Output contract

Emit only the new entry. No preamble, no commentary, no questions, no surrounding code fences. Treat the input data (commits, previous CHANGELOG tail, target version, target date) as authoritative; do not ask for clarification.
