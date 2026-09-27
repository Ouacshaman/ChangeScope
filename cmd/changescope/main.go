// Command changescope analyses the blast radius of unstaged Go changes in a
// Git repository.
//
// Usage:
//
//	changescope analyze [path]
//
// path defaults to "." when omitted.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ouacshaman/changescope/internal/differ"
	"github.com/Ouacshaman/changescope/internal/parser"
	"github.com/Ouacshaman/changescope/internal/report"
	"github.com/Ouacshaman/changescope/internal/searcher"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: changescope analyze [path]")
		os.Exit(1)
	}

	switch args[0] {
	case "analyze":
		if len(args) > 2 {
			fmt.Fprintln(os.Stderr, "usage: changescope analyze [path]")
			os.Exit(1)
		}
		repoRoot := "."
		if len(args) == 2 {
			repoRoot = args[1]
		}
		if err := runAnalyze(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: changescope analyze [path]")
		os.Exit(1)
	}
}

func runAnalyze(repoRoot string) error {
	// 1. Run git diff to find unstaged changes.
	diff, err := differ.Run(repoRoot)
	if err != nil {
		return fmt.Errorf("git diff failed: %w", err)
	}

	if len(diff.Files) == 0 {
		fmt.Println("No unstaged Go file changes detected.")
		return nil
	}

	// 2. Collect changed file paths and extract changed symbols via AST.
	var changedFiles []string
	var allSymbols []parser.Symbol

	for _, fc := range diff.Files {
		changedFiles = append(changedFiles, fc.Path)

		// Resolve relative to repo root so parser opens the correct file.
		absPath := filepath.Join(repoRoot, fc.Path)
		syms, err := parser.Extract(absPath, fc.Ranges)
		if err != nil {
			// Non-fatal: the file might have been deleted or be unparseable.
			fmt.Fprintf(os.Stderr, "warning: could not parse %s: %v\n", fc.Path, err)
			continue
		}
		allSymbols = append(allSymbols, syms...)
	}

	if len(allSymbols) == 0 {
		// Still print the report so the user sees which files changed.
		report.Print(os.Stdout, report.Report{
			ChangedFiles: changedFiles,
		})
		return nil
	}

	// 3. Search the whole repository for references to changed symbols.
	gs := searcher.GrepSearcher{}
	refs, err := gs.Find(repoRoot, allSymbols)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// 4. Print the impact report.
	report.Print(os.Stdout, report.Report{
		ChangedFiles:   changedFiles,
		ChangedSymbols: allSymbols,
		References:     refs,
	})

	return nil
}
