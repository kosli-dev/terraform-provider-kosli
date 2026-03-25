---
name: changelog-creator
description: Automatically creates user-facing changelogs from git commits by analyzing commit history, categorizing changes, and transforming technical commits into clear, customer-friendly release notes. Turns hours of manual changelog writing into minutes of automated generation.
---

# Changelog Creator

Transform technical git commits into compelling, user-focused changelog entries.

## Purpose

This skill helps you create changelog entries that:
- **Engage users** with clear, benefit-focused language
- **Span products** (API, UI, CLI, Terraform) with proper categorization
- **Support all change types** (features, fixes, security, improvements)
- **Work universally** (CHANGELOG.md files and Slack announcements)

The skill analyzes git commits, asks clarifying questions, and generates polished entries ready for publication.

## When to Use

- Creating a new release announcement
- Documenting changes since last version
- Preparing user-facing release notes
- Generating changelog entries for documentation sites

## How It Works

### 1. Analyze Git History

First, the skill examines recent commits to understand what changed:

**If arguments provided:**
- Commit SHA: Analyzes all commits since that specific commit
- Date: Analyzes commits since that date (e.g., "2024-01-01")
- Tag: Analyzes commits since that tag (e.g., "v1.2.3")

**If no arguments:**
- Looks for the most recent tag
- Analyzes all commits since that tag

```bash
# Example git analysis commands
git log --oneline --since="$DATE"
git log --oneline $COMMIT..HEAD
git log --oneline $(git describe --tags --abbrev=0)..HEAD
```

### 2. Categorize Changes

Group commits into user-facing categories:

- 🎉 **New Features** - New capabilities or functionality
- 🚀 **Improvements** - Enhancements to existing features
- 🐛 **Bug Fixes** - Issues resolved
- 🔒 **Security** - Security-related updates
- ⚠️ **Breaking Changes** - Changes requiring user action
- 📚 **Documentation** - Significant documentation updates
- 🏗️ **Internal** - Behind-the-scenes changes (usually excluded from user-facing changelog)

### 3. Ask Clarifying Questions

For each significant change, ask:

**About the change:**
- What user problem does this solve?
- What's the high-level outcome or benefit?
- Are there any caveats or limitations?

**About product scope:**
- Which products does this affect? (API, UI, CLI, Terraform, All)
- Is this breaking or backwards-compatible?

**About communication:**
- How should we describe this to users?
- Any specific examples or use cases to highlight?

### 4. Generate Entry

Create a changelog entry with this structure:

```markdown
## YYYY-MM-DD

### New Features

**[Product] Feature Title**
- Brief, benefit-focused description
- Key capability or outcome
- Example use case if relevant

### Improvements

**[Product] Enhancement Title**
- What improved and why it matters
- Performance gains, UX improvements, etc.

### Bug Fixes

**[Product] Fix Description**
- What was broken
- What now works correctly

### Security

**[Product] Security Update**
- What was addressed (without exposing vulnerability details)
- Recommended action if any

### Breaking Changes

**[Product] Breaking Change**
- What changed
- **Migration**: How to adapt existing code/usage
- Why the change was necessary
```

### 5. Dual Output

Generate two formats:

**1. CHANGELOG.md Format** (default):
```markdown
## 2024-02-04

### New Features

**[API] Real-time Compliance Monitoring**
- Monitor compliance status in real-time via WebSocket connection
- Receive instant notifications when compliance state changes
- Ideal for building responsive compliance dashboards

**[CLI] Interactive Setup Wizard**
- New `kosli init` command guides you through initial configuration
- Auto-detects project type and suggests best practices
- Reduces setup time from hours to minutes
```

