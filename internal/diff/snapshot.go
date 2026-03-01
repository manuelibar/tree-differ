package diff

import (
	"encoding/binary"
	"encoding/json"
	"hash/fnv"
	"math"
	"sort"

	"github.com/mibar/jsonpath/pkg/jsonpath"
)

type nodeKind int

const (
	kindScalar nodeKind = iota
	kindObject
	kindArray
)

// snapshotNode is a hash-annotated node in a JSON tree.
// Each node stores the FNV-1a hash of its entire subtree's canonical form,
// enabling O(changes) comparison by skipping unchanged subtrees.
type snapshotNode struct {
	hash     uint64
	value    any
	children map[string]*snapshotNode // for objects
	elements []*snapshotNode          // for arrays
	kind     nodeKind
}

// Snapshot is an immutable, hash-annotated copy of a JSON document.
// Build one from a baseline and compare many target documents against it.
// Thread-safe after construction.
type Snapshot struct {
	root   *snapshotNode
	limits Limits
	filter *Filter
}

// BuildSnapshot creates a Snapshot from a parsed JSON tree.
func BuildSnapshot(tree any, limits Limits, filter *Filter) *Snapshot {
	root := buildNode(tree)
	return &Snapshot{root: root, limits: limits, filter: filter}
}

// Diff compares a target document against this snapshot.
// It builds a hash tree for the target, then walks both trees comparing
// hashes to skip unchanged subtrees in O(changes) time.
func (s *Snapshot) Diff(target any) *Result {
	maxDepth := s.limits.EffectiveMaxDepth()
	targetSnap := buildNode(target)
	changes := snapshotCompare(s.root, targetSnap, jsonpath.NewPathBuilder(), 0, maxDepth, s.filter)
	return NewResult(changes)
}

func buildNode(v any) *snapshotNode {
	switch val := v.(type) {
	case map[string]any:
		children := make(map[string]*snapshotNode, len(val))
		for k, child := range val {
			children[k] = buildNode(child)
		}
		h := hashObject(val, children)
		return &snapshotNode{
			hash:     h,
			value:    val,
			children: children,
			kind:     kindObject,
		}
	case []any:
		elements := make([]*snapshotNode, len(val))
		for i, child := range val {
			elements[i] = buildNode(child)
		}
		h := hashArray(elements)
		return &snapshotNode{
			hash:     h,
			value:    val,
			elements: elements,
			kind:     kindArray,
		}
	default:
		h := hashScalar(v)
		return &snapshotNode{
			hash:  h,
			value: v,
			kind:  kindScalar,
		}
	}
}

// hashScalar produces a deterministic hash for a JSON scalar value.
func hashScalar(v any) uint64 {
	h := fnv.New64a()
	switch val := v.(type) {
	case nil:
		h.Write([]byte{0x00})
	case bool:
		if val {
			h.Write([]byte{0x01, 0x01})
		} else {
			h.Write([]byte{0x01, 0x00})
		}
	case json.Number:
		h.Write([]byte{0x02})
		h.Write([]byte(val.String()))
	case string:
		h.Write([]byte{0x03})
		h.Write([]byte(val))
	case float64:
		h.Write([]byte{0x04})
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(val))
		h.Write(buf[:])
	default:
		// Fallback: marshal to JSON
		h.Write([]byte{0xFF})
		b, _ := json.Marshal(val)
		h.Write(b)
	}
	return h.Sum64()
}

// hashObject produces a deterministic hash for a JSON object.
// Keys are sorted for determinism.
func hashObject(obj map[string]any, children map[string]*snapshotNode) uint64 {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := fnv.New64a()
	h.Write([]byte{0x10}) // object type marker
	var buf [8]byte
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0x00}) // key terminator
		binary.LittleEndian.PutUint64(buf[:], children[k].hash)
		h.Write(buf[:])
	}
	return h.Sum64()
}

// hashArray produces a deterministic hash for a JSON array.
func hashArray(elements []*snapshotNode) uint64 {
	h := fnv.New64a()
	h.Write([]byte{0x11}) // array type marker
	var buf [8]byte
	for _, elem := range elements {
		binary.LittleEndian.PutUint64(buf[:], elem.hash)
		h.Write(buf[:])
	}
	return h.Sum64()
}

// snapshotCompare walks two snapshot trees, using hash comparison for early exits.
// Both left and right are pre-hashed, so subtree comparison is O(1) per node.
func snapshotCompare(left, right *snapshotNode, path *jsonpath.PathBuilder, depth, maxDepth int, filter *Filter) []Change {
	if maxDepth > 0 && depth > maxDepth {
		var from, to any
		if left != nil {
			from = left.value
		}
		if right != nil {
			to = right.value
		}
		return []Change{{Path: path.String(), Type: Replaced, From: from, To: to}}
	}

	p := path.String()

	if filter != nil && filter.ShouldSkip(p) {
		return nil
	}

	// Fast path: both exist and hashes match → entire subtree unchanged
	if left != nil && right != nil && left.hash == right.hash {
		return nil
	}

	if left == nil && right == nil {
		return nil
	}
	if left == nil {
		return []Change{{Path: p, Type: Added, Value: right.value}}
	}
	if right == nil {
		return []Change{{Path: p, Type: Removed, Value: left.value}}
	}

	// Type mismatch
	if left.kind != right.kind {
		return []Change{{Path: p, Type: Replaced, From: left.value, To: right.value}}
	}

	switch left.kind {
	case kindObject:
		return snapshotWalkObjects(left, right, path, depth, maxDepth, filter)
	case kindArray:
		return snapshotWalkArrays(left, right, path, depth, maxDepth, filter)
	default:
		// Scalar — hashes already compared and differ
		return []Change{{Path: p, Type: Replaced, From: left.value, To: right.value}}
	}
}

func snapshotWalkObjects(left, right *snapshotNode, path *jsonpath.PathBuilder, depth, maxDepth int, filter *Filter) []Change {
	leftObj := left.value.(map[string]any)
	rightObj := right.value.(map[string]any)
	keys := sortedUnion(leftObj, rightObj)
	var changes []Change

	for _, k := range keys {
		childPath := path.Child(k)
		leftChild, lOk := left.children[k]
		rightChild, rOk := right.children[k]

		switch {
		case lOk && rOk:
			changes = append(changes, snapshotCompare(leftChild, rightChild, childPath, depth+1, maxDepth, filter)...)
		case lOk:
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Removed, Value: leftChild.value})
			}
		default:
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Added, Value: rightChild.value})
			}
		}
	}
	return changes
}

func snapshotWalkArrays(left, right *snapshotNode, path *jsonpath.PathBuilder, depth, maxDepth int, filter *Filter) []Change {
	leftArr := left.elements
	rightArr := right.elements
	maxLen := len(leftArr)
	if len(rightArr) > maxLen {
		maxLen = len(rightArr)
	}

	var changes []Change
	for i := 0; i < maxLen; i++ {
		childPath := path.Index(i)
		switch {
		case i < len(leftArr) && i < len(rightArr):
			changes = append(changes, snapshotCompare(leftArr[i], rightArr[i], childPath, depth+1, maxDepth, filter)...)
		case i < len(leftArr):
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Removed, Value: leftArr[i].value})
			}
		default:
			p := childPath.String()
			if filter == nil || !filter.ShouldSkip(p) {
				changes = append(changes, Change{Path: p, Type: Added, Value: rightArr[i].value})
			}
		}
	}
	return changes
}
