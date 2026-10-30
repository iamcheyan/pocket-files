package storaged

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Device struct {
	ID, Name, Kind, Status string
	Mock, ReadOnly         bool
	Total, Used            int64
	Root                   string
}
type Entry struct {
	ID, StorageID, Name, Parent string
	Kind                        string
	Size                        int64
	Modified, Created           time.Time
	Extension                   string
}
type Provider interface {
	Discover(context.Context) ([]Device, error)
	Mount(context.Context, string) error
	Unmount(context.Context, string) error
	Browse(context.Context, string, string) ([]Entry, error)
	Search(context.Context, string, string) ([]Entry, error)
	Stat(context.Context, string, string) (Entry, error)
	Copy(context.Context, string, string, string) error
	Move(context.Context, string, string, string) error
	Delete(context.Context, string, string) error
	CreateDir(context.Context, string, string) error
	Trash(context.Context, string, string) (string, error)
	Restore(context.Context, string, string, string) error
	AvailableSpace(context.Context, string) (int64, int64, error)
}

type Sandbox struct{ roots map[string]string }

func (s Sandbox) Resolve(storage, rel string, allowMissing bool) (string, error) {
	root, ok := s.roots[storage]
	if !ok {
		return "", fmt.Errorf("unknown storage")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute paths are forbidden")
	}
	clean := filepath.Clean(rel)
	if clean == "." {
		clean = ""
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes storage root")
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(base, clean)
	cur := candidate
	for {
		real, e := filepath.EvalSymlinks(cur)
		if e == nil {
			if real != base && !strings.HasPrefix(real, base+string(filepath.Separator)) {
				return "", fmt.Errorf("symlink escapes storage root")
			}
			break
		}
		if !allowMissing || !errors.Is(e, os.ErrNotExist) {
			return "", e
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", e
		}
		cur = parent
	}
	return candidate, nil
}

type MockProvider struct {
	roots   map[string]string
	devices []Device
}

func NewMockProvider() (*MockProvider, error) {
	base, err := os.MkdirTemp("", "pocket-files-mock-")
	if err != nil {
		return nil, err
	}
	p := &MockProvider{roots: map[string]string{}}
	defs := []struct {
		id, name, kind string
		total          int64
	}{
		{"internal", "Internal", "internal", 128 << 30},
		{"tf-a", "TF-A", "tf", 64 << 30},
		{"tf-b", "TF-B", "tf", 128 << 30},
		{"usb", "USB", "usb", 256 << 30},
	}
	for _, d := range defs {
		root := filepath.Join(base, d.id)
		if err = os.MkdirAll(root, 0700); err != nil {
			return nil, err
		}
		p.roots[d.id] = root
		for _, n := range []string{"Books", "Music", "Videos", "ROMs", "Documents", "Downloads"} {
			if err = os.Mkdir(filepath.Join(root, n), 0700); err != nil {
				return nil, err
			}
		}
		if err = SeedMockVolume(root, d.id, len(p.devices)); err != nil {
			return nil, err
		}
		p.devices = append(p.devices, Device{ID: d.id, Name: d.name, Kind: d.kind, Status: "ready", Mock: true, Total: d.total, Root: root})
	}
	return p, nil
}

func (p *MockProvider) Discover(context.Context) ([]Device, error) {
	return append([]Device(nil), p.devices...), nil
}
func (p *MockProvider) Mount(context.Context, string) error   { return nil }
func (p *MockProvider) Unmount(context.Context, string) error { return nil }
func (p *MockProvider) root(id string) (string, error) {
	r, ok := p.roots[id]
	if !ok {
		return "", fmt.Errorf("unknown storage")
	}
	return r, nil
}

func getTimes(info os.FileInfo) (time.Time, time.Time) {
	mod := info.ModTime()
	var ctime time.Time
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		mod = time.Unix(st.Mtim.Sec, st.Mtim.Nsec)
		ctime = time.Unix(st.Ctim.Sec, st.Ctim.Nsec)
	}
	if ctime.IsZero() {
		ctime = mod
	}
	return mod, ctime
}

