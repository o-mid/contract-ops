package connectors

import (
	"context"
	"fmt"
	"sort"
)

type Registry struct {
	byKind map[string]Connector
}

func NewRegistry(list ...Connector) *Registry {
	byKind := make(map[string]Connector, len(list))
	for _, connector := range list {
		byKind[connector.Kind()] = connector
	}
	return &Registry{byKind: byKind}
}

func (r *Registry) Get(kind string) (Connector, bool) {
	connector, ok := r.byKind[kind]
	return connector, ok
}

func (r *Registry) Kinds() []string {
	kinds := make([]string, 0, len(r.byKind))
	for kind := range r.byKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// Disabled is a connector that is registered so the API can name it,
// and that refuses to call the vendor.
type Disabled struct {
	KindName    string
	DisplayName string
}

func (d Disabled) Kind() string { return d.KindName }

func (d Disabled) Describe() Descriptor {
	return Descriptor{Kind: d.KindName, DisplayName: d.DisplayName}
}

func (d Disabled) Verify(context.Context, Credential) error { return d.disabled() }

func (d Disabled) Fetch(context.Context, Credential, Window, Cursor) (Page, error) {
	return Page{}, d.disabled()
}

func (d Disabled) Normalize(RawRecord) ([]CostRow, error) { return nil, d.disabled() }

func (d Disabled) disabled() error {
	return fmt.Errorf("%s connector is not enabled", d.KindName)
}
