package searcher

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ouacshaman/changescope/internal/parser"
)

// GrepSearcher implements Searcher using filesystem walking and text scanning.
// It inspects every .go file under root (excluding vendor/ and .git/) and
// records lines that contain any of the provided symbol names.
type GrepSearcher struct{}

// Find walks the directory tree at root and returns all lines in .go files
// that contain the name of at least one symbol. Directories named "vendor" or
// ".git" are skipped entirely. Results are labelled as potential references —
// no semantic analysis is performed.
func (GrepSearcher) Find(root string, symbols []parser.Symbol) ([]Reference, error) {
	// Build a deduplicated list of names to search for.
	names := uniqueNames(symbols)
	if len(names) == 0 {
		return nil, nil
	}

	var refs []Reference

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories.
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only inspect .go files.
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		isTest := strings.HasSuffix(path, "_test.go")

		found, err := scanFile(path, names, isTest)
		if err != nil {
			return err
		}
		refs = append(refs, found...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return refs, nil
}

// scanFile opens the file at path and returns a Reference for every line that
// contains at least one of the target names.
func scanFile(path string, names []string, isTest bool) ([]Reference, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var refs []Reference
	lineNum := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		for _, name := range names {
			if strings.Contains(line, name) {
				refs = append(refs, Reference{
					File:    path,
					Line:    lineNum,
					Snippet: trimmed,
					IsTest:  isTest,
				})
				break // one reference per line is enough
			}
		}
	}
	return refs, scanner.Err()
}

// uniqueNames returns a deduplicated slice of symbol names.
func uniqueNames(symbols []parser.Symbol) []string {
	seen := make(map[string]bool, len(symbols))
	var names []string
	for _, s := range symbols {
		if s.Name != "" && !seen[s.Name] {
			seen[s.Name] = true
			names = append(names, s.Name)
		}
	}
	return names
}
