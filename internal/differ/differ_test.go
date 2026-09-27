package differ

import (
	"testing"
)

// fixture builds a minimal unified diff string.
func fixture(files ...string) string {
	return join(files...)
}

func join(parts ...string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}

// fileSection returns a unified diff section for a single file with the given
// hunk headers already included.
func fileSection(path string, hunks ...string) string {
	s := "diff --git a/" + path + " b/" + path + "\n"
	s += "--- a/" + path + "\n"
	s += "+++ b/" + path + "\n"
	for _, h := range hunks {
		s += h + "\n"
	}
	return s
}

// hunk returns a standard hunk header line.
func hunk(oldStart, oldCount, newStart, newCount int) string {
	return "@@ -" + itoa(oldStart) + "," + itoa(oldCount) +
		" +" + itoa(newStart) + "," + itoa(newCount) + " @@"
}

// hunkNoCount returns a hunk header where both counts are omitted (both == 1).
func hunkNoCount(oldStart, newStart int) string {
	return "@@ -" + itoa(oldStart) + " +" + itoa(newStart) + " @@"
}

func itoa(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{digits[n%10]}, buf...)
		n /= 10
	}
	return string(buf)
}

// Test 1: one Go file, one hunk.
func TestParseDiff_OneFileOneHunk(t *testing.T) {
	raw := fixture(fileSection("internal/task/service.go", hunk(1, 5, 3, 7)))
	result := ParseDiff(raw)

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	f := result.Files[0]
	if f.Path != "internal/task/service.go" {
		t.Errorf("path = %q, want %q", f.Path, "internal/task/service.go")
	}
	if len(f.Ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(f.Ranges))
	}
	r := f.Ranges[0]
	// new_start=3, new_count=7 → Start=3, End=9
	if r.Start != 3 || r.End != 9 {
		t.Errorf("range = {%d,%d}, want {3,9}", r.Start, r.End)
	}
}

// Test 2: one file, multiple hunks.
func TestParseDiff_OneFileMultipleHunks(t *testing.T) {
	raw := fixture(fileSection("pkg/foo.go",
		hunk(1, 3, 1, 3),
		hunk(20, 4, 20, 6),
	))
	result := ParseDiff(raw)

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	f := result.Files[0]
	if len(f.Ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(f.Ranges))
	}
	// hunk 1: new_start=1, count=3 → {1,3}
	if f.Ranges[0].Start != 1 || f.Ranges[0].End != 3 {
		t.Errorf("range[0] = {%d,%d}, want {1,3}", f.Ranges[0].Start, f.Ranges[0].End)
	}
	// hunk 2: new_start=20, count=6 → {20,25}
	if f.Ranges[1].Start != 20 || f.Ranges[1].End != 25 {
		t.Errorf("range[1] = {%d,%d}, want {20,25}", f.Ranges[1].Start, f.Ranges[1].End)
	}
}

// Test 3: multiple Go files.
func TestParseDiff_MultipleGoFiles(t *testing.T) {
	raw := fixture(
		fileSection("a/a.go", hunk(5, 2, 5, 4)),
		fileSection("b/b.go", hunk(10, 1, 10, 1)),
	)
	result := ParseDiff(raw)

	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}
	if result.Files[0].Path != "a/a.go" {
		t.Errorf("files[0].Path = %q, want %q", result.Files[0].Path, "a/a.go")
	}
	if result.Files[1].Path != "b/b.go" {
		t.Errorf("files[1].Path = %q, want %q", result.Files[1].Path, "b/b.go")
	}
}

// Test 4: non-Go files are ignored.
func TestParseDiff_NonGoFilesIgnored(t *testing.T) {
	raw := fixture(
		fileSection("README.md", hunk(1, 1, 1, 1)),
		fileSection("config.yaml", hunk(3, 2, 3, 2)),
		fileSection("main.go", hunk(1, 1, 1, 2)),
	)
	result := ParseDiff(raw)

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file (main.go only), got %d", len(result.Files))
	}
	if result.Files[0].Path != "main.go" {
		t.Errorf("path = %q, want %q", result.Files[0].Path, "main.go")
	}
}

// Test 5: omitted hunk count defaults to 1.
func TestParseDiff_OmittedCountDefaultsToOne(t *testing.T) {
	raw := fixture(fileSection("cmd/main.go", hunkNoCount(7, 12)))
	result := ParseDiff(raw)

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	r := result.Files[0].Ranges
	if len(r) != 1 {
		t.Fatalf("expected 1 range, got %d", len(r))
	}
	// new_start=12, count=1 → Start=12, End=12
	if r[0].Start != 12 || r[0].End != 12 {
		t.Errorf("range = {%d,%d}, want {12,12}", r[0].Start, r[0].End)
	}
}

func TestParseDiff_NewNonGoFileDoesNotAffectPreviousGoFile(t *testing.T) {
	raw := fileSection("main.go", hunk(1, 1, 1, 2)) +
		"diff --git a/README.md b/README.md\n" +
		"new file mode 100644\n" +
		"--- /dev/null\n" +
		"+++ b/README.md\n" +
		"@@ -0,0 +1,3 @@\n"

	result := ParseDiff(raw)

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 Go file, got %d", len(result.Files))
	}

	if len(result.Files[0].Ranges) != 1 {
		t.Fatalf(
			"expected main.go to have 1 range, got %d",
			len(result.Files[0].Ranges),
		)
	}
}
