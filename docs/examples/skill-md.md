# Skill Definition for Claude Code

tree-differ can be integrated as a Claude Code skill for JSON comparison tasks.

## Skill configuration

```markdown
---
name: json-diff
description: Compare two JSON documents and return structured changes with JSONPath-annotated paths
triggers:
  - compare JSON
  - diff JSON
  - what changed between
---

Use the `differ` CLI to compare JSON documents.

## Usage

\`\`\`bash
# Compare two files
differ before.json after.json -pretty

# Compare inline JSON
differ -left-input '<left>' -right-input '<right>' -pretty

# Get just the stats
differ before.json after.json -format stat

# Get RFC 6902 patch
differ before.json after.json -format patch
\`\`\`

## When to use

- Comparing API responses before and after changes
- Detecting configuration drift
- Validating state transitions
- Generating change summaries for PRs
```

## Agent workflow

1. Agent captures baseline JSON (API response, config file, etc.)
2. Agent captures current state
3. Agent runs `differ` to get structured changes
4. Agent uses the machine-readable output to report or act on changes
