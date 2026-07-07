#!/bin/bash

# Local dry-run of the CI changelog generator. NOT used by any workflow.
# Uses the same skill spec at .claude/skills/changelog-creator/SKILL.md
# that changelog.yaml loads, so the local output mirrors what CI produces
# on a real release tag.
#
# Set your env vars
#export ANTHROPIC_API_KEY=sk-...
#export GITHUB_REF_NAME=v0.0.0

if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo "ERROR: Set ANTHROPIC_API_KEY " >&2
    exit 1
fi

VERSION="${GITHUB_REF_NAME}"
# Use the tag's own date (matches CI); if the tag doesn't exist yet — the
# usual dry-run case — fall back to today. "Month D, YYYY" to match prior entries.
DATE=$(git log -1 --format=%ad --date=format:'%B %e, %Y' "${VERSION}" 2>/dev/null | sed 's/  / /')
DATE="${DATE:-$(date +"%B %d, %Y" | sed 's/ 0/ /')}"

# Range policy (matches CI): a stable release consolidates everything since
# the last stable tag; a pre-release is incremental since the previous tag
# of any kind.
if echo "$VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  TAG_FILTER='^v[0-9]+\.[0-9]+\.[0-9]+$'
else
  TAG_FILTER='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'
fi
PREV=$(git tag --sort=-version:refname \
  | grep -E "$TAG_FILTER" \
  | grep -v "^${VERSION}$" \
  | head -n 1)
PREV="${PREV:-$(git rev-list --max-parents=0 HEAD)}"

COMMITS=$(git log "${PREV}..HEAD" --no-merges --pretty=format:"%s|%b")
EXISTING=$(tail -n 100 CHANGELOG.md 2>/dev/null || echo "")
SKILL=$(cat .claude/skills/changelog-creator/SKILL.md)

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
      model: "claude-sonnet-5",
      max_tokens: 2048,
      system: $system,
      messages: [{
        role: "user",
        content: (
          "Generate one CHANGELOG.md entry for this Terraform provider.\n\n" +
          "Follow the system prompt strictly; it is the authoritative spec for " +
          "scope, format, ordering, and style.\n\n" +
          "Existing CHANGELOG.md (tail, style reference only, not content):\n" + $existing + "\n\n" +
          "New commits since the previous tag (format: \"subject|body\" per line):\n" + $commits + "\n\n" +
          "Generate the changelog section for " + $version + " released on " + $date + "."
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
