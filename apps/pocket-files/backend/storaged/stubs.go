package storaged

import (
	"context"
)

// UnimplementedProvider is a stub for network/cloud providers that are not yet
// wired up. It exists so the registry can advertise them and the API can return
// 501 instead of 500 when someone tries to use them.
type UnimplementedProvider struct {
	id      string
	name    string
	kind    string
	capable string
}

func NewUnimplementedProvider(id, name, kind, capable string) *UnimplementedProvider {
	return &UnimplementedProvider{id: id, name: name, kind: kind, capable: capable}
}

func (p *UnimplementedProvider) ID() string   { return p.id }
func (p *UnimplementedProvider) Name() string { return p.name }
func (p *UnimplementedProvider) Kind() string { return p.kind }

func (p *UnimplementedProvider) Discover(context.Context) ([]Device, error) {
	return []Device{{
		ID: p.id, Name: p.name, Kind: p.kind, Status: "not_implemented",
		Mock: true, ReadOnly: true,
	}}, nil
}

func (p *UnimplementedProvider) Mount(context.Context, string) error   { return ErrNotImplemented }
func (p *UnimplementedProvider) Unmount(context.Context, string) error { return ErrNotImplemented }
func (p *UnimplementedProvider) Browse(context.Context, string, string) ([]Entry, error) {
	return nil, ErrNotImplemented
}
func (p *UnimplementedProvider) Search(context.Context, string, string) ([]Entry, error) {
	return nil, ErrNotImplemented
}
func (p *UnimplementedProvider) Stat(context.Context, string, string) (Entry, error) {
	return Entry{}, ErrNotImplemented
}
func (p *UnimplementedProvider) Copy(context.Context, string, string, string) error {
	return ErrNotImplemented
}
func (p *UnimplementedProvider) Move(context.Context, string, string, string) error {
	return ErrNotImplemented
}
func (p *UnimplementedProvider) Delete(context.Context, string, string) error {
	return ErrNotImplemented
}
func (p *UnimplementedProvider) CreateDir(context.Context, string, string) error {
	return ErrNotImplemented
}
func (p *UnimplementedProvider) AvailableSpace(context.Context, string) (int64, int64, error) {
	return 0, 0, ErrNotImplemented
}
func (p *UnimplementedProvider) Trash(context.Context, string, string) (string, error) {
	return "", ErrNotImplemented
}
func (p *UnimplementedProvider) Restore(context.Context, string, string, string) error {
	return ErrNotImplemented
}
