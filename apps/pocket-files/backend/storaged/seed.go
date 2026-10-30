package storaged

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// mockSample is generated content for one mock volume. Everything here is
// synthetic: small files written from Go strings, never copies of a real device
// or of the development host's files.
type mockSample struct {
	rel  string
	body []byte
}

func sampleSet(id string, index int) []mockSample {
	tag := fmt.Sprintf("Pocket Files mock sample for %s.\n", id)
	return []mockSample{
		{"Books/Field Notes.pdf", minimalPDF(fmt.Sprintf("Pocket Files mock sample - %s", id))},
		{"Books/Reading List.md", []byte("# Reading list\n\n" + tag + "\n- Placeholder entry one\n- Placeholder entry two\n")},
		{"Music/Ambient Sketch.mp3", []byte("ID3mock-" + tag)},
		{"ROMs/Demo Cartridge.gba", []byte("mockrom-" + tag)},
		{"ROMs/Demo Cartridge.gb", []byte("mockrom-" + tag)},
		{"Documents/Notes.txt", []byte("Notes for " + id + "\n\n" + tag)},
		{"Documents/Welcome.txt", []byte("Pocket Files mock storage. This is generated temporary data.\n")},
		{"Downloads/Example Archive.zip", []byte("mockzip-" + tag)},
		{"Documents/Scratch/ideas.txt", []byte("Idea list\n\n" + tag)},
		{"Videos/Scratch/clip-two.mp4", []byte("mockmp4-two-" + tag)},
	}
}

// minimalPDF builds a one-page, text-only PDF with a correct cross-reference
// table. It exists so the preview pipeline can be exercised end to end with a
// document a real PDF viewer accepts; it is still clearly labelled as mock
// content on the page itself.
func minimalPDF(title string) []byte {
	escaped := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(title)
	content := fmt.Sprintf("BT /F1 16 Tf 72 720 Td (%s) Tj ET", escaped)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var builder strings.Builder
	builder.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = builder.Len()
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteString(" 0 obj\n")
		builder.WriteString(body)
		builder.WriteString("\nendobj\n")
	}
	xref := builder.Len()
	builder.WriteString("xref\n0 ")
	builder.WriteString(strconv.Itoa(len(objects) + 1))
	builder.WriteString("\n0000000000 65535 f \n")
	for _, offset := range offsets {
		builder.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	builder.WriteString("trailer\n<< /Size ")
	builder.WriteString(strconv.Itoa(len(objects) + 1))
	builder.WriteString(" /Root 1 0 R >>\nstartxref\n")
	builder.WriteString(strconv.Itoa(xref))
	builder.WriteString("\n%%EOF\n")
	return []byte(builder.String())
}

// minimalPNG is a valid 64x64 PNG with five colour bands and a white disc. It
// is generated here rather than copied so mock image previews are genuinely
// renderable and visibly not a photograph.
var minimalPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x40,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x25, 0x0B, 0xE6, 0x89, 0x00, 0x00, 0x00,
	0xB5, 0x49, 0x44, 0x41, 0x54, 0x78, 0xDA, 0xED, 0xD9, 0xB1, 0x0D, 0x02,
	0x31, 0x10, 0x44, 0x51, 0xD7, 0x47, 0x76, 0xD9, 0x49, 0x84, 0xD7, 0xC4,
	0xF5, 0x40, 0x01, 0xF4, 0x40, 0x07, 0x34, 0x42, 0x7C, 0x64, 0x74, 0x31,
	0xC4, 0x48, 0x48, 0x5E, 0x84, 0xBD, 0x5E, 0xAF, 0xFE, 0x68, 0x1A, 0x78,
	0x99, 0xBD, 0x53, 0xB6, 0xF3, 0x61, 0xE9, 0xE9, 0x7E, 0xB1, 0xF4, 0x75,
	0x5B, 0x2C, 0x5D, 0xAF, 0x4F, 0x4B, 0x1F, 0xEB, 0x5E, 0x6D, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xD0, 0x0A, 0x20, 0x5B,
	0x22, 0x02, 0xF4, 0x7B, 0xA2, 0x00, 0xF4, 0x5F, 0x06, 0x03, 0xD4, 0x22,
	0x63, 0x00, 0x6A, 0x1D, 0x57, 0x80, 0xFA, 0xC4, 0x09, 0xA0, 0x9E, 0x01,
	0x50, 0x03, 0xA8, 0x7F, 0x3A, 0x02, 0xE4, 0x15, 0x00, 0xDF, 0x00, 0xF2,
	0x0D, 0x80, 0xCF, 0x6A, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xF0, 0x1A, 0x4D, 0x03, 0xE0, 0x4B, 0x19, 0x00, 0x30,
	0xFD, 0x55, 0x82, 0xC3, 0x56, 0x0C, 0xC0, 0xF4, 0xB7, 0xD1, 0x0C, 0xD7,
	0xE9, 0x0C, 0xFB, 0x40, 0x92, 0x85, 0x26, 0xC3, 0x46, 0x96, 0x64, 0xA5,
	0x64, 0xE8, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x4C, 0x08,
	0x78, 0x03, 0x65, 0x0F, 0x3A, 0x0A, 0x88, 0xF4, 0xBC, 0x24, 0x00, 0x00,
	0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

func SeedMockVolume(root, id string, index int) error {
	if os.Getenv("POCKET_FILES_MOCK_BIG") == "1" {
		// Optional large sample so transfer progress can be observed end to end.
		// Off by default to keep test and startup time short.
		big := filepath.Join(root, "Videos", "Large Sample.mp4")
		file, err := os.Create(big)
		if err != nil {
			return err
		}
		if err = file.Truncate(48 << 20); err != nil {
			file.Close()
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
	}
	samples := append(sampleSet(id, index),
		mockSample{"Images/Sample Pixel.png", minimalPNG},
	)
	for _, sample := range samples {
		full := filepath.Join(root, filepath.FromSlash(sample.rel))
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(full, sample.body, 0600); err != nil {
			return err
		}
	}
	return nil
}
