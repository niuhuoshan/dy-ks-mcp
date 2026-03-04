package registry

import (
	"fmt"
	"path/filepath"
	"sort"

	"dy-ks-mcp/internal/config"
	base "dy-ks-mcp/internal/platform"
	"dy-ks-mcp/internal/platform/douyin"
	"dy-ks-mcp/internal/platform/kuaishou"
)

type Registry struct {
	clients map[string]base.Client
}

func New(cfg config.PlatformConfig) (*Registry, error) {
	dyClient, err := douyin.NewClient(filepath.Join(cfg.SelectorsDir, "douyin.yaml"), cfg.Browser)
	if err != nil {
		return nil, err
	}
	ksClient, err := kuaishou.NewClient(filepath.Join(cfg.SelectorsDir, "kuaishou.yaml"), cfg.Browser)
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
