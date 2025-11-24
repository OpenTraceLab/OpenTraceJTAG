// Package kicadsexp provides a lightweight streaming S-expression parser
// optimized for KiCad board files. Unlike general-purpose sexp libraries,
// this parser can handle arbitrarily large files by streaming.
package kicadsexp

import (
	"io"
)

// Sexp represents an S-expression node.
// It can be either a leaf (atom) or a list (cons cell).
type Sexp interface {
	// IsLeaf returns true if this is an atom (not a list)
	IsLeaf() bool

	// LeafCount returns the number of elements in a list (0 for atoms)
	LeafCount() int

	// Head returns the first element of a list (nil for atoms)
	Head() Sexp

	// Tail returns the rest of the list after the first element (nil for atoms)
	Tail() Sexp

	// String returns the string representation
	String() string
}

// Symbol represents an atomic symbol (string, number, identifier)
type Symbol string

func (s Symbol) IsLeaf() bool      { return true }
func (s Symbol) LeafCount() int    { return 1 }
func (s Symbol) Head() Sexp        { return s }
func (s Symbol) Tail() Sexp        { return nil }
func (s Symbol) String() string    { return string(s) }

// List represents a list of S-expressions
type List struct {
	elements []Sexp
}

func (l *List) IsLeaf() bool { return false }

func (l *List) LeafCount() int {
	return len(l.elements)
}

func (l *List) Head() Sexp {
	if len(l.elements) == 0 {
		return nil
	}
	return l.elements[0]
}

func (l *List) Tail() Sexp {
	if len(l.elements) <= 1 {
		return nil
	}
	return &List{elements: l.elements[1:]}
}

func (l *List) String() string {
	result := "("
	for i, elem := range l.elements {
		if i > 0 {
			result += " "
		}
		result += elem.String()
	}
	result += ")"
	return result
}

// Get returns the element at the given index
func (l *List) Get(index int) Sexp {
	if index < 0 || index >= len(l.elements) {
		return nil
	}
	return l.elements[index]
}

// Len returns the number of elements in the list
func (l *List) Len() int {
	return len(l.elements)
}

// Parse parses S-expressions from an io.Reader.
// This is the main entry point that's compatible with existing code.
func Parse(r io.Reader) ([]Sexp, error) {
	parser := NewParser(r)
	return parser.ParseAll()
}

// ParseString parses S-expressions from a string (convenience function)
func ParseString(s string) ([]Sexp, error) {
	return Parse(stringReader(s))
}

// stringReader wraps a string as an io.Reader
type stringReader string

func (s stringReader) Read(p []byte) (n int, err error) {
	if len(s) == 0 {
		return 0, io.EOF
	}
	n = copy(p, s)
	s = s[n:]
	if n < len(p) {
		err = io.EOF
	}
	return
}
