package diff

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustParse(t *testing.T, s string) any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCompareEqualScalars(t *testing.T) {
	left := mustParse(t, `"hello"`)
	right := mustParse(t, `"hello"`)
	r := Compare(left, right, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal")
	}
	if r.Stats.Total != 0 {
		t.Errorf("expected 0 changes, got %d", r.Stats.Total)
	}
}

func TestCompareReplacedScalar(t *testing.T) {
	left := mustParse(t, `"hello"`)
	right := mustParse(t, `"world"`)
	r := Compare(left, right, Limits{}, nil)
	if r.Equal {
		t.Error("expected not equal")
	}
	if len(r.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(r.Changes))
	}
	c := r.Changes[0]
	if c.Path != "$" || c.Type != Replaced {
		t.Errorf("got path=%q type=%q", c.Path, c.Type)
	}
}

func TestCompareNumbers(t *testing.T) {
	left := mustParse(t, `42`)
	right := mustParse(t, `42`)
	r := Compare(left, right, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal for same numbers")
	}

	right2 := mustParse(t, `43`)
	r2 := Compare(left, right2, Limits{}, nil)
	if r2.Equal {
		t.Error("expected not equal for different numbers")
	}
}

func TestCompareNumberPrecision(t *testing.T) {
	// json.Number preserves string representation
	left := mustParse(t, `1.0`)
	right := mustParse(t, `1.00`)
	r := Compare(left, right, Limits{}, nil)
	// "1.0" != "1.00" as strings — this is intentional for precision preservation
	if r.Equal {
		t.Error("expected not equal: json.Number treats 1.0 and 1.00 as different representations")
	}
}

func TestCompareBooleans(t *testing.T) {
	left := mustParse(t, `true`)
	right := mustParse(t, `false`)
	r := Compare(left, right, Limits{}, nil)
	if r.Equal {
		t.Error("expected not equal")
	}
}

func TestCompareNulls(t *testing.T) {
	left := mustParse(t, `null`)
	right := mustParse(t, `null`)
	r := Compare(left, right, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal for null == null")
	}
}

func TestCompareLeftNull(t *testing.T) {
	right := mustParse(t, `"hello"`)
	r := Compare(nil, right, Limits{}, nil)
	if r.Equal {
		t.Error("expected not equal")
	}
	if len(r.Changes) != 1 || r.Changes[0].Type != Added {
		t.Errorf("expected 1 added change, got %v", r.Changes)
	}
}

func TestCompareRightNull(t *testing.T) {
	left := mustParse(t, `"hello"`)
	r := Compare(left, nil, Limits{}, nil)
	if r.Equal {
		t.Error("expected not equal")
	}
	if len(r.Changes) != 1 || r.Changes[0].Type != Removed {
		t.Errorf("expected 1 removed change, got %v", r.Changes)
	}
}

func TestCompareBothNull(t *testing.T) {
	r := Compare(nil, nil, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal for nil == nil")
	}
}

func TestCompareTypeMismatch(t *testing.T) {
	left := mustParse(t, `"hello"`)
	right := mustParse(t, `42`)
	r := Compare(left, right, Limits{}, nil)
	if r.Equal {
		t.Error("expected not equal")
	}
	if len(r.Changes) != 1 || r.Changes[0].Type != Replaced {
		t.Errorf("expected replaced, got %v", r.Changes)
	}
}

func TestCompareObjectToArray(t *testing.T) {
	left := mustParse(t, `{"a":1}`)
	right := mustParse(t, `[1]`)
	r := Compare(left, right, Limits{}, nil)
	if len(r.Changes) != 1 || r.Changes[0].Type != Replaced {
		t.Errorf("expected replaced for object→array, got %v", r.Changes)
	}
}

func TestCompareEqualObjects(t *testing.T) {
	left := mustParse(t, `{"name":"Alice","age":30}`)
	right := mustParse(t, `{"age":30,"name":"Alice"}`)
	r := Compare(left, right, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal regardless of key order")
	}
}

