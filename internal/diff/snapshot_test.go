package diff

import "testing"

func TestSnapshotHashStability(t *testing.T) {
	tree := mustParse(t, `{"a":1,"b":[2,3],"c":{"d":true}}`)
	s1 := BuildSnapshot(tree, Limits{}, nil)
	s2 := BuildSnapshot(tree, Limits{}, nil)
	if s1.root.hash != s2.root.hash {
		t.Error("same tree should produce same root hash")
	}
}

func TestSnapshotHashKeyOrderIndependent(t *testing.T) {
	// Both parse to the same logical structure; Go maps are unordered
	// but our hashing sorts keys, so the hash should be the same.
	tree1 := mustParse(t, `{"a":1,"b":2}`)
	tree2 := mustParse(t, `{"b":2,"a":1}`)
	s1 := BuildSnapshot(tree1, Limits{}, nil)
	s2 := BuildSnapshot(tree2, Limits{}, nil)
	if s1.root.hash != s2.root.hash {
		t.Error("key order should not affect hash")
	}
}

func TestSnapshotHashDifferentValues(t *testing.T) {
	tree1 := mustParse(t, `{"a":1}`)
	tree2 := mustParse(t, `{"a":2}`)
	s1 := BuildSnapshot(tree1, Limits{}, nil)
	s2 := BuildSnapshot(tree2, Limits{}, nil)
	if s1.root.hash == s2.root.hash {
		t.Error("different values should produce different hashes")
	}
}

func TestSnapshotDiffEqual(t *testing.T) {
	tree := mustParse(t, `{"users":[{"name":"Alice"},{"name":"Bob"}],"version":1}`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `{"users":[{"name":"Alice"},{"name":"Bob"}],"version":1}`)
	r := snap.Diff(target)
	if !r.Equal {
		t.Errorf("expected equal, got changes: %v", r.Changes)
	}
}

func TestSnapshotDiffChanged(t *testing.T) {
	tree := mustParse(t, `{"users":[{"name":"Alice"},{"name":"Bob"}],"version":1}`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `{"users":[{"name":"Alice"},{"name":"Charlie"}],"version":1}`)
	r := snap.Diff(target)
	if r.Equal {
		t.Error("expected not equal")
	}
	if r.Stats.Replaced != 1 || r.Stats.Total != 1 {
		t.Errorf("expected 1 replaced, got stats=%+v", r.Stats)
	}
	if r.Changes[0].Path != "$.users[1].name" {
		t.Errorf("expected path $.users[1].name, got %q", r.Changes[0].Path)
	}
}

func TestSnapshotSkipsUnchangedSubtree(t *testing.T) {
	// The users subtree is identical — snapshot should skip it entirely
	tree := mustParse(t, `{"users":[{"name":"Alice"}],"version":1}`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `{"users":[{"name":"Alice"}],"version":2}`)
	r := snap.Diff(target)
	if r.Stats.Replaced != 1 || r.Stats.Total != 1 {
		t.Errorf("expected exactly 1 change (version), got stats=%+v changes=%v", r.Stats, r.Changes)
	}
}

func TestSnapshotDiffAdded(t *testing.T) {
	tree := mustParse(t, `{"a":1}`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `{"a":1,"b":2}`)
	r := snap.Diff(target)
	if r.Stats.Added != 1 || r.Stats.Total != 1 {
		t.Errorf("expected 1 added, got stats=%+v", r.Stats)
	}
}

func TestSnapshotDiffRemoved(t *testing.T) {
	tree := mustParse(t, `{"a":1,"b":2}`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `{"a":1}`)
	r := snap.Diff(target)
	if r.Stats.Removed != 1 || r.Stats.Total != 1 {
		t.Errorf("expected 1 removed, got stats=%+v", r.Stats)
	}
}

func TestSnapshotDiffTypeMismatch(t *testing.T) {
	tree := mustParse(t, `{"a":1}`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `[1]`)
	r := snap.Diff(target)
	if r.Stats.Replaced != 1 {
		t.Errorf("expected 1 replaced for type mismatch, got stats=%+v", r.Stats)
	}
}

func TestSnapshotDiffNullTarget(t *testing.T) {
	tree := mustParse(t, `{"a":1}`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	r := snap.Diff(nil)
	// null target vs object baseline → replaced at root
	if r.Stats.Replaced != 1 {
		t.Errorf("expected 1 replaced, got stats=%+v", r.Stats)
	}
}

func TestSnapshotDiffArrayGrow(t *testing.T) {
	tree := mustParse(t, `[1,2]`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `[1,2,3]`)
	r := snap.Diff(target)
	if r.Stats.Added != 1 {
		t.Errorf("expected 1 added, got stats=%+v", r.Stats)
	}
}

func TestSnapshotDiffArrayShrink(t *testing.T) {
	tree := mustParse(t, `[1,2,3]`)
	snap := BuildSnapshot(tree, Limits{}, nil)

	target := mustParse(t, `[1,2]`)
	r := snap.Diff(target)
	if r.Stats.Removed != 1 {
		t.Errorf("expected 1 removed, got stats=%+v", r.Stats)
	}
}

func TestSnapshotImmutability(t *testing.T) {
	tree := mustParse(t, `{"a":1,"b":2}`)
	snap := BuildSnapshot(tree, Limits{}, nil)
	hashBefore := snap.root.hash

	// Diffing should not mutate the snapshot
	target := mustParse(t, `{"a":1,"b":3}`)
	snap.Diff(target)

	if snap.root.hash != hashBefore {
		t.Error("snapshot hash changed after Diff")
	}
}

// Verify snapshot produces same results as one-shot Compare.
func TestSnapshotMatchesCompare(t *testing.T) {
	left := mustParse(t, `{"users":[{"name":"Alice","age":30},{"name":"Bob"}],"version":1,"meta":{"k":"v"}}`)
	right := mustParse(t, `{"users":[{"name":"Alice","age":31},{"name":"Charlie"}],"version":2}`)

	oneShot := Compare(left, right, Limits{}, nil)
	snap := BuildSnapshot(left, Limits{}, nil)
	guided := snap.Diff(right)

	if oneShot.Equal != guided.Equal {
		t.Errorf("equal mismatch: oneShot=%v guided=%v", oneShot.Equal, guided.Equal)
	}
	if oneShot.Stats != guided.Stats {
		t.Errorf("stats mismatch: oneShot=%+v guided=%+v", oneShot.Stats, guided.Stats)
	}
	if len(oneShot.Changes) != len(guided.Changes) {
		t.Fatalf("change count mismatch: oneShot=%d guided=%d", len(oneShot.Changes), len(guided.Changes))
	}
	for i, oc := range oneShot.Changes {
		gc := guided.Changes[i]
		if oc.Path != gc.Path || oc.Type != gc.Type {
			t.Errorf("change[%d] mismatch: oneShot={%s %s} guided={%s %s}", i, oc.Path, oc.Type, gc.Path, gc.Type)
		}
	}
}