func safeOS(rel, action string, e error) error {
	if errors.Is(e, os.ErrNotExist) {
		return fmt.Errorf("%s %q: not found", action, rel)
	}
	if errors.Is(e, os.ErrPermission) {
		return fmt.Errorf("%s %q: permission denied", action, rel)
	}
	if errors.Is(e, os.ErrExist) {
		return fmt.Errorf("%s %q: already exists", action, rel)
	}
	return fmt.Errorf("%s %q: io error", action, rel)
}

func randID() (string, error) {
	b := make([]byte, 12)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}

func (p *MockProvider) entries(id, rel string) ([]Entry, error) {
	if _, e := p.root(id); e != nil {
		return nil, e
	}
	full, e := (Sandbox{p.roots}).Resolve(id, rel, false)
	if e != nil {
		return nil, e
	}
	items, e := os.ReadDir(full)
	if e != nil {
		return nil, safeOS(rel, "read", e)
	}
	out := make([]Entry, 0, len(items))
	for _, it := range items {
		info, e := it.Info()
		if e != nil {
			return nil, safeOS(rel, "stat", e)
		}
		kind := "file"
		if it.IsDir() {
			kind = "directory"
		}
		mod, ctime := getTimes(info)
		out = append(out, Entry{
			ID:        filepath.ToSlash(filepath.Join(rel, it.Name())),
			StorageID: id, Name: it.Name(), Parent: rel, Kind: kind,
			Size: info.Size(), Modified: mod, Created: ctime,
			Extension: strings.TrimPrefix(filepath.Ext(it.Name()), "."),
		})
	}
	return out, nil
}

func (p *MockProvider) Browse(_ context.Context, id, rel string) ([]Entry, error) {
	return p.entries(id, rel)
}

func (p *MockProvider) Search(_ context.Context, id, q string) ([]Entry, error) {
	all, e := p.entries(id, "")
	if e != nil {
		return nil, e
	}
	var out []Entry
	ql := strings.ToLower(q)
	for _, x := range all {
		if strings.Contains(strings.ToLower(x.Name), ql) {
			out = append(out, x)
		} else if x.Kind == "directory" {
			sub, se := p.entries(id, x.ID)
			if se == nil {
				for _, y := range sub {
					if strings.Contains(strings.ToLower(y.Name), ql) {
						out = append(out, y)
					}
				}
			}
		}
	}
	return out, nil
}

func (p *MockProvider) Stat(_ context.Context, id, rel string) (Entry, error) {
	full, e := (Sandbox{p.roots}).Resolve(id, rel, false)
	if e != nil {
		return Entry{}, e
	}
	info, e := os.Stat(full)
	if e != nil {
		return Entry{}, safeOS(rel, "stat", e)
	}
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	mod, ctime := getTimes(info)
	return Entry{ID: rel, StorageID: id, Name: filepath.Base(rel), Kind: kind, Size: info.Size(), Modified: mod, Created: ctime, Extension: strings.TrimPrefix(filepath.Ext(rel), ".")}, nil
}

func (p *MockProvider) CreateDir(_ context.Context, id, rel string) error {
	full, e := (Sandbox{p.roots}).Resolve(id, rel, true)
	if e != nil {
		return e
	}
	if e := os.Mkdir(full, 0700); e != nil {
		return safeOS(rel, "create", e)
	}
	return nil
}

func (p *MockProvider) Copy(_ context.Context, id, src, dst string) error {
	a, e := (Sandbox{p.roots}).Resolve(id, src, false)
	if e != nil {
		return e
	}
	b, e := (Sandbox{p.roots}).Resolve(id, dst, true)
	if e != nil {
		return e
	}
	in, e := os.Open(a)
	if e != nil {
		return safeOS(src, "read", e)
	}
	defer in.Close()
	if e := os.MkdirAll(filepath.Dir(b), 0700); e != nil {
		return fmt.Errorf("create %q: io error", dst)
	}
	out, e := os.OpenFile(b, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return safeOS(dst, "write", e)
	}
	if _, e = io.Copy(out, in); e != nil {
		out.Close()
		_ = os.Remove(b)
		return safeOS(dst, "write", e)
	}
	return out.Sync()
}

