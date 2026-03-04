package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"dy-ks-mcp/internal/engine"
	"dy-ks-mcp/internal/service"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux, mcpHandler http.Handler) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/v1/run", h.runTask)
	mux.HandleFunc("/api/v1/search", h.searchPosts)
	mux.HandleFunc("/api/v1/comment/prepare", h.prepareCommentTarget)
	mux.HandleFunc("/api/v1/comment/submit", h.submitComment)
	mux.HandleFunc("/api/v1/comment/verify", h.verifyComment)
	mux.HandleFunc("/api/v1/login/status", h.loginStatus)
	mux.HandleFunc("/api/v1/login/start", h.loginStart)
	mux.Handle("/mcp", mcpHandler)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) runTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req struct {
		engine.RunRequest
		AutoSubmit  bool `json:"auto_submit"`
		TargetIndex int  `json:"target_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	ctx, cancel := withRequestTimeout(r.Context(), 95*time.Second)
	defer cancel()

	result := h.svc.RunCommentTaskWithStatus(ctx, req.RunRequest, req.AutoSubmit, req.TargetIndex)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) searchPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req service.SearchPostsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	ctx, cancel := withRequestTimeout(r.Context(), 75*time.Second)
	defer cancel()

	result := h.svc.SearchPosts(ctx, req)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) prepareCommentTarget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req service.PrepareCommentTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	ctx, cancel := withRequestTimeout(r.Context(), 75*time.Second)
	defer cancel()

	result := h.svc.PrepareCommentTarget(ctx, req)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) submitComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req service.SubmitCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	ctx, cancel := withRequestTimeout(r.Context(), 90*time.Second)
	defer cancel()

	result := h.svc.SubmitComment(ctx, req)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) verifyComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if r.Method == http.MethodGet {
		platformName := r.URL.Query().Get("platform")
		accountID := r.URL.Query().Get("account_id")
		postID := r.URL.Query().Get("post_id")
		ctx, cancel := withRequestTimeout(r.Context(), 20*time.Second)
		defer cancel()
		result := h.svc.VerifyComment(ctx, platformName, accountID, postID)
		writeJSON(w, http.StatusOK, result)
		return
	}

	defer r.Body.Close()
	var req struct {
		Platform  string `json:"platform"`
		AccountID string `json:"account_id"`
		PostID    string `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	ctx, cancel := withRequestTimeout(r.Context(), 20*time.Second)
	defer cancel()

	result := h.svc.VerifyComment(ctx, req.Platform, req.AccountID, req.PostID)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) loginStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	platformName := r.URL.Query().Get("platform")
	accountID := r.URL.Query().Get("account_id")
	ctx, cancel := withRequestTimeout(r.Context(), 45*time.Second)
	defer cancel()

	status, err := h.svc.CheckLoginStatus(ctx, platformName, accountID)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			writeError(w, http.StatusGatewayTimeout, "request timeout: check_login_status")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) loginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	platformName := r.URL.Query().Get("platform")
	accountID := r.URL.Query().Get("account_id")
	ctx, cancel := withRequestTimeout(r.Context(), 45*time.Second)
	defer cancel()

	if err := h.svc.StartLogin(ctx, platformName, accountID); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			writeError(w, http.StatusGatewayTimeout, "request timeout: start_login")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"platform":   platformName,
		"account_id": defaultAccount(accountID),
		"started":    true,
	})
}

func defaultAccount(accountID string) string {
	if accountID == "" {
		return "default"
	}
	return accountID
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func withRequestTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining <= d {
			return context.WithCancel(parent)
		}
	}
	return context.WithTimeout(parent, d)
}
