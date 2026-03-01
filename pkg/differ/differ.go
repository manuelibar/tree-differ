// Package differ provides structured JSON diffing with two modes:
// one-shot comparison and compiled snapshot mode for repeated diffs
// against a baseline document.
package differ

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mibar/tree-differ/internal/diff"
)

// Re-exported types from internal/diff.
type (
	Result     = diff.Result
	Change     = diff.Change
	ChangeType = diff.ChangeType
	Stats      = diff.Stats
	Format     = diff.Format
	Limits     = diff.Limits
	DepthError = diff.DepthError
)

// Re-exported constants.
const (
	Added    = diff.Added
	Removed  = diff.Removed
	Replaced = diff.Replaced

	FormatDelta = diff.FormatDelta
	FormatPatch = diff.FormatPatch
	FormatMerge = diff.FormatMerge
	FormatStat  = diff.FormatStat
	FormatPaths = diff.FormatPaths

	DefaultMaxDepth = diff.DefaultMaxDepth
)

// Option configures a diff operation.
type Option func(*options)

type options struct {
	format Format
	only   []string
	ignore []string
	limits Limits
	pretty bool
}

// WithFormat sets the output format.
func WithFormat(f Format) Option {
	return func(o *options) { o.format = f }
}

// WithOnly limits the diff to the specified JSONPath prefixes.
func WithOnly(paths ...string) Option {
	return func(o *options) { o.only = paths }
}

// WithIgnore excludes the specified JSONPath prefixes from the diff.
func WithIgnore(paths ...string) Option {
	return func(o *options) { o.ignore = paths }
}

// WithLimits sets safety bounds for traversal.
func WithLimits(l Limits) Option {
	return func(o *options) { o.limits = l }
}

// WithPretty enables indented JSON output.
func WithPretty(p bool) Option {
	return func(o *options) { o.pretty = p }
}

func buildOptions(opts []Option) options {
	var o options
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// Diff compares two JSON documents and returns a structured result.
func Diff(left, right []byte, opts ...Option) (*Result, error) {
	o := buildOptions(opts)
	leftTree, err := parseJSON(left)
	if err != nil {
		return nil, fmt.Errorf("parse left: %w", err)
	}
	rightTree, err := parseJSON(right)
	if err != nil {
		return nil, fmt.Errorf("parse right: %w", err)
	}
	filter := diff.NewFilter(o.only, o.ignore)
	return diff.Compare(leftTree, rightTree, o.limits, filter), nil
}

// MustDiff is like Diff but panics on error.
func MustDiff(left, right []byte, opts ...Option) *Result {
	r, err := Diff(left, right, opts...)
	if err != nil {
		panic(err)
	}
	return r
}

// Snapshot is an immutable, hash-annotated copy of a JSON document.
// Build one from a baseline and compare many target documents against it.
// Thread-safe after construction.
type Snapshot struct {
	snap *diff.Snapshot
}

// Compile parses a baseline JSON document and builds a hash-annotated snapshot.
// The snapshot can then be used to efficiently diff many documents against
// this baseline in O(changes) time.
func Compile(baseline []byte, opts ...Option) (*Snapshot, error) {
	o := buildOptions(opts)
	tree, err := parseJSON(baseline)
	if err != nil {
		return nil, fmt.Errorf("parse baseline: %w", err)
	}
	filter := diff.NewFilter(o.only, o.ignore)
	snap := diff.BuildSnapshot(tree, o.limits, filter)
	return &Snapshot{snap: snap}, nil
}

// Diff compares a target JSON document against the compiled snapshot.
func (s *Snapshot) Diff(target []byte) (*Result, error) {
	tree, err := parseJSON(target)
	if err != nil {
		return nil, fmt.Errorf("parse target: %w", err)
	}
	return s.snap.Diff(tree), nil
}

// FormatResult encodes a Result in the given format.
func FormatResult(r *Result, f Format, pretty bool) ([]byte, error) {
	return diff.FormatResult(r, f, pretty)
}

// ParseFormat converts a string to a Format.
func ParseFormat(s string) (Format, error) {
	return diff.ParseFormat(s)
}

// DiffRequest is a wire format for embedding diff operations in REST/MCP APIs.
type DiffRequest struct {
	Left   json.RawMessage `json:"left"`
	Right  json.RawMessage `json:"right"`
	Only   []string        `json:"only,omitempty"`
	Ignore []string        `json:"ignore,omitempty"`
	Format string          `json:"format,omitempty"`
}

// UnmarshalJSON validates the DiffRequest during deserialization.
func (r *DiffRequest) UnmarshalJSON(data []byte) error {
	type alias DiffRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if len(a.Left) == 0 {
		return fmt.Errorf("left document is required")
	}
	if len(a.Right) == 0 {
		return fmt.Errorf("right document is required")
	}
	if a.Format != "" {
		if _, err := diff.ParseFormat(a.Format); err != nil {
			return err
		}
	}
	*r = DiffRequest(a)
	return nil
}

// Execute runs the diff request and returns formatted output.
func (r *DiffRequest) Execute() ([]byte, error) {
	var opts []Option
	if len(r.Only) > 0 {
		opts = append(opts, WithOnly(r.Only...))
	}
	if len(r.Ignore) > 0 {
		opts = append(opts, WithIgnore(r.Ignore...))
	}
	result, err := Diff(r.Left, r.Right, opts...)
	if err != nil {
		return nil, err
	}
	f, _ := diff.ParseFormat(r.Format)
	return diff.FormatResult(result, f, false)
}

// Ptr returns a pointer to v. Useful for setting Limits fields.
func Ptr[T any](v T) *T {
	return &v
}

func parseJSON(data []byte) (any, error) {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
