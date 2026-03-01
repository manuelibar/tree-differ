# REST Middleware Integration

Use `DiffRequest` as the wire format for embedding diff operations in HTTP APIs.

## Handler

```go
package main

import (
    "encoding/json"
    "net/http"

    "github.com/mibar/tree-differ/pkg/differ"
)

func diffHandler(w http.ResponseWriter, r *http.Request) {
    var req differ.DiffRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    out, err := req.Execute()
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnprocessableEntity)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(out)
}
```

## Request format

```json
{
  "left": {"name": "Alice", "age": 30},
  "right": {"name": "Bob", "age": 30, "email": "bob@x.com"},
  "format": "delta",
  "ignore": ["$.metadata"]
}
```

## Response

```json
{
  "equal": false,
  "stats": {"added": 1, "removed": 0, "replaced": 1, "total": 2},
  "changes": [
    {"path": "$.email", "type": "added", "value": "bob@x.com"},
    {"path": "$.name", "type": "replaced", "from": "Alice", "to": "Bob"}
  ]
}
```
