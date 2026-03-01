package diff

import (
	"encoding/json"
	"testing"
)

func sampleResult() *Result {
	return &Result{
		Equal: false,
		Stats: Stats{Added: 1, Removed: 1, Replaced: 1, Total: 3},
		Changes: []Change{
			{Path: "$.name", Type: Replaced, From: "Alice", To: "Bob"},
			{Path: "$.settings.theme", Type: Added, Value: "dark"},
			{Path: "$.legacy", Type: Removed, Value: true},
		},
	}
}

func TestFormatDelta(t *testing.T) {
	r := sampleResult()
	out, err := FormatResult(r, FormatDelta, false)
	if err != nil {
		t.Fatal(err)
	}
	var parsed Result
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.Equal {
		t.Error("expected equal=false")
	}
	if parsed.Stats.Total != 3 {
		t.Errorf("expected total=3, got %d", parsed.Stats.Total)
	}
	if len(parsed.Changes) != 3 {
		t.Errorf("expected 3 changes, got %d", len(parsed.Changes))
	}
}

func TestFormatDeltaPretty(t *testing.T) {
	r := sampleResult()
	out, err := FormatResult(r, FormatDelta, true)
	if err != nil {
		t.Fatal(err)
	}
	// Pretty output should contain newlines
	if len(out) == 0 {
		t.Error("empty output")
	}
	// Should still be valid JSON
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestFormatPatch(t *testing.T) {
	r := sampleResult()
	out, err := FormatResult(r, FormatPatch, false)
	if err != nil {
		t.Fatal(err)
	}
	var ops []patchOp
	if err := json.Unmarshal(out, &ops); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(ops) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(ops))
	}
	// First op: replace /name
	if ops[0].Op != "replace" || ops[0].Path != "/name" {
		t.Errorf("op[0] = %+v", ops[0])
	}
	// Second op: add /settings/theme
	if ops[1].Op != "add" || ops[1].Path != "/settings/theme" {
		t.Errorf("op[1] = %+v", ops[1])
	}
	// Third op: remove /legacy
	if ops[2].Op != "remove" || ops[2].Path != "/legacy" {
		t.Errorf("op[2] = %+v", ops[2])
	}
}

func TestFormatMerge(t *testing.T) {
	r := sampleResult()
	out, err := FormatResult(r, FormatMerge, false)
	if err != nil {
		t.Fatal(err)
	}
	var patch map[string]any
	if err := json.Unmarshal(out, &patch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// name should be "Bob" (replaced)
	if patch["name"] != "Bob" {
		t.Errorf("name = %v, want Bob", patch["name"])
	}
	// legacy should be null (removed)
	if patch["legacy"] != nil {
		t.Errorf("legacy = %v, want nil", patch["legacy"])
	}
	// settings.theme should be "dark" (added)
	settings, ok := patch["settings"].(map[string]any)
	if !ok {
		t.Fatalf("settings not a map: %v", patch["settings"])
	}
	if settings["theme"] != "dark" {
		t.Errorf("settings.theme = %v, want dark", settings["theme"])
	}
}

func TestFormatStat(t *testing.T) {
	r := sampleResult()
	out, err := FormatResult(r, FormatStat, false)
	if err != nil {
		t.Fatal(err)
	}
	var stat statOutput
	if err := json.Unmarshal(out, &stat); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if stat.Equal {
		t.Error("expected equal=false")
	}
	if stat.Stats.Total != 3 {
		t.Errorf("total = %d, want 3", stat.Stats.Total)
	}
}

func TestFormatPaths(t *testing.T) {
	r := sampleResult()
	out, err := FormatResult(r, FormatPaths, false)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	if err := json.Unmarshal(out, &paths); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
	// Should be sorted
	for i := 1; i < len(paths); i++ {
		if paths[i] < paths[i-1] {
			t.Errorf("paths not sorted: %v", paths)
			break
		}
	}
}

func TestFormatEqual(t *testing.T) {
	r := &Result{Equal: true, Stats: Stats{}}
	out, err := FormatResult(r, FormatDelta, false)
	if err != nil {
		t.Fatal(err)
	}
	var parsed Result
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !parsed.Equal {
		t.Error("expected equal=true")
	}
}

func TestParseFormat(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  Format
		err   bool
	}{
		{"delta", FormatDelta, false},
		{"patch", FormatPatch, false},
		{"merge", FormatMerge, false},
		{"stat", FormatStat, false},
		{"paths", FormatPaths, false},
		{"", FormatDelta, false},
		{"unknown", "", true},
	} {
		got, err := ParseFormat(tc.input)
		if tc.err && err == nil {
			t.Errorf("ParseFormat(%q): expected error", tc.input)
		}
		if !tc.err && err != nil {
			t.Errorf("ParseFormat(%q): unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestJsonPathToPointer(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"$.name", "/name"},
		{"$.settings.theme", "/settings/theme"},
		{"$.users[0].name", "/users/0/name"},
		{"$", ""},
		{"$.a[0][1]", "/a/0/1"},
	} {
		got := jsonPathToPointer(tc.input)
		if got != tc.want {
			t.Errorf("jsonPathToPointer(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestJsonPathToPointerEscaping(t *testing.T) {
	// RFC 6901: ~ → ~0, / → ~1
	r := &Result{
		Changes: []Change{
			{Path: `$.a/b`, Type: Added, Value: 1},
		},
	}
	out, err := FormatResult(r, FormatPatch, false)
	if err != nil {
		t.Fatal(err)
	}
	var ops []patchOp
	json.Unmarshal(out, &ops)
	if ops[0].Path != "/a~1b" {
		t.Errorf("expected /a~1b, got %q", ops[0].Path)
	}
}
