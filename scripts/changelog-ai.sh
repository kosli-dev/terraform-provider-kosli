#!/bin/bash

# Local dry-run of the CI changelog generator. NOT used by any workflow.
# Keep the prompt and input shape below aligned with
# .github/workflows/release.yaml so the local output mirrors what CI
# produces on a real release tag.
#
# Set your env vars
#export ANTHROPIC_API_KEY=sk-...
#export GITHUB_REF_NAME=v0.0.0

if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo "ERROR: Set ANTHROPIC_API_KEY " >&2
    exit 1
fi

VERSION="${GITHUB_REF_NAME}"
DATE=$(date +"%B %d, %Y" | sed 's/ 0/ /')  # "Month D, YYYY" to match prior entries

PREV=$(git tag --sort=-version:refname \
  | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
  | grep -v "^${VERSION}$" \
  | head -n 1)
PREV="${PREV:-$(git rev-list --max-parents=0 HEAD)}"

COMMITS=$(git log "${PREV}..HEAD" --no-merges --pretty=format:"%s|%b")
EXISTING=$(tail -n 100 CHANGELOG.md 2>/dev/null || echo "")
SKILL=$(cat .github/changelog-skill.md)

ENTRY=$(curl -s -f --max-time 60 \
  -X POST https://api.anthropic.com/v1/messages \
  -H "x-api-key: ${ANTHROPIC_API_KEY}" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d "$(jq -n \
    --arg system "$SKILL" \
    --arg commits "$COMMITS" \
    --arg existing "$EXISTING" \
    --arg version "$VERSION" \
    --arg date "$DATE" \
    '{
      model: "claude-sonnet-4-6",
      max_tokens: 2048,
      system: $system,
      messages: [{
        role: "user",
        content: (
          "You are generating one new entry for this project'"'"'s CHANGELOG.md.\n\n" +
          "This is a Terraform provider. The CHANGELOG is read by end users running " +
          "`terraform plan`/`apply`, NOT by contributors to this repo. Only include " +
          "commits that change what those users observe.\n\n" +
          "INCLUDE commits that affect:\n" +
          "- Resources or data sources (new, removed, renamed, schema changes, attribute changes)\n" +
          "- Provider configuration (auth, endpoints, environment variables, defaults)\n" +
          "- Runtime behavior at plan/apply time (drift, retries, error messages, validation)\n" +
          "- User-facing docs or examples under docs/ or examples/\n" +
          "- Dependency bumps that change minimum Terraform/Go versions or visible behavior\n\n" +
          "EXCLUDE commits that only touch:\n" +
          "- GitHub Actions workflows or CI configuration (.github/, ci.yaml, release.yaml, pr-quality.yaml)\n" +
          "- Repo tooling, scripts, or contributor docs (scripts/, CLAUDE.md, README dev sections, adrs/)\n" +
          "- Internal-only tests or test infrastructure (anything *_test.go that does not reflect a user-observable fix)\n" +
          "- Dependency bumps with no behavior change, formatting, lint config\n\n" +
          "If after filtering NO commits remain, output a minimal entry of the form:\n" +
          "  ## " + $version + " (" + $date + ")\n" +
          "  NOTES:\n" +
          "  * No user-facing changes in this release.\n\n" +
          "Formatting rules (HashiCorp Terraform changelog conventions):\n" +
          "- Each entry is \"* <subsystem>: <message> [GH-####]\" where GH-#### is the PR or issue number (omit if not known).\n" +
          "- Subsystem prefixes to use: \"resource/<name>:\", \"data_source/<name>:\" (underscore, matching this repo'"'"'s history), \"provider:\", \"client:\", \"docs:\". Do NOT use generic prefixes like \"ci:\", \"chore:\", or \"internal/...:\".\n" +
          "- Within each category, list \"provider:\" (cross-cutting) entries first, then order the rest lexicographically by subsystem.\n" +
          "- Categories, in this order when present: BREAKING CHANGES, NOTES, FEATURES, IMPROVEMENTS, BUG FIXES.\n\n" +
          "The system prompt provides governing instructions for style, structure, and " +
          "tone; follow them strictly, BUT the scope rules above take precedence.\n\n" +
          "Existing CHANGELOG.md (tail, for style reference only, not content):\n" + $existing + "\n\n" +
          "New commits since the previous tag (format: \"subject|body\" per line):\n" + $commits + "\n\n" +
          "Generate the changelog section for " + $version + " released on " + $date + ".\n" +
          "Output ONLY the new changelog entry, with no preamble, no commentary, and no questions.\n" +
          "Assume this is always the latest entry."
        )
      }]
    }'
  )" | jq -r '.content[0].text')

# Just print — don't touch the file or commit
echo "$ENTRY"

HEADER_LINE=$(grep -n '^# ' CHANGELOG.md 2>/dev/null | head -1 | cut -d: -f1)
if [ -n "$HEADER_LINE" ]; then
    head -n "$HEADER_LINE" CHANGELOG.md > cl.md
    printf '\n%s\n' "$ENTRY" >> cl.md
    tail -n "+$((HEADER_LINE + 1))" CHANGELOG.md >> cl.md
else
    printf '%s\n\n' "$ENTRY" > cl.md
    cat CHANGELOG.md >> cl.md 2>/dev/null || true
fi

echo "updated CL:"
echo "--"
cat cl.md
rm -rf cl.md
