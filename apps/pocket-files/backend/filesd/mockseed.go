package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// mockSample is generated content for one mock volume. Everything here is
// synthetic: tiny placeholder files whose bytes say what they are, never copies
// of a real device or of the development host's files.
type mockSample struct {
	rel  string
	body []byte
}

func sampleSet(id string, index int) []mockSample {
	tag := fmt.Sprintf("Pocket Files mock sample for %s.\n", id)
	return []mockSample{
		{"Books/Field Notes.pdf", []byte("%PDF-1.4\n% " + tag + "This placeholder is not a real PDF.\n")},
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

func seedMockVolume(root, id string, index int) error {
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
	for _, sample := range sampleSet(id, index) {
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
