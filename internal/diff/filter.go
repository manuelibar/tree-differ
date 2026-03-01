package diff

import "strings"

// Filter controls which paths are included or excluded from diffing.
type Filter struct {
	only   []string
	ignore []string
}

// NewFilter creates a filter from only and ignore path prefixes.
// Passing nil or empty slices means no filtering for that dimension.
func NewFilter(only, ignore []string) *Filter {
	if len(only) == 0 && len(ignore) == 0 {
		return nil
	}
	return &Filter{only: only, ignore: ignore}
}

// ShouldSkip reports whether a path should be excluded from the diff.
//
// Rules:
//   - If only paths are set, skip anything that is not a prefix of, or prefixed by, an only path
//   - If ignore paths are set, skip anything that matches or is prefixed by an ignore path
func (f *Filter) ShouldSkip(path string) bool {
	if f == nil {
		return false
	}
	if len(f.ignore) > 0 && matchesIgnore(path, f.ignore) {
		return true
	}
	if len(f.only) > 0 && !matchesOnly(path, f.only) {
		return true
	}
	return false
}

// matchesOnly reports whether a path is relevant to the only filter.
// A path is relevant if:
//   - it exactly matches an only prefix (we're at that node)
//   - it starts with an only prefix (we're inside a selected subtree)
//   - an only prefix starts with path (we're on the way to a selected subtree)
func matchesOnly(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix {
			return true
		}
		if strings.HasPrefix(path, prefix) && (len(path) > len(prefix) && (path[len(prefix)] == '.' || path[len(prefix)] == '[')) {
			return true
		}
		if strings.HasPrefix(prefix, path) && (len(prefix) > len(path) && (prefix[len(path)] == '.' || prefix[len(path)] == '[')) {
			return true
		}
	}
	return false
}

// matchesIgnore reports whether a path should be ignored.
// A path is ignored if it exactly matches or is a child of an ignore prefix.
func matchesIgnore(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix {
			return true
		}
		if strings.HasPrefix(path, prefix) && len(path) > len(prefix) && (path[len(prefix)] == '.' || path[len(prefix)] == '[') {
			return true
		}
	}
	return false
}
