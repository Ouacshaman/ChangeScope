package report

import (
	"strings"
	"testing"

	"github.com/Ouacshaman/changescope/internal/parser"
	"github.com/Ouacshaman/changescope/internal/searcher"
)

func TestPrint_empty(t *testing.T) {
	var buf strings.Builder
	Print(&buf, Report{})
	got := buf.String()

	mustContain(t, got, "ChangeScope Impact Report")
	mustContain(t, got, "Changed files:")
	mustContain(t, got, "(none)")
}

func TestPrint_withData(t *testing.T) {
	r := Report{
		ChangedFiles: []string{
			"internal/task/service.go",
			"internal/task/service.go", // duplicate — should appear once
		},
		ChangedSymbols: []parser.Symbol{
			{Name: "UpdateTask", File: "internal/task/service.go"},
			{Name: "UpdateTask", File: "internal/task/service.go"}, // duplicate
			{Name: "DeleteTask", Receiver: "TaskService", File: "internal/task/service.go"},
		},
		References: []searcher.Reference{
			{File: "internal/http/task_handler.go", Line: 42, IsTest: false},
			{File: "internal/http/task_handler.go", Line: 55, IsTest: false}, // dup file
			{File: "internal/task/service_test.go", Line: 10, IsTest: true},
		},
	}

	var buf strings.Builder
	Print(&buf, r)
	got := buf.String()

	mustContain(t, got, "ChangeScope Impact Report")

	// Changed files — deduped
	mustContain(t, got, "internal/task/service.go")
	if strings.Count(got, "internal/task/service.go") != 1 {
		t.Errorf("expected changed file to appear once; got:\n%s", got)
	}

	// Changed symbols
	mustContain(t, got, "UpdateTask")
	mustContain(t, got, "TaskService.DeleteTask")

	// Potential impact — non-test reference file
	mustContain(t, got, "Potential impact:")
	mustContain(t, got, "internal/http/task_handler.go")

	// Related tests — test reference file
	mustContain(t, got, "Related tests:")
	mustContain(t, got, "internal/task/service_test.go")

	// Test file must NOT appear under Potential impact
	impactSection := sectionBetween(t, got, "Potential impact:", "Related tests:")
	if strings.Contains(impactSection, "_test.go") {
		t.Errorf("_test.go file appeared under Potential impact; impact section:\n%s", impactSection)
	}

	// Suggested review
	mustContain(t, got, "Suggested review:")
	mustContain(t, got, "Review callers of")
}

func TestPrint_noChangedSymbols(t *testing.T) {
	r := Report{
		ChangedFiles: []string{"cmd/main.go"},
	}
	var buf strings.Builder
	Print(&buf, r)
	got := buf.String()

	mustContain(t, got, "Changed symbols:")
	mustContain(t, got, "(none)")
}

func TestPrint_sortedOutput(t *testing.T) {
	r := Report{
		ChangedFiles: []string{"z_file.go", "a_file.go"},
		References: []searcher.Reference{
			{File: "z_handler.go", IsTest: false},
			{File: "a_handler.go", IsTest: false},
		},
	}
	var buf strings.Builder
	Print(&buf, r)
	got := buf.String()

	idxA := strings.Index(got, "a_file.go")
	idxZ := strings.Index(got, "z_file.go")
	if idxA > idxZ {
		t.Errorf("changed files not sorted: a_file.go should appear before z_file.go")
	}

	idxAH := strings.Index(got, "a_handler.go")
	idxZH := strings.Index(got, "z_handler.go")
	if idxAH > idxZH {
		t.Errorf("impact files not sorted: a_handler.go should appear before z_handler.go")
	}
}

// mustContain fails the test if sub is not present in s.
func mustContain(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("expected output to contain %q; got:\n%s", sub, s)
	}
}

// sectionBetween returns the text in s between the first occurrence of start
// and the first occurrence of end after that. Returns empty string if either
// marker is absent.
func sectionBetween(t *testing.T, s, start, end string) string {
	t.Helper()
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	s = s[i+len(start):]
	j := strings.Index(s, end)
	if j < 0 {
		return s
	}
	return s[:j]
}