**2. Slack Format** (if requested):
```
🎉 *Release Notes - February 4, 2024*

*New Features*
• [API] Real-time Compliance Monitoring - Monitor compliance status in real-time via WebSocket connection
• [CLI] Interactive Setup Wizard - New `kosli init` command guides setup in minutes

*Improvements*
• [UI] Faster dashboard loading - 3x performance improvement on artifact views
• [Terraform] Simplified resource configuration - Reduced required fields by 40%

*Bug Fixes*
• [API] Fixed intermittent timeout on large artifact uploads
• [CLI] Resolved Windows path handling issue
```

## Product Tags

Use consistent tags to indicate which products are affected:

- `[API]` - REST API, GraphQL endpoints
- `[UI]` - Web interface, dashboard
- `[CLI]` - Command-line tool
- `[Terraform]` - Terraform provider
- `[All]` - Affects all products
- `[Docs]` - Documentation changes

Tags help users quickly scan for relevant changes.

## Writing Guidelines

### Be User-Focused

❌ **Technical (avoid):**
"Refactored artifact service to use async workers"

✅ **User-Focused (good):**
"**[API] Faster Artifact Processing** - Artifact uploads now process 5x faster through improved background handling"

### Highlight Benefits

❌ **Feature-focused:**
"Added new `--parallel` flag to CLI"

✅ **Benefit-focused:**
"**[CLI] Parallel Deployments** - Deploy to multiple environments simultaneously with `--parallel` flag, reducing deployment time by 70%"

### Use Active Voice

❌ **Passive:**
"Support for custom tags has been added"

✅ **Active:**
"**[All] Custom Tags** - Tag artifacts and environments with custom metadata for better organization"

### Show Outcomes

❌ **Vague:**
"Improved performance"

✅ **Specific:**
"**[UI] 3x Faster Dashboard Loading** - Artifact views now load in under 1 second, even with 10,000+ artifacts"

### Keep It Concise

- One-sentence descriptions preferred
- Maximum 2-3 sentences for complex features
- Bullet points for multiple aspects
- Save detailed docs for product documentation

### Example Transformation

**Git Commit:**
```
fix: resolve race condition in artifact validator causing
intermittent validation failures under high load #1234
```

**Clarifying Questions:**
- Q: What user problem did this solve?
- A: Deployments were occasionally failing validation when many teams deployed simultaneously

**Changelog Entry:**
```markdown
### Bug Fixes

**[CLI] Reliable Validation Under Load**
- Fixed intermittent validation failures during high-traffic periods
- Deployments now succeed consistently even when multiple teams deploy simultaneously
```

## Workflow

### Step 1: Determine Scope

```bash
# If user provided argument
if [ -n "$ARGUMENTS" ]; then
  SINCE="$ARGUMENTS"
else
  # Find most recent tag
  SINCE=$(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~10")
fi
```

### Step 2: Analyze Commits

```bash
# Get commit history
git log --oneline --no-merges $SINCE..HEAD

# Get detailed commit info
git log --no-merges --format="%h|%s|%b" $SINCE..HEAD
```

### Step 3: Categorize & Question

For each commit:
1. Identify change type from commit message or content
2. Determine if user-facing or internal
3. Ask clarifying questions for user-facing changes:
   - User benefit or outcome?
   - Which products affected?
   - Any breaking changes?
   - How to describe for users?

### Step 4: Generate Entry

Create dated section with categorized changes:

```markdown
## YYYY-MM-DD

[Categories with entries...]
```

### Step 5: Insert into CHANGELOG

Read existing CHANGELOG.md (or create if missing):
- Insert new entry at the top (after title/intro)
- Preserve existing entries
- Add blank line between entries

### Step 6: Offer Formats

Ask user which format they want:
- **File only**: Update CHANGELOG.md
- **Slack format**: Generate Slack-friendly version
- **Both**: Update file and show Slack version

## Example: Full Workflow

**User invokes:**
```
/changelog-creator v1.5.0
```

**Skill analyzes:**
```
Analyzing commits since v1.5.0...
Found 12 commits:
- feat: add real-time compliance monitoring
- feat(cli): interactive setup wizard
- fix: artifact upload timeout on large files
- perf: optimize dashboard rendering
- ...
```

