package registry

import (
	"fmt"
	"path/filepath"
	"sort"

	base "dy-ks-mcp/internal/platform"
	"dy-ks-mcp/internal/platform/douyin"
	"dy-ks-mcp/internal/platform/kuaishou"
)

type Registry struct {
	clients map[string]base.Client
}

func New(selectorDir string) (*Registry, error) {
	dyClient, err := douyin.NewClient(filepath.Join(selectorDir, "douyin.yaml"))
	if err != nil {
		return nil, err
	}
	ksClient, err := kuaishou.NewClient(filepath.Join(selectorDir, "kuaishou.yaml"))
	if err != nil {
		return nil, err
	}
	return &Registry{
		clients: map[string]base.Client{
			"douyin":   dyClient,
			"kuaishou": ksClient,
		},
	}, nil
}

func (r *Registry) Get(name string) (base.Client, error) {
	c, ok := r.clients[name]
	if !ok {
		return nil, fmt.Errorf("unsupported platform %q", name)
	}
	return c, nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.clients))
	for name := range r.clients {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
