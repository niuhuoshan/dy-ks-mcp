package httpapi

import (
	"encoding/json"
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

	var req engine.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	result, err := h.svc.RunCommentTask(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) loginStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	platformName := r.URL.Query().Get("platform")
	accountID := r.URL.Query().Get("account_id")
	status, err := h.svc.CheckLoginStatus(r.Context(), platformName, accountID)
	if err != nil {
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
	if err := h.svc.StartLogin(r.Context(), platformName, accountID); err != nil {
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