**Skill asks questions:**
```
I found 12 changes to document. Let me ask about the significant ones:

1. "Add real-time compliance monitoring"
   - What's the key user benefit? [User responds: Monitor compliance instantly via WebSocket]
   - Which products? [User selects: API]
   - Breaking change? [User: No]

2. "Interactive setup wizard"
   - What's the key benefit? [User: Guides initial config, reduces setup time]
   - Which products? [User: CLI]
   - Breaking change? [User: No]

...
```

**Skill generates entry:**
```markdown
## 2024-02-04

### New Features

**[API] Real-time Compliance Monitoring**
- Monitor compliance status in real-time via WebSocket connection
- Receive instant notifications when compliance state changes
- Ideal for building responsive compliance dashboards

**[CLI] Interactive Setup Wizard**
- New `kosli init` command guides you through initial configuration
- Auto-detects project type and suggests best practices
- Reduces setup time from hours to minutes

### Improvements

**[UI] 3x Faster Dashboard Loading**
- Artifact views now load in under 1 second
- Significant performance improvement for workspaces with 10,000+ artifacts

### Bug Fixes

**[API] Reliable Large File Uploads**
- Fixed timeout issue when uploading artifacts larger than 500MB
- Uploads now complete successfully regardless of file size
```

**Skill asks:**
```
Changelog entry created! What format do you need?
- Update CHANGELOG.md only
- Show Slack format for announcement
- Both
```

## Tips for Great Changelog Entries

### Combine Related Changes

If multiple commits address the same feature:

❌ **Separate entries:**
- Add search endpoint
- Add pagination to search
- Add filters to search

✅ **Combined entry:**
**[API] Powerful Search** - Search artifacts with pagination and advanced filters for precise results

### Omit Internal Changes

Skip commits that don't affect users:
- Refactoring without behavior changes
- Test updates
- CI/CD changes
- Dependency updates (unless security-related)

### Group By Impact

Order entries within categories by impact:
1. Most significant or requested features first
2. Major fixes before minor ones
3. Breaking changes always at top of their section

### Use Consistent Formatting

- **Bold** for feature/fix titles
- `Code blocks` for commands, APIs, or technical terms
- **Product tags** at start of every entry
- Sentence case for descriptions
- No period at end of title, period at end of descriptions

## Supporting Files

### Commit Message Patterns

The skill recognizes conventional commit prefixes:
- `feat:` → New Features
- `fix:` → Bug Fixes
- `perf:` → Improvements (performance)
- `security:` → Security
- `BREAKING:` → Breaking Changes
- `docs:` → Documentation
- `refactor:`, `test:`, `ci:` → Usually omitted (internal)

### Product Detection

Detects product from commit scope or file paths:
- `feat(api):` → [API]
- `feat(cli):` → [CLI]
- `feat(ui):` → [UI]
- `fix(terraform):` → [Terraform]
- Changes in `ui/` directory → [UI]
- Changes in `cli/` directory → [CLI]

## Checklist

Before finalizing changelog entry:

- [ ] All user-facing changes included
- [ ] Internal/refactoring changes excluded
- [ ] Each entry has clear user benefit
- [ ] Product tags are accurate
- [ ] Breaking changes clearly marked with migration guidance
- [ ] Security fixes included without exposing vulnerability details
- [ ] Tone is positive and benefit-focused
- [ ] Entry date is correct
- [ ] Existing CHANGELOG.md preserved correctly
- [ ] Entry inserted at correct position (top)

## Related Skills

- See [skill-creator](../skill-creator/SKILL.md) for creating new skills
- See [adr-creator](../adr-creator/SKILL.md) for documenting architectural decisions

## Notes

- Changelog entries are permanent - take time to make them clear
- Users read changelogs to understand what changed and why they should care
- Think "press release" not "commit log"
- When in doubt, focus on user benefits over technical implementation
- Breaking changes deserve extra attention and clear migration paths
