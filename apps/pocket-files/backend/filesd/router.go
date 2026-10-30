package main

import (
	"context"
	"errors"
	"fmt"
)

// storageBackend is the daemon-facing view of storage. Both the mock provider
// and the multi-provider router satisfy it, so backend code never depends on a
// concrete provider family.
type storageBackend interface {
	Discover(context.Context) ([]Device, error)
	Browse(context.Context, string, string) ([]Entry, error)
	Search(context.Context, string, string) ([]Entry, error)
	Stat(context.Context, string, string) (Entry, error)
	Copy(context.Context, string, string, string) error
	Move(context.Context, string, string, string) error
	Delete(context.Context, string, string) error
	CreateDir(context.Context, string, string) error
	AvailableSpace(context.Context, string) (int64, int64, error)
	Trash(context.Context, string, string) (string, error)
	Restore(context.Context, string, string, string) error
	allDevices() []Device
	resolve(storageID, rel string, allowMissing bool) (string, error)
}

// providerRouter routes every operation to the provider that owns the storage
// ID. Local providers are configured by the operator; mock volumes are always
// present; network/cloud providers are visible stubs.
type providerRouter struct {
	mock  *MockProvider
	byID  map[string]Provider
	order []string
	roots map[string]string
}

func newProviderRouter(mock *MockProvider, cfg ProviderConfig) (*providerRouter, error) {
	r := &providerRouter{mock: mock, byID: map[string]Provider{}, roots: map[string]string{}}
	for _, d := range mock.devices {
		r.byID[d.ID] = mock
		r.roots[d.ID] = mock.roots[d.ID]
		r.order = append(r.order, d.ID)
	}
	for _, entry := range cfg.Storages {
		var provider Provider
		switch entry.Kind {
		case string(KindInternal):
			provider = NewInternalStorageProvider(entry.ID, friendlyOrDefault(entry), entry.Root, entry.ReadOnly)
		case string(KindTF):
			provider = NewTFCardProvider(entry.ID, friendlyOrDefault(entry), entry.Root, entry.ReadOnly)
		case string(KindUSB):
			provider = NewUSBStorageProvider(entry.ID, friendlyOrDefault(entry), entry.Root, entry.ReadOnly)
		default:
			return nil, fmt.Errorf("unsupported provider kind %q", entry.Kind)
		}
		if _, exists := r.byID[entry.ID]; exists {
			return nil, fmt.Errorf("duplicate storage id %q", entry.ID)
		}
		r.byID[entry.ID] = provider
		r.roots[entry.ID] = entry.Root
		r.order = append(r.order, entry.ID)
	}
	for _, stub := range []*UnimplementedProvider{
		NewUnimplementedProvider("smb", "SMB", string(KindSMB), "Browse, Search, Stat, transfer"),
		NewUnimplementedProvider("webdav", "WebDAV", string(KindWebDAV), "Browse, Search, Stat, transfer"),
		NewUnimplementedProvider("cloud", "Cloud", string(KindCloud), "Browse, Search, Stat, transfer"),
	} {
		r.byID[stub.ID()] = stub
		r.order = append(r.order, stub.ID())
	}
	return r, nil
}

func friendlyOrDefault(entry StorageEntry) string {
	if entry.FriendlyName != "" {
		return entry.FriendlyName
	}
	return entry.ID
}

func (r *providerRouter) provider(id string) (Provider, error) {
	p, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("unknown storage")
	}
	return p, nil
}

func (r *providerRouter) Discover(ctx context.Context) ([]Device, error) {
	out := make([]Device, 0, len(r.order))
	for _, id := range r.order {
		devices, err := r.byID[id].Discover(ctx)
		if err != nil {
			continue
		}
		out = append(out, devices...)
	}
	return out, nil
}

func (r *providerRouter) Browse(ctx context.Context, id, rel string) ([]Entry, error) {
	p, err := r.provider(id)
	if err != nil {
		return nil, err
	}
	return p.Browse(ctx, id, rel)
}

func (r *providerRouter) Search(ctx context.Context, id, q string) ([]Entry, error) {
	p, err := r.provider(id)
	if err != nil {
		return nil, err
	}
	return p.Search(ctx, id, q)
}

func (r *providerRouter) Stat(ctx context.Context, id, rel string) (Entry, error) {
	p, err := r.provider(id)
	if err != nil {
		return Entry{}, err
	}
	return p.Stat(ctx, id, rel)
}

func (r *providerRouter) Copy(ctx context.Context, id, src, dst string) error {
	p, err := r.provider(id)
	if err != nil {
		return err
	}
	return p.Copy(ctx, id, src, dst)
}

func (r *providerRouter) Move(ctx context.Context, id, src, dst string) error {
	p, err := r.provider(id)
	if err != nil {
		return err
	}
	return p.Move(ctx, id, src, dst)
}

func (r *providerRouter) Delete(ctx context.Context, id, rel string) error {
	p, err := r.provider(id)
	if err != nil {
		return err
	}
	return p.Delete(ctx, id, rel)
}

func (r *providerRouter) CreateDir(ctx context.Context, id, rel string) error {
	p, err := r.provider(id)
	if err != nil {
		return err
	}
	return p.CreateDir(ctx, id, rel)
}

func (r *providerRouter) AvailableSpace(ctx context.Context, id string) (int64, int64, error) {
	p, err := r.provider(id)
	if err != nil {
		return 0, 0, err
	}
	return p.AvailableSpace(ctx, id)
}

func (r *providerRouter) Trash(ctx context.Context, id, rel string) (string, error) {
	switch p := r.byID[id].(type) {
	case *MockProvider:
		return p.Trash(ctx, id, rel)
	case *InternalStorageProvider:
		return p.Trash(ctx, id, rel)
	case *TFCardProvider:
		return p.Trash(ctx, id, rel)
	case *USBStorageProvider:
		return p.Trash(ctx, id, rel)
	}
	return "", ErrNotImplemented
}

func (r *providerRouter) Restore(ctx context.Context, id, src, trashRel string) error {
	switch p := r.byID[id].(type) {
	case *MockProvider:
		return p.Restore(ctx, id, src, trashRel)
	case *InternalStorageProvider:
		return p.Restore(ctx, id, src, trashRel)
	case *TFCardProvider:
		return p.Restore(ctx, id, src, trashRel)
	case *USBStorageProvider:
		return p.Restore(ctx, id, src, trashRel)
	}
	return ErrNotImplemented
}

func (r *providerRouter) allDevices() []Device {
	devices, err := r.Discover(context.Background())
	if err != nil {
		return nil
	}
	return devices
}

// resolve returns a host path only for local providers. Network stubs refuse so
// no caller can read a filesystem on their behalf.
func (r *providerRouter) resolve(storageID, rel string, allowMissing bool) (string, error) {
	root, ok := r.roots[storageID]
	if !ok {
		if _, known := r.byID[storageID]; known {
			return "", ErrNotImplemented
		}
		return "", fmt.Errorf("unknown storage")
	}
	return resolveLocalPath(root, rel, allowMissing)
}

func isNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented)
}
