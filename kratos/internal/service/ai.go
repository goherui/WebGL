package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const aiSystemPrompt = `你是 heyongrui666.top 的站内 AI 助手，名字叫 Future AI。
你的职责：帮助用户了解网站登录、注册、欢迎页、未来实验室等功能；也可以帮用户生成个人简介、文案、标题、摘要和简单创意。
回答要求：使用中文，简洁友好，不编造不存在的网站功能；涉及账号密码时提醒用户不要泄露隐私。`

type aiLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time
}

var chatLimiter = &aiLimiter{windows: make(map[string][]time.Time)}

func (l *aiLimiter) allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)

	l.mu.Lock()
	defer l.mu.Unlock()

	items := l.windows[key]
	kept := items[:0]
	for _, t := range items {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		l.windows[key] = kept
		return false
	}
	kept = append(kept, now)
	l.windows[key] = kept
	return true
}

type aiChatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId"`
}

type aiChatData struct {
	Reply          string `json:"reply"`
	ConversationID string `json:"conversationId,omitempty"`
	Model          string `json:"model,omitempty"`
	Mock           bool   `json:"mock,omitempty"`
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (s *LoginService) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, response{Code: 1, Msg: "只支持 POST"})
		return
	}

	var req aiChatRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 16*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Code: 1, Msg: "参数错误"})
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		writeJSON(w, http.StatusBadRequest, response{Code: 1, Msg: "请输入问题"})
		return
	}
	if len([]rune(message)) > 1000 {
		writeJSON(w, http.StatusBadRequest, response{Code: 1, Msg: "问题太长了，请控制在 1000 字以内"})
		return
	}

	username := "guest"
	if cookie, err := r.Cookie("session_id"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		username = strings.TrimSpace(cookie.Value)
	}

	limit := 5
	if username != "guest" {
		limit = 20
	}
	if !chatLimiter.allow(username+":"+clientIP(r), limit, 24*time.Hour) {
		writeJSON(w, http.StatusTooManyRequests, response{Code: 1, Msg: "今日 AI 体验次数已用完，请明天再来"})
		return
	}

	reply, model, mock, err := s.generateAIReply(r.Context(), username, message)
	if err != nil {
		s.log.Errorf("ai chat failed: %v", err)
		writeJSON(w, http.StatusBadGateway, response{Code: 1, Msg: "AI 服务暂时不可用，请稍后再试"})
		return
	}

	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		conversationID = fmt.Sprintf("conv_%d", time.Now().UnixNano())
	}
	writeJSON(w, http.StatusOK, response{Code: 0, Msg: "ok", Data: aiChatData{Reply: reply, ConversationID: conversationID, Model: model, Mock: mock}})
}

func (s *LoginService) generateAIReply(ctx context.Context, username, message string) (reply string, model string, mock bool, err error) {
	apiKey := strings.TrimSpace(os.Getenv("AI_API_KEY"))
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("AI_API_BASE")), "/")
	model = strings.TrimSpace(os.Getenv("AI_MODEL"))
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	if model == "" {
		model = "deepseek-chat"
	}

	if apiKey == "" {
		return localAIFallback(username, message), model, true, nil
	}

	payload := openAIChatRequest{
		Model: model,
		Messages: []openAIMessage{
			{Role: "system", Content: aiSystemPrompt},
			{Role: "user", Content: fmt.Sprintf("当前用户：%s\n用户问题：%s", username, message)},
		},
		Temperature: 0.7,
		MaxTokens:   800,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", model, false, err
	}

	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", model, false, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", model, false, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", model, false, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", model, false, fmt.Errorf("model api status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", model, false, err
	}
	if parsed.Error != nil {
		return "", model, false, errors.New(parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", model, false, errors.New("empty ai response")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), model, false, nil
}

func localAIFallback(username, message string) string {
	lower := strings.ToLower(message)
	if strings.Contains(message, "注册") || strings.Contains(message, "账号") || strings.Contains(lower, "login") {
		return "我是 Future AI。当前还没有配置真实大模型 API Key，所以先用本地模式回答：你可以在首页注册账号，用户名至少 3 位、密码至少 6 位；登录后可以进入欢迎页和未来实验室。正式接入大模型后，我可以继续帮你处理账号引导、文案生成和站内问答。"
	}
	if strings.Contains(message, "简介") || strings.Contains(message, "文案") || strings.Contains(message, "签名") {
		return "可以，给你一个示例：\n\n“探索未来、记录灵感、把每一次登录都变成一次新的实验。欢迎来到 heyongrui666.top。”\n\n配置 AI_API_KEY 后，我可以按你的风格继续生成更多版本。"
	}
	return "我是 Future AI，已经接入到网站界面里了。当前服务器还没配置真实大模型 API Key，所以我先以本地体验模式运行。你可以问我网站使用方式、账号问题，也可以让我帮你生成简介、标题、文案。"
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
