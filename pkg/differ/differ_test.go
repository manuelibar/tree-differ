package differ

import (
	"encoding/json"
	"testing"
)

func TestDiffOneShot(t *testing.T) {
	left := []byte(`{"name":"Alice","age":30}`)
	right := []byte(`{"name":"Bob","age":30,"email":"bob@x.com"}`)
	r, err := Diff(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if r.Equal {
		t.Error("expected not equal")
	}
	if r.Stats.Replaced != 1 || r.Stats.Added != 1 || r.Stats.Total != 2 {
		t.Errorf("unexpected stats: %+v", r.Stats)
	}
}

func TestDiffEqual(t *testing.T) {
	doc := []byte(`{"a":1,"b":[2,3]}`)
	r, err := Diff(doc, doc)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Equal {
		t.Error("expected equal")
	}
}

func TestDiffInvalidJSON(t *testing.T) {
	_, err := Diff([]byte(`{invalid`), []byte(`{}`))
	if err == nil {
		t.Error("expected error for invalid left JSON")
	}
	_, err = Diff([]byte(`{}`), []byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid right JSON")
	}
}

func TestMustDiffPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid JSON")
		}
	}()
	MustDiff([]byte(`{bad`), []byte(`{}`))
}

func TestMustDiffSuccess(t *testing.T) {
	r := MustDiff([]byte(`{"a":1}`), []byte(`{"a":2}`))
	if r.Equal {
		t.Error("expected not equal")
	}
}

func TestCompileAndDiff(t *testing.T) {
	baseline := []byte(`{"users":[{"name":"Alice"},{"name":"Bob"}],"version":1}`)
	snap, err := Compile(baseline)
	if err != nil {
		t.Fatal(err)
	}

	// Same document
	r, err := snap.Diff(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Equal {
		t.Error("expected equal for same document")
	}

	// Changed document
	target := []byte(`{"users":[{"name":"Alice"},{"name":"Charlie"}],"version":1}`)
	r, err = snap.Diff(target)
	if err != nil {
		t.Fatal(err)
	}
	if r.Equal {
		t.Error("expected not equal")
	}
	if r.Stats.Replaced != 1 {
		t.Errorf("expected 1 replaced, got stats=%+v", r.Stats)
	}
}

func TestCompileInvalidJSON(t *testing.T) {
	_, err := Compile([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid baseline")
	}
}

func TestSnapshotDiffInvalidTarget(t *testing.T) {
	snap, _ := Compile([]byte(`{}`))
	_, err := snap.Diff([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid target")
	}
}

func TestDiffWithOnly(t *testing.T) {
	left := []byte(`{"users":{"name":"Alice"},"settings":{"theme":"light"}}`)
	right := []byte(`{"users":{"name":"Bob"},"settings":{"theme":"dark"}}`)
	r, err := Diff(left, right, WithOnly("$.users"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Stats.Total != 1 {
		t.Errorf("expected 1 change (only users), got stats=%+v changes=%v", r.Stats, r.Changes)
	}
}

func TestDiffWithIgnore(t *testing.T) {
	left := []byte(`{"name":"Alice","metadata":{"ts":"2024"}}`)
	right := []byte(`{"name":"Bob","metadata":{"ts":"2025"}}`)
	r, err := Diff(left, right, WithIgnore("$.metadata"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Stats.Total != 1 {
		t.Errorf("expected 1 change (metadata ignored), got stats=%+v", r.Stats)
	}
}

func TestDiffWithLimits(t *testing.T) {
	left := []byte(`{"a":{"b":{"c":1}}}`)
	right := []byte(`{"a":{"b":{"c":2}}}`)
	r, err := Diff(left, right, WithLimits(Limits{MaxDepth: Ptr(2)}))
	if err != nil {
		t.Fatal(err)
	}
	if r.Equal {
		t.Error("expected not equal even with depth limit")
	}
}

func TestDiffWithFormat(t *testing.T) {
	left := []byte(`{"a":1}`)
	right := []byte(`{"a":2}`)
	r, err := Diff(left, right, WithFormat(FormatPatch))
	if err != nil {
		t.Fatal(err)
	}
	out, err := FormatResult(r, FormatPatch, false)
	if err != nil {
		t.Fatal(err)
	}
	var ops []map[string]any
	if err := json.Unmarshal(out, &ops); err != nil {
		t.Fatalf("invalid patch JSON: %v", err)
	}
	if len(ops) != 1 || ops[0]["op"] != "replace" {
		t.Errorf("unexpected patch: %s", out)
	}
}

func TestDiffRequestUnmarshal(t *testing.T) {
	data := []byte(`{"left":{"a":1},"right":{"a":2},"format":"patch"}`)
	var req DiffRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatal(err)
	}
	out, err := req.Execute()
	if err != nil {
		t.Fatal(err)
	}
	var ops []map[string]any
	if err := json.Unmarshal(out, &ops); err != nil {
		t.Fatalf("invalid output: %v", err)
	}
	if len(ops) != 1 {
		t.Errorf("expected 1 op, got %d", len(ops))
	}
}

func TestDiffRequestValidation(t *testing.T) {
	var req DiffRequest

	// Missing left
	err := json.Unmarshal([]byte(`{"right":{"a":1}}`), &req)
	if err == nil {
		t.Error("expected error for missing left")
	}

	// Missing right
	err = json.Unmarshal([]byte(`{"left":{"a":1}}`), &req)
	if err == nil {
		t.Error("expected error for missing right")
	}

	// Invalid format
	err = json.Unmarshal([]byte(`{"left":{"a":1},"right":{"a":2},"format":"bad"}`), &req)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestDiffOneShotMatchesSnapshot(t *testing.T) {
	left := []byte(`{"a":1,"b":{"c":[1,2,3]},"d":"hello"}`)
	right := []byte(`{"a":2,"b":{"c":[1,2,4]},"e":"new"}`)

	oneShot, err := Diff(left, right)
	if err != nil {
		t.Fatal(err)
	}

	snap, err := Compile(left)
	if err != nil {
		t.Fatal(err)
	}
	guided, err := snap.Diff(right)
	if err != nil {
		t.Fatal(err)
	}

	if oneShot.Equal != guided.Equal {
		t.Errorf("equal mismatch: oneShot=%v guided=%v", oneShot.Equal, guided.Equal)
	}
	if oneShot.Stats != guided.Stats {
		t.Errorf("stats mismatch: oneShot=%+v guided=%+v", oneShot.Stats, guided.Stats)
	}
	if len(oneShot.Changes) != len(guided.Changes) {
		t.Fatalf("change count mismatch: %d vs %d", len(oneShot.Changes), len(guided.Changes))
	}
	for i, oc := range oneShot.Changes {
		gc := guided.Changes[i]
		if oc.Path != gc.Path || oc.Type != gc.Type {
			t.Errorf("change[%d] mismatch: {%s %s} vs {%s %s}", i, oc.Path, oc.Type, gc.Path, gc.Type)
		}
	}
}

func TestPtr(t *testing.T) {
	p := Ptr(42)
	if *p != 42 {
		t.Errorf("Ptr(42) = %d, want 42", *p)
	}
}
