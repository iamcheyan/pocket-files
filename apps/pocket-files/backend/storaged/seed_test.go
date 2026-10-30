package storaged

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGeneratedPDFIsStructurallyValid(t *testing.T) {
	pdf := minimalPDF("Pocket Files mock sample - test")
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatalf("missing PDF header")
	}
	if !bytes.HasSuffix(pdf, []byte("%%EOF\n")) {
		t.Fatalf("missing PDF trailer")
	}
	if !bytes.Contains(pdf, []byte("/Type /Catalog")) || !bytes.Contains(pdf, []byte("startxref")) {
		t.Fatalf("PDF missing required structures")
	}
	// xref offsets must point at "N 0 obj" markers.
	idx := bytes.Index(pdf, []byte("\nxref\n"))
	if idx < 0 {
		t.Fatalf("missing xref table")
	}
	lines := bytes.Split(pdf[idx+len("\nxref\n"):], []byte("\n"))
	if len(lines) < 3 {
		t.Fatalf("xref table too short")
	}
	// First xref line is "0 <count>".
	fields := strings.Fields(string(lines[0]))
	if len(fields) != 2 {
		t.Fatalf("xref header = %q, want \"0 <count>\"", lines[0])
	}
	var count int
	if _, err := fmtSscan(fields[1], &count); err != nil {
		t.Fatalf("xref count unparsable: %v", err)
	}
	if count != 6 {
		t.Fatalf("xref count = %d, want 6 (5 objects + free entry)", count)
	}
	for _, line := range lines[2 : 2+5] {
		var offset int
		if _, err := fmtSscan(string(line[:10]), &offset); err != nil {
			t.Fatalf("bad xref offset line %q: %v", line, err)
		}
		if !bytes.HasPrefix(pdf[offset:], []byte(" obj")) {
			// offsets point at the object number, then " 0 obj"
			if !bytes.Contains(pdf[offset:offset+16], []byte("0 obj")) {
				t.Fatalf("xref offset %d does not point at an object", offset)
			}
		}
	}
}

func TestGeneratedPNGHeader(t *testing.T) {
	if !bytes.HasPrefix(minimalPNG, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		t.Fatalf("PNG signature missing")
	}
	if !bytes.Contains(minimalPNG, []byte("IHDR")) || !bytes.Contains(minimalPNG, []byte("IEND")) {
		t.Fatalf("PNG chunks missing")
	}
}

func TestSeededMockVolumeContainsPreviewableFiles(t *testing.T) {
	root := t.TempDir()
	if err := SeedMockVolume(root, "internal", 0); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"Books/Field Notes.pdf", "Images/Sample Pixel.png", "Documents/Welcome.txt"} {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s missing: %v", rel, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", rel)
		}
	}
}

func fmtSscan(s string, out *int) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	*out = n
	return 1, nil
}
