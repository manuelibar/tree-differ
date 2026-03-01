package diff

import "fmt"

// ChangeType classifies the kind of difference between two JSON values.
type ChangeType string

const (
	Added    ChangeType = "added"
	Removed  ChangeType = "removed"
	Replaced ChangeType = "replaced"
)

// Change represents a single difference between two JSON documents.
// Path uses JSONPath notation (e.g. "$.users[0].name").
type Change struct {
	Path  string     `json:"path"`
	Type  ChangeType `json:"type"`
	From  any        `json:"from,omitempty"`
	To    any        `json:"to,omitempty"`
	Value any        `json:"value,omitempty"`
}

// Stats summarises the number of changes by type.
type Stats struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Replaced int `json:"replaced"`
	Total    int `json:"total"`
}

// Result holds the complete output of a diff operation.
type Result struct {
	Equal   bool     `json:"equal"`
	Stats   Stats    `json:"stats"`
	Changes []Change `json:"changes,omitempty"`
}

// NewResult builds a Result from a list of changes.
func NewResult(changes []Change) *Result {
	var s Stats
	for _, c := range changes {
		switch c.Type {
		case Added:
			s.Added++
		case Removed:
			s.Removed++
		case Replaced:
			s.Replaced++
		}
	}
	s.Total = s.Added + s.Removed + s.Replaced
	return &Result{
		Equal:   s.Total == 0,
		Stats:   s,
		Changes: changes,
	}
}

// Limits controls safety bounds for traversal.
type Limits struct {
	MaxDepth *int
}

const DefaultMaxDepth = 1000

// EffectiveMaxDepth returns the max depth to use, applying defaults.
func (l Limits) EffectiveMaxDepth() int {
	if l.MaxDepth == nil {
		return DefaultMaxDepth
	}
	if *l.MaxDepth == 0 {
		return 0 // explicitly disabled
	}
	return *l.MaxDepth
}

// DepthError reports that a JSON document exceeds the maximum allowed nesting depth.
type DepthError struct {
	Path  string
	Depth int
	Max   int
}

func (e *DepthError) Error() string {
	return fmt.Sprintf("maximum depth %d exceeded at %s (depth %d)", e.Max, e.Path, e.Depth)
}
