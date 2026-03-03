package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Store    StoreConfig    `yaml:"store"`
	Platform PlatformConfig `yaml:"platform"`
	Engine   EngineConfig   `yaml:"engine"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type StoreConfig struct {
	SQLitePath string `yaml:"sqlite_path"`
}

type PlatformConfig struct {
	SelectorsDir string `yaml:"selectors_dir"`
}

type EngineConfig struct {
	RateLimit        RateLimitConfig      `yaml:"rate_limit"`
	RandomPause      RandomPauseConfig    `yaml:"random_pause"`
	TimeWindow       TimeWindowConfig     `yaml:"time_window"`
	CircuitBreaker   CircuitBreakerConfig `yaml:"circuit_breaker"`
	Filter           FilterConfig         `yaml:"filter"`
	CommentTemplates []string             `yaml:"comment_templates"`
}

type RateLimitConfig struct {
	Actions int    `yaml:"actions"`
	Per     string `yaml:"per"`
}

type RandomPauseConfig struct {
	MinMS int `yaml:"min_ms"`
	MaxMS int `yaml:"max_ms"`
}

type TimeWindowConfig struct {
	Timezone string `yaml:"timezone"`
	Start    string `yaml:"start"`
	End      string `yaml:"end"`
}

type CircuitBreakerConfig struct {
	FailureThreshold int    `yaml:"failure_threshold"`
	OpenFor          string `yaml:"open_for"`
}

type FilterConfig struct {
	BlockedKeywords []string `yaml:"blocked_keywords"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Store.SQLitePath == "" {
		c.Store.SQLitePath = "./data/dy-ks-mcp.db"
	}
	if c.Platform.SelectorsDir == "" {
		c.Platform.SelectorsDir = "./selectors"
	}
	if c.Engine.RateLimit.Actions <= 0 {
		c.Engine.RateLimit.Actions = 10
	}
	if c.Engine.RateLimit.Per == "" {
		c.Engine.RateLimit.Per = "1m"
	}
	if c.Engine.RandomPause.MinMS < 0 {
		c.Engine.RandomPause.MinMS = 0
	}
	if c.Engine.RandomPause.MaxMS <= 0 {
		c.Engine.RandomPause.MaxMS = 1000
	}
	if c.Engine.TimeWindow.Timezone == "" {
		c.Engine.TimeWindow.Timezone = "Asia/Shanghai"
	}
	if c.Engine.TimeWindow.Start == "" {
		c.Engine.TimeWindow.Start = "00:00"
	}
	if c.Engine.TimeWindow.End == "" {
		c.Engine.TimeWindow.End = "23:59"
	}
	if c.Engine.CircuitBreaker.FailureThreshold <= 0 {
		c.Engine.CircuitBreaker.FailureThreshold = 3
	}
	if c.Engine.CircuitBreaker.OpenFor == "" {
		c.Engine.CircuitBreaker.OpenFor = "1m"
	}
	if len(c.Engine.CommentTemplates) == 0 {
		c.Engine.CommentTemplates = []string{"Thanks for sharing."}
	}
}

func (c *Config) Validate() error {
	if _, err := c.Engine.RateLimit.PerDuration(); err != nil {
		return fmt.Errorf("invalid rate_limit.per: %w", err)
	}
	if _, err := c.Engine.CircuitBreaker.OpenDuration(); err != nil {
		return fmt.Errorf("invalid circuit_breaker.open_for: %w", err)
	}
	if _, _, err := c.Engine.TimeWindow.Parsed(); err != nil {
		return fmt.Errorf("invalid time_window: %w", err)
	}
	if c.Engine.RandomPause.MinMS > c.Engine.RandomPause.MaxMS {
		return fmt.Errorf("random_pause.min_ms must be <= random_pause.max_ms")
	}
	if c.Engine.RateLimit.Actions <= 0 {
		return fmt.Errorf("rate_limit.actions must be > 0")
	}
	if c.Engine.CircuitBreaker.FailureThreshold <= 0 {
		return fmt.Errorf("circuit_breaker.failure_threshold must be > 0")
	}
	return nil
}

func (c RateLimitConfig) PerDuration() (time.Duration, error) {
	return time.ParseDuration(c.Per)
}

func (c RandomPauseConfig) MinDuration() time.Duration {
	return time.Duration(c.MinMS) * time.Millisecond
}

func (c RandomPauseConfig) MaxDuration() time.Duration {
	return time.Duration(c.MaxMS) * time.Millisecond
}

func (c TimeWindowConfig) Parsed() (startMin int, endMin int, err error) {
	startMin, err = parseHHMM(c.Start)
	if err != nil {
		return 0, 0, err
	}
	endMin, err = parseHHMM(c.End)
	if err != nil {
		return 0, 0, err
	}
	return startMin, endMin, nil
}

func (c TimeWindowConfig) Location() (*time.Location, error) {
	return time.LoadLocation(c.Timezone)
}

func (c CircuitBreakerConfig) OpenDuration() (time.Duration, error) {
	return time.ParseDuration(c.OpenFor)
}

func parseHHMM(v string) (int, error) {
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}
