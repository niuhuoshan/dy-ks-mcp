package config

import "testing"

func TestApplyDefaultsSetsExpectedValues(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Store.SQLitePath != "./data/dy-ks-mcp.db" {
		t.Fatalf("Store.SQLitePath = %q, want ./data/dy-ks-mcp.db", cfg.Store.SQLitePath)
	}
	if len(cfg.Engine.CommentTemplates) != 1 || cfg.Engine.CommentTemplates[0] != "Thanks for sharing." {
		t.Fatalf("Engine.CommentTemplates = %#v, want default template", cfg.Engine.CommentTemplates)
	}
}

func TestValidateRejectsInvalidRandomPauseRange(t *testing.T) {
	cfg := &Config{
		Engine: EngineConfig{
			RateLimit: RateLimitConfig{
				Actions: 1,
				Per:     "1m",
			},
			RandomPause: RandomPauseConfig{
				MinMS: 200,
				MaxMS: 100,
			},
			TimeWindow: TimeWindowConfig{
				Timezone: "UTC",
				Start:    "00:00",
				End:      "23:59",
			},
			CircuitBreaker: CircuitBreakerConfig{
				FailureThreshold: 1,
				OpenFor:          "1m",
			},
			CommentTemplates: []string{"ok"},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate error = nil, want non-nil")
	}
}
