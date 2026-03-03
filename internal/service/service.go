package service

import (
	"context"
	"fmt"
	"strings"

	"dy-ks-mcp/internal/engine"
	"dy-ks-mcp/internal/platform"
	"dy-ks-mcp/internal/platform/registry"
)

type Service struct {
	registry *registry.Registry
	runner   *engine.Runner
}

func New(reg *registry.Registry, runner *engine.Runner) *Service {
	return &Service{registry: reg, runner: runner}
}

func (s *Service) RunCommentTask(ctx context.Context, req engine.RunRequest) (engine.RunResult, error) {
	client, err := s.client(req.Platform)
	if err != nil {
		return engine.RunResult{}, err
	}
	if req.AccountID == "" {
		req.AccountID = "default"
	}
	return s.runner.Run(ctx, client, req)
}

func (s *Service) CheckLoginStatus(ctx context.Context, platformName string, accountID string) (platform.LoginStatus, error) {
	client, err := s.client(platformName)
	if err != nil {
		return platform.LoginStatus{}, err
	}
	if accountID == "" {
		accountID = "default"
	}
	return client.CheckLogin(ctx, accountID)
}

func (s *Service) StartLogin(ctx context.Context, platformName string, accountID string) error {
	client, err := s.client(platformName)
	if err != nil {
		return err
	}
	if accountID == "" {
		accountID = "default"
	}
	return client.Login(ctx, accountID)
}

func (s *Service) SupportedPlatforms() []string {
	return s.registry.Names()
}

func (s *Service) client(platformName string) (platform.Client, error) {
	name := strings.ToLower(strings.TrimSpace(platformName))
	if name == "" {
		return nil, fmt.Errorf("platform is required")
	}
	return s.registry.Get(name)
}
