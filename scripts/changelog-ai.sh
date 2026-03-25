#!/bin/bash

# Set your env vars
#export ANTHROPIC_API_KEY=sk-...
#export GITHUB_REF_NAME=v0.0.0

VERSION="${GITHUB_REF_NAME}"
DATE=$(date +%Y-%m-%d)

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
      max_tokens: 1024,
      system: $system,
      messages: [{
        role: "user",
        content: ("Existing CHANGELOG.md:\n" + $existing + "\n\nNew commits:\n" + $commits + "\n\nGenerate the changelog section for " + $version + " released on " + $date + " and just output the new changelog entry wihout any comments. Do NOT ask questions and assume this is always the latest entry!")
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
#mv /tmp/cl.md CHANGELOG.md
