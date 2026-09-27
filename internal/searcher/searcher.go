// Package searcher defines the Searcher interface and related types for finding
// symbol references across a repository.
package searcher

import "github.com/Ouacshaman/changescope/internal/parser"

// Reference represents a potential occurrence of a changed symbol in the
// repository. Results are text-based and may include false positives.
type Reference struct {
	File    string // file path where the symbol name was found
	Line    int    // 1-based line number within File
	Snippet string // trimmed source line
	IsTest  bool   // true when File ends in _test.go
}

// Searcher finds potential references to the given symbols within a repository.
type Searcher interface {
	Find(root string, symbols []parser.Symbol) ([]Reference, error)
}
