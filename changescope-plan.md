# ChangeScope — MVP Implementation Plan

## Overview

ChangeScope is a Go CLI that reads unstaged Git changes in the current repository, identifies which Go symbols were changed, finds potential references to those symbols across the codebase, highlights related test files, and prints a readable terminal impact report.

**Approach:** Git diff → AST parse changed functions → text search for references → structured report.

**Reference search abstraction:** a thin `Searcher` interface is introduced so the text-grep implementation can later be swapped for `go/types`-based analysis without redesigning the pipeline.

**Left out of MVP deliberately:**
- `go/types` static analysis (no guaranteed call-graph resolution)
- Staged (index) or committed diff — unstaged only
- Multi-module repositories
- Non-Go files
- Any web UI, JSON/machine output, or config file
- CLI framework (Cobra etc.) — `os.Args` only
- Test function name extraction from snippets — showing related `_test.go` files is sufficient

---

## Package Structure

```
changescope/
├── go.mod
├── cmd/
│   └── changescope/
│       └── main.go            # CLI entry point — wires everything together
│                              # invoked as: go run ./cmd/changescope analyze
└── internal/
    ├── differ/
    │   └── differ.go          # Run `git diff`, parse unified diff into DiffResult
    ├── parser/
    │   └── parser.go          # AST-parse changed .go files, extract changed symbols
    ├── searcher/
    │   ├── searcher.go        # Searcher interface + Reference type
    │   └── grep.go            # Text/regex implementation of Searcher
    └── report/
        └── report.go          # Build and print the terminal impact report
```

---

## Key Types

```go
// differ
type LineRange struct {
    Start int
    End   int
}

type FileChange struct {
    Path   string
    Ranges []LineRange
}

type DiffResult struct {
    Files   []FileChange
    RawDiff string
}

// parser
type Symbol struct {
    Name     string // e.g. "UpdateTask"
    Receiver string // e.g. "TaskService" — empty for package-level funcs
    File     string // path within repo
}

// searcher
type Reference struct {
    File    string
    Line    int
    Snippet string // trimmed source line
    IsTest  bool   // true when file ends in _test.go
}

type Searcher interface {
    Find(root string, symbols []Symbol) ([]Reference, error)
}

// report
type Report struct {
    ChangedFiles   []string
    ChangedSymbols []Symbol
    References     []Reference
}
```

---

## Data Flow

```
git diff (unstaged)
      │
      ▼
differ.Run(repoRoot)     → DiffResult{Files[]FileChange, RawDiff}
      │                    Each FileChange carries Path + []LineRange
      │                    (parsed from @@ hunk headers inside differ)
      ▼
parser.Extract(filePath, []LineRange)
      │                  → []Symbol
      │                    AST walks FuncDecl nodes; keeps those whose
      │                    declaration or body intersects a changed range
      ▼
searcher.Find(root, []Symbol)
      │                  → []Reference
      │                    Walks repo; skips vendor/, .git/
      │                    Text-searches each .go file for symbol names
      │                    Marks _test.go files with IsTest:true
      │                    Results labelled "potential references"
      ▼
report.Print(w, Report)  → terminal output
```

---

## Sub-Tasks

### Sub-Task 1 — Project Scaffolding

**Intent:** Establish the directory tree and skeleton files so every subsequent sub-task has a clear home and `go build ./...` passes from the start.

**Expected Outcomes:**
- `cmd/changescope/main.go` exists with a stub `main()`.
- All `internal/` packages exist with a stub `.go` file declaring the correct package name.
- `go build ./...` and `go test ./...` succeed (zero tests, zero failures).

**Todo List:**
1. Create `cmd/changescope/main.go` — stub `main()` printing `"ChangeScope"`.
2. Create `internal/differ/differ.go`, `internal/parser/parser.go`, `internal/searcher/searcher.go`, `internal/searcher/grep.go`, `internal/report/report.go` — each declaring only its package name.
3. Verify `go build ./...` passes.

**Relevant Context:** `go.mod` already declares module `github.com/Ouacshaman/changescope`, Go 1.27.1. No external dependencies.

**Status:** [ ] pending

