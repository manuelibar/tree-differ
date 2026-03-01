# Architecture

tree-differ operates as a three-stage pipeline:

```
Parse → Compare → Format
```

## Stage 1: Parse

JSON input is decoded with `json.Decoder.UseNumber()` to preserve numeric precision. The result is a standard Go `any` tree (`map[string]any`, `[]any`, scalars).

## Stage 2: Compare

Two comparison modes share the same walker structure:

### One-shot mode

Direct recursive walk of two `any` trees. Objects are compared by sorted key union, arrays by positional index.

### Snapshot-guided mode

Both baseline and target are converted to **snapshot trees** — each node annotated with the FNV-1a hash of its canonical subtree representation.

```
Snapshot of {"users":[{"name":"Alice"}],"version":1}

root (hash: 0xa3f1...)
├── "users" (hash: 0xb7c2...)
│   └── [0] (hash: 0xd4e5...)
│       └── "name" (hash: 0xf6a7...) = "Alice"
└── "version" (hash: 0x5f6a...) = 1
```

During comparison, if `left.hash == right.hash`, the entire subtree is skipped. This makes repeated diffs against a baseline O(changes) rather than O(document).

### Canonical hashing

Deterministic byte representation per node type:
- **Scalars**: type marker byte + value bytes
- **Objects**: sorted `key:childHash` pairs
- **Arrays**: ordered `childHash` sequence

This ensures structurally identical subtrees produce identical hashes regardless of original key order.

## Stage 3: Format

The flat `[]Change` list is formatted into one of five output formats:

| Format | Description |
|--------|-------------|
| delta  | Full structured diff (default) |
| patch  | RFC 6902 JSON Patch |
| merge  | RFC 7396 JSON Merge Patch |
| stat   | Stats only |
| paths  | Changed paths only |

## Package boundaries

```
cmd/differ/                            CLI: flag parsing, I/O, exit codes
pkg/differ/                            Public API: Diff(), Compile(), options, wire format
internal/diff/                         Core: snapshot, compare, format, filter, path
github.com/mibar/jsonpath/pkg/jsonpath PathBuilder for JSONPath-annotated change paths
```

`pkg/differ` is a thin layer that parses JSON, delegates to `internal/diff`, and exposes type aliases. All algorithmic logic lives in `internal/diff`. JSONPath string construction uses `PathBuilder` from the standalone `github.com/mibar/jsonpath` module.
