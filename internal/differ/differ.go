// Package differ shells out to git diff and parses the unified diff output
// into structured per-file change information.
package differ

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

// LineRange is an inclusive range of line numbers in the new version of a file.
type LineRange struct {
	Start int
	End   int
}

// FileChange records which line ranges were changed in a single Go file.
type FileChange struct {
	Path   string
	Ranges []LineRange
}

// DiffResult is the parsed output of git diff.
type DiffResult struct {
	Files   []FileChange
	RawDiff string
}

// Run executes git diff (unstaged changes) inside repoRoot and returns the
// parsed result. An empty diff is not an error.
func Run(repoRoot string) (DiffResult, error) {
	cmd := exec.Command("git", "diff")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return DiffResult{}, err
	}
	raw := string(out)
	result := ParseDiff(raw)
	result.RawDiff = raw
	return result, nil
}

// ParseDiff parses a unified diff string and returns a DiffResult containing
// only .go files. It does not execute any external commands.
func ParseDiff(raw string) DiffResult {
	var result DiffResult
	result.RawDiff = raw

	var current *FileChange

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()

		// "--- a/<path>" marks the start of a new file section.
		if strings.HasPrefix(line, "--- ") {
			current = nil
			continue
		}

		if strings.HasPrefix(line, "+++ ") {
			current = nil

			if !strings.HasPrefix(line, "+++ b/") {
				continue
			}

			path := strings.TrimPrefix(line, "+++ b/")
			if strings.HasSuffix(path, ".go") {
				result.Files = append(result.Files, FileChange{Path: path})
				current = &result.Files[len(result.Files)-1]
			}

			continue
		}

		// Hunk header: @@ -old_start[,old_count] +new_start[,new_count] @@
		if strings.HasPrefix(line, "@@") && current != nil {
			lr, ok := parseHunkHeader(line)
			if ok {
				current.Ranges = append(current.Ranges, lr)
			}
		}
	}

	return result
}

// parseHunkHeader extracts the new-file LineRange from a unified diff hunk
// header of the form:  @@ -a[,b] +c[,d] @@ ...
// If the count is omitted it defaults to 1.
func parseHunkHeader(line string) (LineRange, bool) {
	// Find the +new_start[,new_count] token between the @@ markers.
	// Format: "@@ -a,b +c,d @@" or "@@ -a +c @@"
	first := strings.Index(line, "@@")
	if first < 0 {
		return LineRange{}, false
	}
	rest := line[first+2:]
	second := strings.Index(rest, "@@")
	if second < 0 {
		return LineRange{}, false
	}
	inner := strings.TrimSpace(rest[:second]) // e.g. "-1,5 +3,10" or "-1 +3"

	fields := strings.Fields(inner)
	for _, f := range fields {
		if !strings.HasPrefix(f, "+") {
			continue
		}
		newPart := f[1:] // strip leading "+"
		start, count, err := parseStartCount(newPart)
		if err != nil {
			return LineRange{}, false
		}
		end := start + count - 1
		if end < start {
			end = start
		}
		return LineRange{Start: start, End: end}, true
	}
	return LineRange{}, false
}

// parseStartCount parses "start" or "start,count" and returns the two values.
// If count is absent it is treated as 1.
func parseStartCount(s string) (start, count int, err error) {
	parts := strings.SplitN(s, ",", 2)
	start, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	if len(parts) == 1 {
		return start, 1, nil
	}
	count, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return start, count, nil
}
