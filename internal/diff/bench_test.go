package diff

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// --- Data generators ---

func flatObject(n int) []byte {
	var b strings.Builder
	b.WriteByte('{')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"field_%d":%d`, i, i)
	}
	b.WriteByte('}')
	return []byte(b.String())
}

func flatObjectChanged(n, nChanged int) []byte {
	var b strings.Builder
	b.WriteByte('{')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		v := i
		if i < nChanged {
			v = i + 1000
		}
		fmt.Fprintf(&b, `"field_%d":%d`, i, v)
	}
	b.WriteByte('}')
	return []byte(b.String())
}

func nestedObject() []byte {
	return []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@x.com", "settings": {"theme": "light", "lang": "en"}},
			{"id": 2, "name": "Bob", "email": "bob@x.com", "settings": {"theme": "dark", "lang": "es"}},
			{"id": 3, "name": "Charlie", "email": "charlie@x.com", "settings": {"theme": "light", "lang": "fr"}}
		],
		"metadata": {"version": 1, "count": 3, "updated": "2024-01-01"},
		"config": {"debug": false, "retries": 3, "timeout": 30}
	}`)
}

func nestedObjectChanged() []byte {
	return []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@x.com", "settings": {"theme": "dark", "lang": "en"}},
			{"id": 2, "name": "Bob", "email": "bob@x.com", "settings": {"theme": "dark", "lang": "es"}},
			{"id": 3, "name": "Charlie", "email": "charlie@x.com", "settings": {"theme": "light", "lang": "fr"}}
		],
		"metadata": {"version": 2, "count": 3, "updated": "2024-06-01"},
		"config": {"debug": false, "retries": 3, "timeout": 30}
	}`)
}

func largeArray(n int) []byte {
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"value":"item_%d"}`, i, i)
	}
	b.WriteByte(']')
	return []byte(b.String())
}

func largeArrayChanged(n, nChanged int) []byte {
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		v := fmt.Sprintf("item_%d", i)
		if i < nChanged {
			v = fmt.Sprintf("changed_%d", i)
		}
		fmt.Fprintf(&b, `{"id":%d,"value":"%s"}`, i, v)
	}
	b.WriteByte(']')
	return []byte(b.String())
}

func deeplyNested(depth int) []byte {
	var b strings.Builder
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&b, `{"d%d":`, i)
	}
	b.WriteString(`"leaf"`)
	for i := 0; i < depth; i++ {
		b.WriteByte('}')
	}
	return []byte(b.String())
}

func mustUnmarshal(data []byte) any {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		panic(err)
	}
	return v
}

// --- Benchmarks: One-shot Compare ---

func BenchmarkCompareFlat100_3Changed(b *testing.B) {
	left := mustUnmarshal(flatObject(100))
	right := mustUnmarshal(flatObjectChanged(100, 3))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Compare(left, right, Limits{}, nil)
	}
}

func BenchmarkCompareNested(b *testing.B) {
	left := mustUnmarshal(nestedObject())
	right := mustUnmarshal(nestedObjectChanged())
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Compare(left, right, Limits{}, nil)
	}
}

func BenchmarkCompareLargeArray1000_10Changed(b *testing.B) {
	left := mustUnmarshal(largeArray(1000))
	right := mustUnmarshal(largeArrayChanged(1000, 10))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Compare(left, right, Limits{}, nil)
	}
}

func BenchmarkCompareDeep100(b *testing.B) {
	left := mustUnmarshal(deeplyNested(100))
	right := mustUnmarshal(deeplyNested(100)) // same structure — should be equal
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		Compare(left, right, Limits{}, nil)
	}
}

// --- Benchmarks: Snapshot build ---

func BenchmarkSnapshotBuildFlat100(b *testing.B) {
	tree := mustUnmarshal(flatObject(100))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		BuildSnapshot(tree, Limits{}, nil)
	}
}

func BenchmarkSnapshotBuildNested(b *testing.B) {
	tree := mustUnmarshal(nestedObject())
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		BuildSnapshot(tree, Limits{}, nil)
	}
}

func BenchmarkSnapshotBuildLargeArray1000(b *testing.B) {
	tree := mustUnmarshal(largeArray(1000))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		BuildSnapshot(tree, Limits{}, nil)
	}
}

// --- Benchmarks: Snapshot-guided diff ---

func BenchmarkSnapshotDiffFlat100_3Changed(b *testing.B) {
	left := mustUnmarshal(flatObject(100))
	right := mustUnmarshal(flatObjectChanged(100, 3))
	snap := BuildSnapshot(left, Limits{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		snap.Diff(right)
	}
}

func BenchmarkSnapshotDiffNested(b *testing.B) {
	left := mustUnmarshal(nestedObject())
	right := mustUnmarshal(nestedObjectChanged())
	snap := BuildSnapshot(left, Limits{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		snap.Diff(right)
	}
}

func BenchmarkSnapshotDiffLargeArray1000_10Changed(b *testing.B) {
	left := mustUnmarshal(largeArray(1000))
	right := mustUnmarshal(largeArrayChanged(1000, 10))
	snap := BuildSnapshot(left, Limits{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		snap.Diff(right)
	}
}

func BenchmarkSnapshotDiffEqual(b *testing.B) {
	tree := mustUnmarshal(largeArray(1000))
	snap := BuildSnapshot(tree, Limits{}, nil)
	target := mustUnmarshal(largeArray(1000))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		snap.Diff(target)
	}
}

// --- Benchmarks: Format ---

func BenchmarkFormatDelta(b *testing.B) {
	left := mustUnmarshal(flatObject(100))
	right := mustUnmarshal(flatObjectChanged(100, 10))
	result := Compare(left, right, Limits{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		FormatResult(result, FormatDelta, false)
	}
}

func BenchmarkFormatPatch(b *testing.B) {
	left := mustUnmarshal(flatObject(100))
	right := mustUnmarshal(flatObjectChanged(100, 10))
	result := Compare(left, right, Limits{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		FormatResult(result, FormatPatch, false)
	}
}
