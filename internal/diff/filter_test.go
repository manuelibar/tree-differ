package diff

import "testing"

func TestFilterNilAlwaysIncludes(t *testing.T) {
	f := NewFilter(nil, nil)
	if f != nil {
		t.Error("expected nil filter for empty inputs")
	}
}

func TestFilterIgnoreExactMatch(t *testing.T) {
	f := NewFilter(nil, []string{"$.metadata"})
	if !f.ShouldSkip("$.metadata") {
		t.Error("expected skip for exact ignore match")
	}
}

func TestFilterIgnoreChildPath(t *testing.T) {
	f := NewFilter(nil, []string{"$.metadata"})
	if !f.ShouldSkip("$.metadata.created") {
		t.Error("expected skip for child of ignored path")
	}
}

func TestFilterIgnoreNoMatch(t *testing.T) {
	f := NewFilter(nil, []string{"$.metadata"})
	if f.ShouldSkip("$.users") {
		t.Error("should not skip unrelated path")
	}
}

func TestFilterIgnorePartialKeyNoMatch(t *testing.T) {
	f := NewFilter(nil, []string{"$.meta"})
	if f.ShouldSkip("$.metadata") {
		t.Error("should not skip partial key match (meta vs metadata)")
	}
}

func TestFilterOnlyExactMatch(t *testing.T) {
	f := NewFilter([]string{"$.users"}, nil)
	if f.ShouldSkip("$.users") {
		t.Error("should not skip exact only match")
	}
}

func TestFilterOnlyChildPath(t *testing.T) {
	f := NewFilter([]string{"$.users"}, nil)
	if f.ShouldSkip("$.users.name") {
		t.Error("should not skip child of only path")
	}
}

func TestFilterOnlyParentPath(t *testing.T) {
	f := NewFilter([]string{"$.users.name"}, nil)
	// $ is on the way to $.users.name — should NOT skip
	if f.ShouldSkip("$.users") {
		t.Error("should not skip parent that leads to only path")
	}
}

func TestFilterOnlyUnrelatedPath(t *testing.T) {
	f := NewFilter([]string{"$.users"}, nil)
	if !f.ShouldSkip("$.settings") {
		t.Error("expected skip for path outside only filter")
	}
}

func TestFilterOnlyRootPath(t *testing.T) {
	f := NewFilter([]string{"$.users"}, nil)
	// Root "$" is on the way to "$.users" — but $ doesn't have the separator check
	// so this tests the edge case. Actually "$" is an ancestor of "$.users".
	// The only check: prefix starts with path and prefix[len(path)] == '.' or '['
	// "$.users" starts with "$" and "$.users"[1] == '.', so it should match.
	if f.ShouldSkip("$") {
		t.Error("should not skip root when only filter targets a descendant")
	}
}

func TestFilterCombinedOnlyAndIgnore(t *testing.T) {
	f := NewFilter([]string{"$.users"}, []string{"$.users.password"})
	if f.ShouldSkip("$.users") {
		t.Error("should not skip users (matches only)")
	}
	if f.ShouldSkip("$.users.name") {
		t.Error("should not skip users.name (inside only)")
	}
	if !f.ShouldSkip("$.users.password") {
		t.Error("expected skip for users.password (matches ignore)")
	}
	if !f.ShouldSkip("$.settings") {
		t.Error("expected skip for settings (outside only)")
	}
}

func TestFilterOnlyWithArray(t *testing.T) {
	f := NewFilter([]string{"$.users[0]"}, nil)
	if f.ShouldSkip("$.users[0]") {
		t.Error("should not skip exact array match")
	}
	if f.ShouldSkip("$.users[0].name") {
		t.Error("should not skip child of only array path")
	}
	if f.ShouldSkip("$.users") {
		t.Error("should not skip parent of only path")
	}
}

func TestFilterIntegrationWithCompare(t *testing.T) {
	left := mustParse(t, `{"users":{"name":"Alice"},"metadata":{"ts":"2024"}}`)
	right := mustParse(t, `{"users":{"name":"Bob"},"metadata":{"ts":"2025"}}`)

	f := NewFilter(nil, []string{"$.metadata"})
	r := Compare(left, right, Limits{}, f)
	if r.Stats.Total != 1 {
		t.Errorf("expected 1 change (metadata ignored), got stats=%+v changes=%v", r.Stats, r.Changes)
	}
	if r.Changes[0].Path != "$.users.name" {
		t.Errorf("expected path $.users.name, got %q", r.Changes[0].Path)
	}
}

func TestFilterOnlyIntegrationWithCompare(t *testing.T) {
	left := mustParse(t, `{"users":{"name":"Alice"},"settings":{"theme":"light"}}`)
	right := mustParse(t, `{"users":{"name":"Bob"},"settings":{"theme":"dark"}}`)

	f := NewFilter([]string{"$.users"}, nil)
	r := Compare(left, right, Limits{}, f)
	if r.Stats.Total != 1 {
		t.Errorf("expected 1 change (only users), got stats=%+v changes=%v", r.Stats, r.Changes)
	}
}
