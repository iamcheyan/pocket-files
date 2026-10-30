package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotImplemented is returned by stub providers for SMB/WebDAV/Cloud.
var ErrNotImplemented = errors.New("not_implemented")

// LocalProvider serves one configured absolute root directory. The root is
// supplied by configuration, never by API callers, so callers cannot point it
// at production data or escape the sandbox.
type LocalProvider struct {
	id   string
	name string
	kind string
	root string
	ro   bool
}

func NewLocalProvider(id, name, kind, root string, readOnly bool) *LocalProvider {
	return &LocalProvider{id: id, name: name, kind: kind, root: root, ro: readOnly}
}

func (p *LocalProvider) ID() string     { return p.id }
func (p *LocalProvider) Name() string   { return p.name }
func (p *LocalProvider) Kind() string   { return p.kind }
func (p *LocalProvider) Root() string   { return p.root }
func (p *LocalProvider) ReadOnly() bool { return p.ro }

func (p *LocalProvider) resolve(rel string, allowMissing bool) (string, error) {
	return resolveLocalPath(p.root, rel, allowMissing)
}

func (p *LocalProvider) Discover(context.Context) ([]Device, error) {
	return []Device{{
		ID: p.id, Name: p.name, Kind: p.kind, Status: "ready",
		Mock: false, ReadOnly: p.ro, Root: p.root,
	}}, nil
}

func (p *LocalProvider) Mount(context.Context, string) error   { return ErrNotImplemented }
func (p *LocalProvider) Unmount(context.Context, string) error { return ErrNotImplemented }

func (p *LocalProvider) entries(rel string) ([]Entry, error) {
	full, err := p.resolve(rel, false)
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(items))
	for _, it := range items {
		info, err := it.Info()
		if err != nil {
			return nil, err
		}
		kind := "file"
		if it.IsDir() {
			kind = "directory"
		}
		out = append(out, Entry{
			ID:        filepath.ToSlash(filepath.Join(rel, it.Name())),
			StorageID: p.id, Name: it.Name(), Parent: rel, Kind: kind,
			Size: info.Size(), Modified: info.ModTime(), Created: info.ModTime(),
			Extension: strings.TrimPrefix(filepath.Ext(it.Name()), "."),
		})
	}
	return out, nil
}

func (p *LocalProvider) Browse(_ context.Context, _, rel string) ([]Entry, error) {
	return p.entries(rel)
}

func (p *LocalProvider) Search(_ context.Context, _, q string) ([]Entry, error) {
	all, err := p.entries("")
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, x := range all {
		if strings.Contains(strings.ToLower(x.Name), strings.ToLower(q)) {
			out = append(out, x)
		} else if x.Kind == "directory" {
			sub, e := p.entries(x.ID)
			if e == nil {
				for _, y := range sub {
					if strings.Contains(strings.ToLower(y.Name), strings.ToLower(q)) {
						out = append(out, y)
					}
				}
			}
		}
	}
	return out, nil
}

func (p *LocalProvider) Stat(_ context.Context, _, rel string) (Entry, error) {
	full, err := p.resolve(rel, false)
	if err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return Entry{}, err
	}
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	return Entry{
		ID: rel, StorageID: p.id, Name: filepath.Base(rel), Kind: kind,
		Size: info.Size(), Modified: info.ModTime(), Created: info.ModTime(),
		Extension: strings.TrimPrefix(filepath.Ext(rel), "."),
	}, nil
}

func (p *LocalProvider) CreateDir(_ context.Context, _, rel string) error {
	full, err := p.resolve(rel, true)
	if err != nil {
		return err
	}
	return os.Mkdir(full, 0700)
}

func (p *LocalProvider) Copy(_ context.Context, _, src, dst string) error {
	a, err := p.resolve(src, false)
	if err != nil {
		return err
	}
	b, err := p.resolve(dst, true)
	if err != nil {
		return err
	}
	in, err := os.Open(a)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(b, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(b)
		return err
	}
	return out.Sync()
}

func (p *LocalProvider) Move(_ context.Context, _, src, dst string) error {
	a, err := p.resolve(src, false)
	if err != nil {
		return err
	}
	b, err := p.resolve(dst, true)
	if err != nil {
		return err
	}
	return os.Rename(a, b)
}

func (p *LocalProvider) Delete(_ context.Context, _, rel string) error {
	full, err := p.resolve(rel, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(full)
}

func (p *LocalProvider) Trash(_ context.Context, _, rel string) (string, error) {
	full, err := p.resolve(rel, false)
	if err != nil {
		return "", err
	}
	trashRoot := filepath.Join(p.root, ".Trash")
	if err = os.MkdirAll(trashRoot, 0700); err != nil {
		return "", err
	}
	id, err := randID()
	if err != nil {
		return "", err
	}
	rel = strings.TrimPrefix(rel, "/")
	target := filepath.Join(trashRoot, id, rel)
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return "", err
	}
	if err = os.Rename(full, target); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(id, rel)), nil
}

func (p *LocalProvider) Restore(_ context.Context, _, src, trashRel string) error {
	full, err := p.resolve(src, true)
	if err != nil {
		return err
	}
	trashRoot := filepath.Join(p.root, ".Trash")
	orig := filepath.Join(trashRoot, trashRel)
	if err = os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		return err
	}
	if err = os.Rename(orig, full); err != nil {
		return err
	}
	return nil
}

func (p *LocalProvider) AvailableSpace(_ context.Context, _ string) (int64, int64, error) {
	total, _, free, err := statfsNumbers(p.root)
	if err != nil {
		return 0, 0, err
	}
	return total, total - free, nil
}
