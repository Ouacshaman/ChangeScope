package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ouacshaman/changescope/internal/differ"
)

// writeTempFile creates a temporary .go file with the given source content and
// returns its path. The file is cleaned up automatically via t.Cleanup.
func writeTempFile(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("writeTempFile: %v", err)
	}
	return path
}

// TestExtract_FunctionOverlapsRange verifies that a function whose body falls
// within the changed range is returned.
func TestExtract_FunctionOverlapsRange(t *testing.T) {
	src := `package fixture

func Hello() string {
	return "hello"
}
`
	path := writeTempFile(t, src)
	// func Hello spans lines 3-5 → range [3,5] should match
	symbols, err := Extract(path, []differ.LineRange{{Start: 3, End: 5}})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "Hello" {
		t.Errorf("Name = %q, want %q", symbols[0].Name, "Hello")
	}
	if symbols[0].Receiver != "" {
		t.Errorf("Receiver = %q, want empty", symbols[0].Receiver)
	}
}

// TestExtract_FunctionOutsideRange verifies that a function whose body does
// not intersect the changed range is excluded.
func TestExtract_FunctionOutsideRange(t *testing.T) {
	src := `package fixture

func Alpha() {}

func Beta() {}
`
	path := writeTempFile(t, src)
	// Alpha is on line 3; Beta is on line 5. Only change line 3.
	symbols, err := Extract(path, []differ.LineRange{{Start: 3, End: 3}})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "Alpha" {
		t.Errorf("Name = %q, want %q", symbols[0].Name, "Alpha")
	}
}

// TestExtract_PartialOverlap verifies that a function is matched when the
// changed range overlaps only the closing brace of the function.
func TestExtract_PartialOverlap(t *testing.T) {
	src := `package fixture

func Compute() int {
	x := 1
	return x
}
`
	path := writeTempFile(t, src)
	// func Compute spans lines 3-7; changed range is lines 6-7 (partial overlap)
	symbols, err := Extract(path, []differ.LineRange{{Start: 6, End: 7}})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "Compute" {
		t.Errorf("Name = %q, want %q", symbols[0].Name, "Compute")
	}
}

// TestExtract_MethodReceiver verifies that a method's receiver type name is
// captured correctly.
func TestExtract_MethodReceiver(t *testing.T) {
	src := `package fixture

type Service struct{}

func (s *Service) UpdateTask() error {
	return nil
}
`
	path := writeTempFile(t, src)
	// UpdateTask is on lines 5-7
	symbols, err := Extract(path, []differ.LineRange{{Start: 5, End: 7}})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "UpdateTask" {
		t.Errorf("Name = %q, want %q", symbols[0].Name, "UpdateTask")
	}
	if symbols[0].Receiver != "Service" {
		t.Errorf("Receiver = %q, want %q", symbols[0].Receiver, "Service")
	}
}

// TestExtract_MultipleFunctionsOnlyOneChanged verifies that when multiple
// functions are present only the one intersecting the changed range is returned.
func TestExtract_MultipleFunctionsOnlyOneChanged(t *testing.T) {
	src := `package fixture

func First() {}

func Second() {}

func Third() {}
`
	path := writeTempFile(t, src)
	// Second() is on line 5; range [5,5] should match only Second.
	symbols, err := Extract(path, []differ.LineRange{{Start: 5, End: 5}})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "Second" {
		t.Errorf("Name = %q, want %q", symbols[0].Name, "Second")
	}
}
