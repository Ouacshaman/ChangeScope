package searcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ouacshaman/changescope/internal/parser"
)

// writeFile is a helper that creates a file inside dir with the given content.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", name, err)
	}
	return path
}

var gs GrepSearcher

// TestFind_ReferenceInProductionFile verifies that a symbol name appearing in
// a non-test .go file is reported with IsTest=false.
func TestFind_ReferenceInProductionFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "handler.go", `package main

func main() {
	UpdateTask()
}
`)
	symbols := []parser.Symbol{{Name: "UpdateTask"}}
	refs, err := gs.Find(root, symbols)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected at least one reference, got none")
	}
	found := false
	for _, r := range refs {
		if r.IsTest {
			t.Errorf("unexpected IsTest=true for file %s", r.File)
		}
		if r.Snippet == "UpdateTask()" {
			found = true
		}
	}
	if !found {
		t.Error("expected snippet \"UpdateTask()\" not found in results")
	}
}

// TestFind_ReferenceInTestFile verifies that a symbol name appearing in a
// _test.go file is reported with IsTest=true.
func TestFind_ReferenceInTestFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "service_test.go", `package main

import "testing"

func TestUpdateTask(t *testing.T) {
	UpdateTask()
}
`)
	symbols := []parser.Symbol{{Name: "UpdateTask"}}
	refs, err := gs.Find(root, symbols)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected at least one reference, got none")
	}
	for _, r := range refs {
		if !r.IsTest {
			t.Errorf("expected IsTest=true for %s line %d, got false", r.File, r.Line)
		}
	}
}

// TestFind_MissingReference verifies that when the symbol name does not appear
// in any file, an empty (non-nil) result is returned without error.
func TestFind_MissingReference(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "other.go", `package main

func Hello() {}
`)
	symbols := []parser.Symbol{{Name: "UpdateTask"}}
	refs, err := gs.Find(root, symbols)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 references, got %d", len(refs))
	}
}

// TestFind_VendorIgnored verifies that .go files inside a vendor/ directory
// are not scanned.
func TestFind_VendorIgnored(t *testing.T) {
	root := t.TempDir()
	vendorDir := filepath.Join(root, "vendor", "pkg")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Put the symbol reference only inside vendor/.
	writeFile(t, vendorDir, "lib.go", `package pkg

func UpdateTask() {}
`)
	symbols := []parser.Symbol{{Name: "UpdateTask"}}
	refs, err := gs.Find(root, symbols)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	for _, r := range refs {
		if filepath.ToSlash(r.File) != filepath.ToSlash(filepath.Join(vendorDir, "lib.go")) {
			continue
		}
		t.Errorf("vendor file should be ignored, but got reference: %s:%d", r.File, r.Line)
	}
	// No references from vendor means we should have zero results.
	if len(refs) != 0 {
		t.Errorf("expected 0 references (vendor skipped), got %d", len(refs))
	}
}

// TestFind_NonGoFileIgnored verifies that files not ending in .go are skipped.
func TestFind_NonGoFileIgnored(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", `# docs

UpdateTask is a function.
`)
	writeFile(t, root, "config.yaml", `function: UpdateTask
`)
	symbols := []parser.Symbol{{Name: "UpdateTask"}}
	refs, err := gs.Find(root, symbols)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 references (non-Go files ignored), got %d", len(refs))
	}
}
