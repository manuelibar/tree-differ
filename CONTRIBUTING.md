# Contributing to tree-differ

## Project Structure

```
tree-differ/
├── cmd/differ/       CLI entry point
├── pkg/differ/       Public API (thin re-exports from internal)
├── internal/diff/    Core engine (snapshot, compare, format)
└── docs/             Architecture and examples
```

All real logic lives in `internal/diff/`. The `pkg/differ/` package re-exports types and provides the public-facing API with functional options.

## Go Style

Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).

## Doc Comments

- Every exported symbol MUST have a doc comment
- Start with the symbol name: `// Diff compares two JSON documents.`
- Say _why_, not _what_
- Bool-returning functions: use "reports whether"
- `Must*` pattern: "like X but panics on error"

## Functional Options

Use functional options for optional parameters:

```go
result, err := differ.Diff(left, right,
    differ.WithFormat(differ.FormatPatch),
    differ.WithIgnore("$.metadata"),
)
```

Do not create function variants for optional behaviour.

## Tests

- Tests live next to code (`*_test.go` in the same package)
- Use `t.Fatal()` for setup failures, `t.Errorf()` for assertions
- Benchmarks: `b.ReportAllocs()` and `b.ResetTimer()`

## Commit Messages

Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `bench:`

## PR Checklist

- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean
- [ ] New exported symbols have doc comments
- [ ] No external dependencies added