func (p *MockProvider) Move(_ context.Context, id, src, dst string) error {
	a, e := (Sandbox{p.roots}).Resolve(id, src, false)
	if e != nil {
		return e
	}
	b, e := (Sandbox{p.roots}).Resolve(id, dst, true)
	if e != nil {
		return e
	}
	if e := os.MkdirAll(filepath.Dir(b), 0700); e != nil {
		return fmt.Errorf("create %q: io error", dst)
	}
	if e := os.Rename(a, b); e != nil {
		return safeOS(src, "move", e)
	}
	return nil
}

func (p *MockProvider) Delete(_ context.Context, id, rel string) error {
	full, e := (Sandbox{p.roots}).Resolve(id, rel, false)
	if e != nil {
		return e
	}
	if e := os.RemoveAll(full); e != nil {
		return safeOS(rel, "delete", e)
	}
	return nil
}

func (p *MockProvider) Trash(_ context.Context, id, rel string) (string, error) {
	full, e := (Sandbox{p.roots}).Resolve(id, rel, false)
	if e != nil {
		return "", e
	}
	if _, e = os.Stat(full); e != nil {
		return "", safeOS(rel, "stat", e)
	}
	root, e := p.root(id)
	if e != nil {
		return "", e
	}
	trashDir := filepath.Join(root, ".Trash")
	if e := os.MkdirAll(trashDir, 0700); e != nil {
		return "", fmt.Errorf("trash unavailable")
	}
	id2, e := randID()
	if e != nil {
		return "", fmt.Errorf("trash unavailable")
	}
	target := filepath.Join(trashDir, id2, rel)
	if e := os.MkdirAll(filepath.Dir(target), 0700); e != nil {
		return "", fmt.Errorf("trash unavailable")
	}
	if e := os.Rename(full, target); e != nil {
		return "", safeOS(rel, "delete", e)
	}
	return filepath.ToSlash(filepath.Join(id2, rel)), nil
}

func (p *MockProvider) Restore(_ context.Context, id, src, trashRel string) error {
	root, e := p.root(id)
	if e != nil {
		return e
	}
	srcFull, e := (Sandbox{p.roots}).Resolve(id, src, true)
	if e != nil {
		return e
	}
	if e := os.MkdirAll(filepath.Dir(srcFull), 0700); e != nil {
		return fmt.Errorf("restore unavailable")
	}
	trashFull := filepath.Join(root, ".Trash", trashRel)
	if e := os.Rename(trashFull, srcFull); e != nil {
		return safeOS(src, "restore", e)
	}
	return nil
}

func (p *MockProvider) AvailableSpace(_ context.Context, id string) (int64, int64, error) {
	for _, d := range p.devices {
		if d.ID == id {
			return d.Total, d.Used, nil
		}
	}
	return 0, 0, fmt.Errorf("unknown storage")
}

var _ Provider = (*MockProvider)(nil)

// AllDevices and Resolve/FroviderFor make the mock provider usable on its own,
// which keeps storage tests independent of the router. Production always goes
// through Router.
func (p *MockProvider) AllDevices() []Device { return append([]Device(nil), p.devices...) }

func (p *MockProvider) Resolve(storageID, rel string, allowMissing bool) (string, error) {
	return (Sandbox{p.roots}).Resolve(storageID, rel, allowMissing)
}

func (p *MockProvider) ProviderFor(string) (Provider, error) { return p, nil }

func (p *MockProvider) MockVolumeCount() int       { return len(p.devices) }
func (p *MockProvider) ConfiguredVolumeCount() int { return 0 }

var _ Backend = (*MockProvider)(nil)
