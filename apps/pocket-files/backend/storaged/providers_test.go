package storaged

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	s := Sandbox{map[string]string{"x": root}}
	for _, rel := range []string{"../outside", "escape/file"} {
		if _, err := s.Resolve("x", rel, true); err == nil {
			t.Errorf("accepted unsafe path %q", rel)
		}
	}
	if _, err := s.Resolve("x", "safe/new", true); err != nil {
		t.Fatalf("safe destination rejected: %v", err)
	}
}
func TestMockCreateRenameCopyMoveDelete(t *testing.T) {
	p, err := NewMockProvider()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(p.MockRoot("internal")))
	ctx := t.Context()
	if err = p.CreateDir(ctx, "internal", "Documents/Test"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(p.MockRoot("internal"), "Documents/Test/a.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = p.Copy(ctx, "internal", "Documents/Test/a.txt", "Documents/Test/copy.txt"); err != nil {
		t.Fatal(err)
	}
	if err = p.Move(ctx, "internal", "Documents/Test/copy.txt", "Documents/Test/moved.txt"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(p.MockRoot("internal"), "Documents/Test/moved.txt"))
	if err != nil || string(data) != "hello" {
		t.Fatalf("copy content %q err %v", data, err)
	}
	if err = p.Delete(ctx, "internal", "Documents/Test/moved.txt"); err != nil {
		t.Fatal(err)
	}
}
