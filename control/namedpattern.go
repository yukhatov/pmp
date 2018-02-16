package control

import (
	"goji.io"
	"goji.io/pat"
)

// NamedPattern is an extension to goji.Pattern that allows us to give a pattern/route a simple name
// This can be used by other middleware to get the name of the route that was matched.
// This is useful for things like logging and stats collection so we have the name of the operation that was being performed.
type NamedPattern struct {
	*pat.Pattern
	Name string
}

// NewNamedPattern creates a new named pattern compatible with goji.Pattern
func NewNamedPattern(name string, pat *pat.Pattern) goji.Pattern {
	return &NamedPattern{
		pat,
		name,
	}
}
