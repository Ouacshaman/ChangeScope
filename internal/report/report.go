// Package report builds and prints the ChangeScope terminal impact report.
package report

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ouacshaman/changescope/internal/parser"
	"github.com/Ouacshaman/changescope/internal/searcher"
)

// Report holds all the data gathered by the analysis pipeline.
type Report struct {
	// ChangedFiles is the set of repository-relative paths that git diff reported.
	ChangedFiles []string

	// ChangedSymbols are the Go functions/methods identified inside changed ranges.
	ChangedSymbols []parser.Symbol

	// References are the potential callers/uses found by the searcher.
	References []searcher.Reference
}

// Print writes a formatted impact report to w. File paths are de-duplicated
// and sorted. References whose file name ends in _test.go are listed under
// "Related tests"; all others appear under "Potential impact".
func Print(w io.Writer, r Report) {
	fmt.Fprintln(w, "ChangeScope Impact Report")
	fmt.Fprintln(w)

	// --- Changed files ---
	fmt.Fprintln(w, "Changed files:")
	files := dedupSorted(r.ChangedFiles)
	if len(files) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, f := range files {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}
	fmt.Fprintln(w)

	// --- Changed symbols ---
	fmt.Fprintln(w, "Changed symbols:")
	symNames := symbolNames(r.ChangedSymbols)
	if len(symNames) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, s := range symNames {
			fmt.Fprintf(w, "  %s\n", s)
		}
	}
	fmt.Fprintln(w)

	// Partition references into test and non-test groups.
	impactFiles, testFiles := partitionRefs(r.References)

	// --- Potential impact ---
	fmt.Fprintln(w, "Potential impact:")
	if len(impactFiles) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, f := range impactFiles {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}
	fmt.Fprintln(w)

	// --- Related tests ---
	fmt.Fprintln(w, "Related tests:")
	if len(testFiles) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, f := range testFiles {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}
	fmt.Fprintln(w)

	// --- Suggested review ---
	fmt.Fprintln(w, "Suggested review:")
	if len(r.ChangedSymbols) > 0 {
		for _, name := range symNames {
			fmt.Fprintf(w, "  - Review callers of %s\n", name)
		}
	}
	if len(testFiles) > 0 {
		fmt.Fprintln(w, "  - Run related package tests")
	}
	if hasGoFiles(impactFiles) {
		fmt.Fprintln(w, "  - Review affected integration points")
	}
}

// dedupSorted returns a sorted, de-duplicated copy of ss.
func dedupSorted(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// symbolNames returns a de-duplicated, sorted list of display names for the
// given symbols.  A method is shown as "Receiver.Name"; a function as "Name".
func symbolNames(syms []parser.Symbol) []string {
	seen := make(map[string]struct{}, len(syms))
	var out []string
	for _, s := range syms {
		var display string
		if s.Receiver != "" {
			display = s.Receiver + "." + s.Name
		} else {
			display = s.Name
		}
		if _, ok := seen[display]; !ok {
			seen[display] = struct{}{}
			out = append(out, display)
		}
	}
	sort.Strings(out)
	return out
}

// partitionRefs splits references into (non-test file paths, test file paths),
// each de-duplicated and sorted.
func partitionRefs(refs []searcher.Reference) (impact, tests []string) {
	impactSeen := make(map[string]struct{})
	testSeen := make(map[string]struct{})

	for _, ref := range refs {
		// Normalise to forward-slash for consistency on all platforms.
		f := filepath.ToSlash(ref.File)
		if ref.IsTest {
			if _, ok := testSeen[f]; !ok {
				testSeen[f] = struct{}{}
				tests = append(tests, f)
			}
		} else {
			if _, ok := impactSeen[f]; !ok {
				impactSeen[f] = struct{}{}
				impact = append(impact, f)
			}
		}
	}

	sort.Strings(impact)
	sort.Strings(tests)
	return impact, tests
}

// hasGoFiles reports whether any path in ss ends in ".go".
func hasGoFiles(ss []string) bool {
	for _, s := range ss {
		if strings.HasSuffix(s, ".go") {
			return true
		}
	}
	return false
}
