package diff

import (
	"encoding/json"
	"sort"

	"github.com/mibar/jsonpath/pkg/jsonpath"
)

// Compare produces a list of changes between two parsed JSON trees.
// Both values must be the result of json.Unmarshal with UseNumber enabled.
func Compare(left, right any, limits Limits, filter *Filter) *Result {
	maxDepth := limits.EffectiveMaxDepth()
	changes := walkCompare(left, right, jsonpath.NewPathBuilder(), 0, maxDepth, filter)
	return NewResult(changes)
}

func walkCompare(left, right any, path *jsonpath.PathBuilder, depth, maxDepth int, filter *Filter) []Change {
	if maxDepth > 0 && depth > maxDepth {
		return []Change{{
			Path:  path.String(),
			Type:  Replaced,
			From:  left,
			To:    right,
		}}
	}

	p := path.String()

	if filter != nil && filter.ShouldSkip(p) {
		return nil
	}

	if left == nil && right == nil {
		return nil
	}
	if left == nil {
		return []Change{{Path: p, Type: Added, Value: right}}
	}
	if right == nil {
		return []Change{{Path: p, Type: Removed, Value: left}}
	}

	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	leftArr, leftIsArr := left.([]any)
	rightArr, rightIsArr := right.([]any)

	// type mismatch → replaced
	if leftIsMap != rightIsMap || leftIsArr != rightIsArr {
		return []Change{{Path: p, Type: Replaced, From: left, To: right}}
	}

	switch {
	case leftIsMap:
		return walkObjects(leftMap, rightMap, path, depth, maxDepth, filter)
	case leftIsArr:
		return walkArrays(leftArr, rightArr, path, depth, maxDepth, filter)
	default:
		if !scalarEqual(left, right) {
			return []Change{{Path: p, Type: Replaced, From: left, To: right}}
		}
		return nil
	}
}

func walkObjects(left, right map[string]any, path *jsonpath.PathBuilder, depth, maxDepth int, filter *Filter) []Change {
	keys := sortedUnion(left, right)
	var changes []Change
	for _, k := range keys {
		childPath := path.Child(k)
		lv, lOk := left[k]
		rv, rOk := right[k]

		switch {
		case lOk && rOk:
			changes = append(changes, walkCompare(lv, rv, childPath, depth+1, maxDepth, filter)...)
		case lOk:
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Removed, Value: lv})
			}
		default:
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Added, Value: rv})
			}
		}
	}
	return changes
}

func walkArrays(left, right []any, path *jsonpath.PathBuilder, depth, maxDepth int, filter *Filter) []Change {
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}

	var changes []Change
	for i := 0; i < maxLen; i++ {
		childPath := path.Index(i)
		switch {
		case i < len(left) && i < len(right):
			changes = append(changes, walkCompare(left[i], right[i], childPath, depth+1, maxDepth, filter)...)
		case i < len(left):
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Removed, Value: left[i]})
			}
		default:
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Added, Value: right[i]})
			}
		}
	}
	return changes
}

// scalarEqual compares two JSON scalar values, handling json.Number correctly.
func scalarEqual(a, b any) bool {
	na, aIsNum := a.(json.Number)
	nb, bIsNum := b.(json.Number)
	if aIsNum && bIsNum {
		return na.String() == nb.String()
	}
	if aIsNum || bIsNum {
		return false
	}
	return a == b
}

// sortedUnion returns the sorted union of keys from two maps.
func sortedUnion(a, b map[string]any) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	keys := make([]string, 0, len(a)+len(b))
	for k := range a {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	for k := range b {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}