---

### Sub-Task 2 — `differ` Package

**Intent:** Capture the unstaged diff and parse it into structured per-file information including changed line ranges, so downstream packages never need to understand Git diff syntax.

**Expected Outcomes:**
- `LineRange`, `FileChange`, and `DiffResult` structs defined.
- `Run(repoRoot string) (DiffResult, error)` shells out to `git diff` (not `--cached`).
- `ParseDiff(raw string) DiffResult` parses `--- a/<path>` headers (filter `.go` only) and `@@ -old +new,count @@` hunk headers into `FileChange.Ranges`.
- Unit tests in `differ_test.go` use a fixture unified diff string — no shelling out in tests.

**Todo List:**
1. Define `LineRange`, `FileChange`, `DiffResult` in `differ.go`.
2. Implement `ParseDiff(raw string) DiffResult` — parse `--- a/` lines and `@@ +start,count @@` hunks; derive `LineRange{Start: start, End: start+count-1}` for each hunk; skip non-`.go` files.
3. Implement `Run(repoRoot string) (DiffResult, error)` using `exec.Command("git", "diff")` with `Dir` set to `repoRoot`; delegate to `ParseDiff`.
4. Write unit tests covering: single file single hunk, single file multiple hunks, multiple files, no Go files.

**Relevant Context:** Standard library — `os/exec`, `strings`, `bufio`, `strconv`. Unified diff `@@` format: `@@ -old_start,old_count +new_start,new_count @@`; use the `+` side for new-file line ranges.

**Status:** [ ] pending

---

### Sub-Task 3 — `parser` Package

**Intent:** Given a `.go` file path and the changed line ranges from the diff, use Go's standard AST tooling to identify which functions or methods were touched.

**Expected Outcomes:**
- `Symbol` struct defined with `Name`, `Receiver`, `File`.
- `Extract(filePath string, ranges []LineRange) ([]Symbol, error)` returns only symbols whose `FuncDecl` body intersects at least one changed range.
- Unit tests use `go/parser.ParseFile` on an in-memory source string (via `token.NewFileSet()` + a temp file or `ParseFile` with `src` parameter) to avoid filesystem coupling.

**Todo List:**
1. Define `Symbol` struct in `parser.go`.
2. Implement `Extract` — parse file with `go/parser.ParseFile`, walk `ast.FuncDecl` nodes, use `fset.Position` to get start/end lines, check intersection with any `LineRange`.
3. Capture receiver name from `FuncDecl.Recv` if present.
4. Write unit tests: function fully inside range, function partially overlapping range, function outside range, method with receiver.

**Relevant Context:** Standard library — `go/parser`, `go/ast`, `go/token`. Intersection check: `funcEnd >= rangeStart && funcStart <= rangeEnd`.

**Status:** [ ] pending

---

### Sub-Task 4 — `searcher` Package

**Intent:** Search the repository for text occurrences of each changed symbol name. Kept behind a minimal interface so the implementation can be replaced later.

**Expected Outcomes:**
- `Searcher` interface: `Find(root string, symbols []Symbol) ([]Reference, error)`.
- `Reference` struct: `File`, `Line`, `Snippet`, `IsTest`.
- `GrepSearcher` implements `Searcher` using `filepath.WalkDir` + line-by-line `strings.Contains`.
- Skips `vendor/` and `.git/` directories; only inspects `.go` files.
- Sets `IsTest = true` for files ending in `_test.go`.
- Unit tests use `t.TempDir()` with fixture `.go` files.

**Todo List:**
1. Define `Searcher` interface and `Reference` struct in `searcher.go`.
2. Implement `GrepSearcher` in `grep.go` — `WalkDir`, skip excluded dirs, open each `.go` file, scan line by line, record matches with trimmed snippet.
3. Ensure changed symbol's own source file is included in results if it genuinely references another changed symbol (i.e., do not unconditionally self-exclude).
4. Write unit tests: symbol found in non-test file, symbol found in test file, symbol not found, vendor dir skipped.

**Relevant Context:** Standard library — `path/filepath`, `os`, `strings`, `bufio`. `Symbol.Name` is the search term. Results are "potential references" — false positives are expected and acceptable.

