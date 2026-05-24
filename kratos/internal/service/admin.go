package service

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"login-page/internal/biz"
)

type aiConfigResponse struct {
	BaseURL      string `json:"baseUrl"`
	Model        string `json:"model"`
	HasAPIKey    bool   `json:"hasApiKey"`
	APIKeyMasked string `json:"apiKeyMasked,omitempty"`
}

func (s *LoginService) AdminOverview(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminJSON(w, r) {
		return
	}
	stats, err := s.uc.AdminStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{Code: 1, Msg: err.Error()})
		return
	}
	users, err := s.uc.ListUsers(r.Context(), 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{Code: 1, Msg: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{Code: 0, Msg: "ok", Data: map[string]interface{}{
		"stats": stats,
		"users": users,
	}})
}

func (s *LoginService) AdminAIConfig(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminJSON(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.uc.GetAIConfig(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{Code: 1, Msg: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, response{Code: 0, Msg: "ok", Data: maskAIConfig(cfg)})
	case http.MethodPost:
		var req biz.AIConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Code: 1, Msg: "参数错误"})
			return
		}
		if err := s.uc.SaveAIConfig(r.Context(), req); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{Code: 1, Msg: err.Error()})
			return
		}
		cfg, _ := s.uc.GetAIConfig(r.Context())
		writeJSON(w, http.StatusOK, response{Code: 0, Msg: "AI 配置已保存", Data: maskAIConfig(cfg)})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, response{Code: 1, Msg: "方法不支持"})
	}
}

func (s *LoginService) CheckAdmin(w http.ResponseWriter, r *http.Request) {
	username, ok := currentUsername(r)
	if !ok || !isAdminUser(username) {
		writeJSON(w, http.StatusUnauthorized, response{Code: 1, Msg: "无管理员权限"})
		return
	}
	writeJSON(w, http.StatusOK, response{Code: 0, Msg: "ok", Data: map[string]string{"username": username}})
}

func (s *LoginService) requireAdminJSON(w http.ResponseWriter, r *http.Request) bool {
	username, ok := currentUsername(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, response{Code: 1, Msg: "请先登录"})
		return false
	}
	if !isAdminUser(username) {
		writeJSON(w, http.StatusForbidden, response{Code: 1, Msg: "无管理员权限"})
		return false
	}
	return true
}

func currentUsername(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", false
	}
	return strings.TrimSpace(cookie.Value), true
}

func isAdminUser(username string) bool {
	username = strings.TrimSpace(username)
	if username == "" {
		return false
	}
	adminUsers := strings.TrimSpace(os.Getenv("ADMIN_USERS"))
	if adminUsers == "" {
		adminUsers = "admin"
	}
	for _, item := range strings.Split(adminUsers, ",") {
		if strings.EqualFold(strings.TrimSpace(item), username) {
			return true
		}
	}
	return false
}

func requireAdminPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, ok := currentUsername(r)
		if !ok || !isAdminUser(username) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func maskAIConfig(cfg *biz.AIConfig) aiConfigResponse {
	resp := aiConfigResponse{BaseURL: cfg.BaseURL, Model: cfg.Model}
	if strings.TrimSpace(cfg.APIKey) != "" {
		resp.HasAPIKey = true
		resp.APIKeyMasked = maskSecret(cfg.APIKey)
	}
	return resp
}

func maskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	runes := []rune(secret)
	if len(runes) <= 8 {
		return "********"
	}
	return string(runes[:4]) + "****" + string(runes[len(runes)-4:])
}
