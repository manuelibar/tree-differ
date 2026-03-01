# MCP Integration

tree-differ can be exposed as an MCP tool for agent-driven JSON comparison.

## Tool definition

```json
{
  "name": "json_diff",
  "description": "Compare two JSON documents and return structured changes",
  "inputSchema": {
    "type": "object",
    "properties": {
      "left": {"description": "Baseline JSON document"},
      "right": {"description": "Target JSON document"},
      "format": {"type": "string", "enum": ["delta", "patch", "merge", "stat", "paths"]},
      "only": {"type": "array", "items": {"type": "string"}},
      "ignore": {"type": "array", "items": {"type": "string"}}
    },
    "required": ["left", "right"]
  }
}
```

## Implementation

```go
func handleDiffTool(params json.RawMessage) (json.RawMessage, error) {
    var req differ.DiffRequest
    if err := json.Unmarshal(params, &req); err != nil {
        return nil, err
    }
    return req.Execute()
}
```

## Agent use case: drift detection

An agent monitoring API responses can use compiled snapshots for efficient repeated comparison:

```go
snap, _ := differ.Compile(baselineResponse)

// On each poll
result, _ := snap.Diff(latestResponse)
if !result.Equal {
    // Report changes to agent
    out, _ := differ.FormatResult(result, differ.FormatPaths, false)
    log.Printf("drift detected: %s", out)
}
```