**Status:** [ ] pending

---

### Sub-Task 5 — `report` Package

**Intent:** Assemble all findings into the readable terminal impact report shown in AGENTS.md.

**Expected Outcomes:**
- `Report` struct defined.
- `Print(w io.Writer, r Report)` writes the formatted report.
- De-duplicates file paths in "Potential impact".
- Separates test files from non-test files in output.
- Related `_test.go` files listed under "Related tests" (file paths only — no test function name parsing).
- Unit test: `Print` into `bytes.Buffer`, assert section headers and key lines present.

**Todo List:**
1. Define `Report` struct in `report.go`.
2. Implement `Print` — emit "Changed files", "Changed symbols", "Potential impact" (de-duped non-test files), "Related tests" (de-duped test files).
3. Write unit tests with a hand-crafted `Report`, assert output contains expected sections and entries.

**Relevant Context:** Standard library — `io`, `fmt`, `strings`, `sort`. Output format from AGENTS.md example. Test files are identified via `Reference.IsTest`.

**Status:** [ ] pending

---

### Sub-Task 6 — `cmd/changescope` Integration

**Intent:** Wire all packages into a working end-to-end CLI with a single `analyze` subcommand.

**Expected Outcomes:**
- `go run ./cmd/changescope analyze [path]` (defaulting to `.`) produces the full impact report.
- Graceful error messages on common failures: not a git repo, no unstaged changes, no Go files changed.
- End-to-end smoke test: run against the ChangeScope repo itself after introducing a deliberate test change.

**Todo List:**
1. Parse `os.Args`: expect `analyze` subcommand and optional positional `[path]` argument; default `path` to `"."` when omitted.
2. Call `differ.Run` → loop `parser.Extract` per `FileChange` → collect all `Symbol`s → `searcher.GrepSearcher.Find` → build `report.Report` → `report.Print(os.Stdout, ...)`.
3. **Exit codes:** use `os.Exit(1)` only for genuine execution or analysis errors (e.g. `git` not found, file unreadable). Print a clear "No unstaged Go changes found." message and exit `0` when the diff is empty or contains no `.go` files.
4. Handle other errors with `fmt.Fprintf(os.Stderr, "error: %v\n", err)` and `os.Exit(1)`.
5. Manual smoke test: introduce a one-line change to a function in the repo, run `go run ./cmd/changescope analyze`, verify the report is correct.

**Relevant Context:** All internal packages. `differ.DiffResult.Files` provides both paths and `[]LineRange` — no diff re-parsing needed in `main`.

**Status:** [ ] pending

---

## Implementation Order

1. **Sub-Task 1** — Scaffolding
2. **Sub-Task 2** — `differ`
3. **Sub-Task 3** — `parser`
4. **Sub-Task 4** — `searcher`
5. **Sub-Task 5** — `report`
6. **Sub-Task 6** — Integration

Each sub-task must have passing tests before the next begins.

---

## Major Risks

| Risk | Mitigation |
|------|------------|
| `git diff` output varies by Git version or config | Parse only the standard `--- a/<path>` and `@@ +start,count @@` headers; test with a fixture string |
| AST line numbers off-by-one vs diff line ranges | Use `go/token.FileSet` positions consistently; unit-test with a known fixture |
| Text search false positives (same name in comment, string, unrelated type) | Label all results "potential references" in the report; acceptable for MVP |
| Changed function has same name as a common identifier | Acceptable noise; `go/types` can fix this post-hackathon |
| Diff contains no changed Go files | Return early with a clean "No Go changes detected" message |
| `@@ +start @@` (count omitted when count=1) | Handle the `+start` form (no comma) as `count=1` in `ParseDiff` |

---

## Deliberately Out of Scope for MVP

- `go/types` call-graph resolution
- Staged (`git diff --cached`) or committed diffs
- Non-Go languages
- Interface-structural impact (changed interface method affecting all implementors)
- Multi-module repositories
- JSON / machine-readable output
- Configuration file
- CLI framework (Cobra, urfave/cli, etc.)
- Test function name extraction from file content
- Web UI
