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
	SelectorsDir string        `yaml:"selectors_dir"`
	Browser      BrowserConfig `yaml:"browser"`
}

type BrowserConfig struct {
	NodeBinary        string `yaml:"node_binary"`
	ScriptPath        string `yaml:"script_path"`
	WSURL             string `yaml:"ws_url"`
	Headless          bool   `yaml:"headless"`
	ExecutablePath    string `yaml:"executable_path"`
	UserDataRoot      string `yaml:"user_data_root"`
	ActionTimeout     string `yaml:"action_timeout"`
	NavigationTimeout string `yaml:"navigation_timeout"`
	LoginTimeout      string `yaml:"login_timeout"`
	PostLoadWaitMS    int    `yaml:"post_load_wait_ms"`
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
	if c.Platform.Browser.NodeBinary == "" {
		c.Platform.Browser.NodeBinary = "node"
	}
	if c.Platform.Browser.ScriptPath == "" {
		c.Platform.Browser.ScriptPath = "./tools/platform-browser.mjs"
	}
	if c.Platform.Browser.UserDataRoot == "" {
		c.Platform.Browser.UserDataRoot = "./data/browser"
	}
	if c.Platform.Browser.ActionTimeout == "" {
		c.Platform.Browser.ActionTimeout = "20s"
	}
	if c.Platform.Browser.NavigationTimeout == "" {
		c.Platform.Browser.NavigationTimeout = "45s"
	}
	if c.Platform.Browser.LoginTimeout == "" {
		c.Platform.Browser.LoginTimeout = "10m"
	}
	if c.Platform.Browser.PostLoadWaitMS <= 0 {
		c.Platform.Browser.PostLoadWaitMS = 1500
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
	if _, err := c.Platform.Browser.ActionTimeoutDuration(); err != nil {
		return fmt.Errorf("invalid platform.browser.action_timeout: %w", err)
	}
	if _, err := c.Platform.Browser.NavigationTimeoutDuration(); err != nil {
		return fmt.Errorf("invalid platform.browser.navigation_timeout: %w", err)
	}
	if _, err := c.Platform.Browser.LoginTimeoutDuration(); err != nil {
		return fmt.Errorf("invalid platform.browser.login_timeout: %w", err)
	}
	if c.Platform.Browser.PostLoadWaitMS < 0 {
		return fmt.Errorf("platform.browser.post_load_wait_ms must be >= 0")
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

func (c BrowserConfig) ActionTimeoutDuration() (time.Duration, error) {
	return time.ParseDuration(c.ActionTimeout)
}

func (c BrowserConfig) NavigationTimeoutDuration() (time.Duration, error) {
	return time.ParseDuration(c.NavigationTimeout)
}

func (c BrowserConfig) LoginTimeoutDuration() (time.Duration, error) {
	return time.ParseDuration(c.LoginTimeout)
}

func (c BrowserConfig) PostLoadWaitDuration() time.Duration {
	return time.Duration(c.PostLoadWaitMS) * time.Millisecond
}

func parseHHMM(v string) (int, error) {
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}
