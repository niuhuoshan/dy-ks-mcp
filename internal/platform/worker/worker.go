package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	base "dy-ks-mcp/internal/platform"
)

type BrowserSettings struct {
	WSURL             string `json:"ws_url"`
	Headless          bool   `json:"headless"`
	ExecutablePath    string `json:"executable_path"`
	UserDataRoot      string `json:"user_data_root"`
	ActionTimeoutMS   int64  `json:"action_timeout_ms"`
	NavigationTimeout int64  `json:"navigation_timeout_ms"`
	LoginTimeoutMS    int64  `json:"login_timeout_ms"`
	PostLoadWaitMS    int64  `json:"post_load_wait_ms"`
}

type Request struct {
	Action    string            `json:"action"`
	Platform  string            `json:"platform"`
	AccountID string            `json:"account_id"`
	Selectors map[string]string `json:"selectors"`
	Browser   BrowserSettings   `json:"browser"`

	Keyword   string `json:"keyword,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	TimeRange string `json:"time_range,omitempty"`
	Limit     int    `json:"limit,omitempty"`

	PostID  string `json:"post_id,omitempty"`
	Content string `json:"content,omitempty"`
}

type Response struct {
	LoggedIn bool        `json:"logged_in"`
	Message  string      `json:"message"`
	Posts    []base.Post `json:"posts"`
}

type rawResponse struct {
	OK       bool        `json:"ok"`
	Error    string      `json:"error"`
	LoggedIn bool        `json:"logged_in"`
	Message  string      `json:"message"`
	Posts    []base.Post `json:"posts"`
}

func Run(ctx context.Context, nodeBinary string, scriptPath string, req Request) (Response, error) {
	if strings.TrimSpace(nodeBinary) == "" {
		nodeBinary = "node"
	}
	if strings.TrimSpace(scriptPath) == "" {
		return Response{}, fmt.Errorf("browser worker script path is empty")
	}

	if ok, err := relayAttached(req.Browser.WSURL); err == nil && !ok {
		return Response{}, fmt.Errorf("relay not attached: no browser tabs connected")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("marshal worker request: %w", err)
	}

	workerCtx, cancel := context.WithTimeout(ctx, requestTimeout(req))
	defer cancel()

	cmd := exec.CommandContext(workerCtx, nodeBinary, scriptPath, string(payload))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(workerCtx.Err(), context.DeadlineExceeded) {
			return Response{}, fmt.Errorf("browser worker timed out after %s", requestTimeout(req))
		}
		return Response{}, fmt.Errorf("browser worker failed: %s", joinMessages(err, stdout.String(), stderr.String()))
	}

	line, err := lastJSONLine(stdout.String())
	if err != nil {
		return Response{}, err
	}

	var raw rawResponse
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Response{}, fmt.Errorf("decode worker response: %w", err)
	}
	if !raw.OK {
		if raw.Error == "" {
			raw.Error = "unknown browser worker error"
		}
		return Response{}, errors.New(raw.Error)
	}

	return Response{
		LoggedIn: raw.LoggedIn,
		Message:  raw.Message,
		Posts:    raw.Posts,
	}, nil
}

func NewBrowserSettings(
	wsURL string,
	headless bool,
	executablePath string,
	userDataRoot string,
	actionTimeout time.Duration,
	navigationTimeout time.Duration,
	loginTimeout time.Duration,
	postLoadWait time.Duration,
) BrowserSettings {
	return BrowserSettings{
		WSURL:             wsURL,
		Headless:          headless,
		ExecutablePath:    executablePath,
		UserDataRoot:      userDataRoot,
		ActionTimeoutMS:   actionTimeout.Milliseconds(),
		NavigationTimeout: navigationTimeout.Milliseconds(),
		LoginTimeoutMS:    loginTimeout.Milliseconds(),
		PostLoadWaitMS:    postLoadWait.Milliseconds(),
	}
}

func lastJSONLine(s string) (string, error) {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			return line, nil
		}
	}
	if strings.TrimSpace(s) == "" {
		return "", errors.New("browser worker returned empty output")
	}
	return "", fmt.Errorf("browser worker returned non-json output: %s", strings.TrimSpace(s))
}

func joinMessages(err error, stdout string, stderr string) string {
	parts := []string{err.Error()}
	if v := strings.TrimSpace(stderr); v != "" {
		parts = append(parts, v)
	}
	if v := strings.TrimSpace(stdout); v != "" {
		parts = append(parts, v)
	}
	return strings.Join(parts, " | ")
}

func requestTimeout(req Request) time.Duration {
	actionTimeout := time.Duration(maxInt64(1000, req.Browser.ActionTimeoutMS)) * time.Millisecond
	navTimeout := time.Duration(maxInt64(1000, req.Browser.NavigationTimeout)) * time.Millisecond
	loginTimeout := time.Duration(maxInt64(1000, req.Browser.LoginTimeoutMS)) * time.Millisecond
	postLoadWait := time.Duration(maxInt64(0, req.Browser.PostLoadWaitMS)) * time.Millisecond

	switch strings.ToLower(strings.TrimSpace(req.Action)) {
	case "login":
		d := loginTimeout + 10*time.Second
		if d > 2*time.Minute {
			d = 2 * time.Minute
		}
		return d
	case "search", "comment":
		d := navTimeout + actionTimeout*3 + postLoadWait*3 + 15*time.Second
		if d < 20*time.Second {
			d = 20 * time.Second
		}
		if d > 90*time.Second {
			d = 90 * time.Second
		}
		return d
	case "check_login":
		d := navTimeout + actionTimeout + 8*time.Second
		if d < 15*time.Second {
			d = 15 * time.Second
		}
		if d > 45*time.Second {
			d = 45 * time.Second
		}
		return d
	default:
		return 45 * time.Second
	}
}

func relayAttached(wsURL string) (bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(wsURL))
	if err != nil {
		return true, nil
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return true, nil
	}
	if !strings.Contains(parsed.Path, "/cdp") {
		return true, nil
	}

	httpScheme := "http"
	if parsed.Scheme == "wss" {
		httpScheme = "https"
	}
	probeURL := fmt.Sprintf("%s://%s/json/list", httpScheme, parsed.Host)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(probeURL)
	if err != nil {
		return true, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return true, nil
	}

	var tabs []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return true, nil
	}
	if len(tabs) == 0 {
		return false, nil
	}
	return true, nil
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
