# tree-differ

Structured JSON diffing for Go. Machine-readable output with JSONPath-annotated changes.

## Why

Existing diff tools produce text-based output that burns context tokens and is hard for machines to parse. tree-differ produces structured diffs with stats, typed changes, and multiple output formats designed for agent consumption.

## Two Modes

**One-shot** — compare two documents directly:

```go
result, err := differ.Diff(left, right)
```

**Compiled snapshot** — build a hash-annotated baseline, diff many targets against it in O(changes) time:

```go
snap, err := differ.Compile(baseline)
// Thread-safe, reusable
r1, _ := snap.Diff(target1)
r2, _ := snap.Diff(target2)
```

The snapshot stores FNV-1a hashes at every node. During comparison, if a subtree's hash matches, it's skipped entirely — no recursion needed.

## Quick Start

### Library

```go
import "github.com/mibar/tree-differ/pkg/differ"

left := []byte(`{"name":"Alice","age":30}`)
right := []byte(`{"name":"Bob","age":30,"email":"bob@x.com"}`)

result, err := differ.Diff(left, right)
// result.Equal == false
// result.Stats == {Added:1, Removed:0, Replaced:1, Total:2}
// result.Changes:
//   {Path:"$.email", Type:"added", Value:"bob@x.com"}
//   {Path:"$.name", Type:"replaced", From:"Alice", To:"Bob"}
```

### CLI

```bash
# Two files
differ a.json b.json

# Inline JSON
differ -left-input '{"a":1}' -right-input '{"a":2}' -pretty

# Pipe from stdin
cat before.json | differ -right after.json

# Output formats
differ a.json b.json -format patch    # RFC 6902 JSON Patch
differ a.json b.json -format merge    # RFC 7396 JSON Merge Patch
differ a.json b.json -format stat     # Stats only
differ a.json b.json -format paths    # Changed paths only

# Filtering
differ a.json b.json -only '$.users'
differ a.json b.json -ignore '$.metadata,$.timestamps'
```

Exit codes: `0` equal, `1` different, `2` error.

## Output Formats

### delta (default)

```json
{"equal":false,"stats":{"added":1,"removed":1,"replaced":1,"total":3},"changes":[{"path":"$.name","type":"replaced","from":"Alice","to":"Bob"},{"path":"$.theme","type":"added","value":"dark"},{"path":"$.legacy","type":"removed","value":true}]}
```

### patch (RFC 6902)

```json
[{"op":"replace","path":"/name","value":"Bob"},{"op":"add","path":"/theme","value":"dark"},{"op":"remove","path":"/legacy"}]
```

### merge (RFC 7396)

```json
{"name":"Bob","theme":"dark","legacy":null}
```

### stat

```json
{"equal":false,"stats":{"added":1,"removed":1,"replaced":1,"total":3}}
```

### paths

```json
["$.legacy","$.name","$.theme"]
```

## API

```go
// One-shot
func Diff(left, right []byte, opts ...Option) (*Result, error)
func MustDiff(left, right []byte, opts ...Option) *Result

// Compiled snapshot
func Compile(baseline []byte, opts ...Option) (*Snapshot, error)
func (s *Snapshot) Diff(target []byte) (*Result, error)

// Options
func WithFormat(f Format) Option
func WithOnly(paths ...string) Option
func WithIgnore(paths ...string) Option
func WithLimits(l Limits) Option
func WithPretty(p bool) Option

// Formatting
func FormatResult(r *Result, f Format, pretty bool) ([]byte, error)
func ParseFormat(s string) (Format, error)

// Wire format for REST/MCP APIs
type DiffRequest struct { ... }
func (r *DiffRequest) Execute() ([]byte, error)
```

## Behaviour

- **Positional array comparison**: elements compared by index, not content
- **`json.Number` preservation**: no floating-point precision loss
- **Sorted keys**: deterministic output regardless of input key order
- **Depth limiting**: configurable max depth (default: 1000)
- **Path filtering**: prefix-based only/ignore filters
- **No external dependencies**: stdlib only

## Development

```bash
./run build   # go build ./...
./run test    # go test ./... -v
./run vet     # go vet ./...
./run bench   # benchmarks
./run differ  # run CLI
./run all     # build + vet + test
```

## License

MIT
