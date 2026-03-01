package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Format identifies an output format for diff results.
type Format string

const (
	FormatDelta Format = "delta"
	FormatPatch Format = "patch"
	FormatMerge Format = "merge"
	FormatStat  Format = "stat"
	FormatPaths Format = "paths"
)

// ParseFormat converts a string to a Format, returning an error for unknown formats.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatDelta, FormatPatch, FormatMerge, FormatStat, FormatPaths:
		return Format(s), nil
	case "":
		return FormatDelta, nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: delta, patch, merge, stat, paths)", s)
	}
}

// FormatResult encodes a Result in the given format.
func FormatResult(r *Result, f Format, pretty bool) ([]byte, error) {
	switch f {
	case FormatDelta, "":
		return formatDelta(r, pretty)
	case FormatPatch:
		return formatPatch(r, pretty)
	case FormatMerge:
		return formatMerge(r, pretty)
	case FormatStat:
		return formatStat(r, pretty)
	case FormatPaths:
		return formatPaths(r, pretty)
	default:
		return nil, fmt.Errorf("unknown format %q", f)
	}
}

// formatDelta produces the default structured diff output.
func formatDelta(r *Result, pretty bool) ([]byte, error) {
	return marshalJSON(r, pretty)
}

// patchOp represents an RFC 6902 JSON Patch operation.
type patchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
	From  any    `json:"from,omitempty"`
}

// formatPatch produces RFC 6902 JSON Patch output.
func formatPatch(r *Result, pretty bool) ([]byte, error) {
	ops := make([]patchOp, 0, len(r.Changes))
	for _, c := range r.Changes {
		ptr := jsonPathToPointer(c.Path)
		switch c.Type {
		case Added:
			ops = append(ops, patchOp{Op: "add", Path: ptr, Value: c.Value})
		case Removed:
			ops = append(ops, patchOp{Op: "remove", Path: ptr})
		case Replaced:
			ops = append(ops, patchOp{Op: "replace", Path: ptr, Value: c.To})
		}
	}
	return marshalJSON(ops, pretty)
}

// formatMerge produces RFC 7396 JSON Merge Patch output.
// Only meaningful for object-level changes; array changes fall back to replacement.
func formatMerge(r *Result, pretty bool) ([]byte, error) {
	patch := make(map[string]any)
	for _, c := range r.Changes {
		setNested(patch, c)
	}
	return marshalJSON(patch, pretty)
}

// setNested sets a value in a nested map structure based on a Change.
func setNested(root map[string]any, c Change) {
	segments := splitPath(c.Path)
	if len(segments) == 0 {
		return
	}

	current := root
	for _, seg := range segments[:len(segments)-1] {
		child, ok := current[seg]
		if !ok {
			child = make(map[string]any)
			current[seg] = child
		}
		if m, ok := child.(map[string]any); ok {
			current = m
		} else {
			return
		}
	}

	last := segments[len(segments)-1]
	switch c.Type {
	case Added:
		current[last] = c.Value
	case Replaced:
		current[last] = c.To
	case Removed:
		current[last] = nil
	}
}

// splitPath splits a JSONPath like "$.a.b[0].c" into segments ["a","b","0","c"].
func splitPath(path string) []string {
	// Strip leading "$"
	if strings.HasPrefix(path, "$") {
		path = path[1:]
	}
	if path == "" {
		return nil
	}

	var segments []string
	i := 0
	for i < len(path) {
		switch path[i] {
		case '.':
			i++
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			if i > start {
				segments = append(segments, path[start:i])
			}
		case '[':
			i++
			start := i
			for i < len(path) && path[i] != ']' {
				i++
			}
			seg := path[start:i]
			// Remove quotes for bracket notation keys
			if len(seg) >= 2 && seg[0] == '"' && seg[len(seg)-1] == '"' {
				seg = seg[1 : len(seg)-1]
			}
			segments = append(segments, seg)
			if i < len(path) {
				i++ // skip ']'
			}
		default:
			i++
		}
	}
	return segments
}

// statOutput is a minimal stats-only output.
type statOutput struct {
	Equal bool  `json:"equal"`
	Stats Stats `json:"stats"`
}

// formatStat produces stats-only output.
func formatStat(r *Result, pretty bool) ([]byte, error) {
	return marshalJSON(statOutput{Equal: r.Equal, Stats: r.Stats}, pretty)
}

// formatPaths produces a list of changed JSONPath strings.
func formatPaths(r *Result, pretty bool) ([]byte, error) {
	paths := make([]string, len(r.Changes))
	for i, c := range r.Changes {
		paths[i] = c.Path
	}
	sort.Strings(paths)
	return marshalJSON(paths, pretty)
}

// jsonPathToPointer converts a JSONPath to a JSON Pointer (RFC 6901).
// "$.foo.bar[0]" → "/foo/bar/0"
func jsonPathToPointer(path string) string {
	segments := splitPath(path)
	if len(segments) == 0 {
		return ""
	}
	var b strings.Builder
	for _, seg := range segments {
		b.WriteByte('/')
		// Escape per RFC 6901: ~ → ~0, / → ~1
		seg = strings.ReplaceAll(seg, "~", "~0")
		seg = strings.ReplaceAll(seg, "/", "~1")
		b.WriteString(seg)
	}
	return b.String()
}

func marshalJSON(v any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
