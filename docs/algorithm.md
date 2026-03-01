# Algorithm

## One-shot comparison

Recursive walk of two JSON trees:

```
walkCompare(left, right, path, depth):
    if depth > maxDepth → [replaced]
    if left == nil && right == nil → []
    if left == nil → [added]
    if right == nil → [removed]
    if typeOf(left) != typeOf(right) → [replaced]

    switch:
    case object → walkObjects(sorted key union, recurse per key)
    case array  → walkArrays(iterate max(len), positional compare)
    default     → if !scalarEqual(left, right) → [replaced]
```

### Complexity

- **Time**: O(n) where n is total nodes in both documents
- **Space**: O(d) stack depth where d is max nesting depth, plus O(c) for the changes list

## Snapshot-guided comparison

The key optimization: pre-compute FNV-1a hashes for every subtree, then compare hashes instead of values.

### Build phase

Post-order traversal of the JSON tree. Each node's hash is computed from its children's hashes:

```
buildNode(value):
    if scalar → hash(type_marker + value_bytes)
    if object → hash(sorted key:childHash pairs)
    if array  → hash(ordered childHash sequence)
```

### Compare phase

```
snapshotCompare(left, right):
    if left.hash == right.hash → [] (skip entire subtree)
    // Only recurse into subtrees with different hashes
    ...same walker structure as one-shot...
```

### Complexity

- **Build**: O(n) time and space for n nodes
- **Compare**: O(c × d) where c is changed nodes and d is average depth to changed nodes
- **Best case** (equal documents): O(1) — single hash comparison at root
- **Worst case** (all nodes changed): O(n) — degrades to full walk

### When to use snapshot mode

Snapshot mode wins when:
1. The baseline is reused across multiple comparisons
2. Documents are mostly identical (common in monitoring/drift detection)
3. Documents are large with sparse changes

The build cost (O(n) per document) is amortized across repeated diffs. For a single comparison, one-shot mode avoids the snapshot construction overhead.

## Scalar equality

`json.Number` values are compared by string representation, preserving precision:
- `1.0` != `1.00` (different representations)
- `42` == `42` (same representation)

Other scalars use Go's `==` operator (strings, bools, nil).