func TestCompareObjectAddedKey(t *testing.T) {
	left := mustParse(t, `{"a":1}`)
	right := mustParse(t, `{"a":1,"b":2}`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Added != 1 || r.Stats.Total != 1 {
		t.Errorf("expected 1 added, got stats=%+v", r.Stats)
	}
	if r.Changes[0].Path != "$.b" {
		t.Errorf("expected path $.b, got %q", r.Changes[0].Path)
	}
}

func TestCompareObjectRemovedKey(t *testing.T) {
	left := mustParse(t, `{"a":1,"b":2}`)
	right := mustParse(t, `{"a":1}`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Removed != 1 || r.Stats.Total != 1 {
		t.Errorf("expected 1 removed, got stats=%+v", r.Stats)
	}
}

func TestCompareObjectReplacedValue(t *testing.T) {
	left := mustParse(t, `{"a":1}`)
	right := mustParse(t, `{"a":2}`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Replaced != 1 || r.Stats.Total != 1 {
		t.Errorf("expected 1 replaced, got stats=%+v", r.Stats)
	}
}

func TestCompareNestedObjects(t *testing.T) {
	left := mustParse(t, `{"user":{"name":"Alice","settings":{"theme":"light"}}}`)
	right := mustParse(t, `{"user":{"name":"Alice","settings":{"theme":"dark"}}}`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Replaced != 1 || r.Stats.Total != 1 {
		t.Errorf("expected 1 replaced, got stats=%+v", r.Stats)
	}
	if r.Changes[0].Path != "$.user.settings.theme" {
		t.Errorf("expected path $.user.settings.theme, got %q", r.Changes[0].Path)
	}
}

func TestCompareEqualArrays(t *testing.T) {
	left := mustParse(t, `[1,2,3]`)
	right := mustParse(t, `[1,2,3]`)
	r := Compare(left, right, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal")
	}
}

func TestCompareArrayReplacedElement(t *testing.T) {
	left := mustParse(t, `[1,2,3]`)
	right := mustParse(t, `[1,99,3]`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Replaced != 1 {
		t.Errorf("expected 1 replaced, got stats=%+v", r.Stats)
	}
	if r.Changes[0].Path != "$[1]" {
		t.Errorf("expected path $[1], got %q", r.Changes[0].Path)
	}
}

func TestCompareArrayShorter(t *testing.T) {
	left := mustParse(t, `[1,2,3]`)
	right := mustParse(t, `[1,2]`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Removed != 1 {
		t.Errorf("expected 1 removed, got stats=%+v", r.Stats)
	}
}

func TestCompareArrayLonger(t *testing.T) {
	left := mustParse(t, `[1,2]`)
	right := mustParse(t, `[1,2,3]`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Added != 1 {
		t.Errorf("expected 1 added, got stats=%+v", r.Stats)
	}
}

func TestCompareEmptyObject(t *testing.T) {
	left := mustParse(t, `{}`)
	right := mustParse(t, `{}`)
	r := Compare(left, right, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal for empty objects")
	}
}

func TestCompareEmptyArray(t *testing.T) {
	left := mustParse(t, `[]`)
	right := mustParse(t, `[]`)
	r := Compare(left, right, Limits{}, nil)
	if !r.Equal {
		t.Error("expected equal for empty arrays")
	}
}

func TestCompareEmptyToPopulated(t *testing.T) {
	left := mustParse(t, `{}`)
	right := mustParse(t, `{"a":1}`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Added != 1 {
		t.Errorf("expected 1 added, got stats=%+v", r.Stats)
	}
}

func TestCompareMixedChanges(t *testing.T) {
	left := mustParse(t, `{"name":"Alice","legacy":true}`)
	right := mustParse(t, `{"name":"Bob","theme":"dark"}`)
	r := Compare(left, right, Limits{}, nil)
	if r.Stats.Replaced != 1 || r.Stats.Added != 1 || r.Stats.Removed != 1 || r.Stats.Total != 3 {
		t.Errorf("expected 1 replaced + 1 added + 1 removed, got stats=%+v", r.Stats)
	}
}

func TestCompareDepthLimit(t *testing.T) {
	maxD := 2
	left := mustParse(t, `{"a":{"b":{"c":1}}}`)
	right := mustParse(t, `{"a":{"b":{"c":2}}}`)
	r := Compare(left, right, Limits{MaxDepth: &maxD}, nil)
	// Should detect change but at truncated depth
	if r.Equal {
		t.Error("expected not equal")
	}
}

func TestCompareComplexDocument(t *testing.T) {
	left := mustParse(t, `{
		"users": [
			{"name": "Alice", "age": 30},
			{"name": "Bob", "age": 25}
		],
		"version": 1,
		"metadata": {"created": "2024-01-01"}
	}`)
	right := mustParse(t, `{
		"users": [
			{"name": "Alice", "age": 31},
			{"name": "Charlie", "age": 25}
		],
		"version": 2,
		"metadata": {"created": "2024-01-01"}
	}`)
	r := Compare(left, right, Limits{}, nil)
	// age changed (30→31), name changed (Bob→Charlie), version changed (1→2)
	if r.Stats.Replaced != 3 {
		t.Errorf("expected 3 replaced, got stats=%+v", r.Stats)
	}
}
