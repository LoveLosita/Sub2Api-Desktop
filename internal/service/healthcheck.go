package service

import (
	"bytes"
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HealthCheckResult struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	Platform    string `json:"platform"`
	Healthy     bool   `json:"healthy"`
	LatencyMs   int64  `json:"latency_ms"`
	Error       string `json:"error,omitempty"`
}

type HealthCheckService struct {
	db      *database.DB
	account *AccountService
	gateway *GatewayService
}

func NewHealthCheckService(db *database.DB) *HealthCheckService {
	return &HealthCheckService{
		db:      db,
		account: NewAccountService(db),
		gateway: NewGatewayService(db),
	}
}

func defaultModel(platform string) string {
	switch platform {
	case "claude":
		return "claude-3-5-haiku-20241022"
	case "openai":
		return "gpt-5.4"
	case "gemini":
		return "gemini-2.0-flash"
	default:
		return ""
	}
}

func (h *HealthCheckService) CheckAccount(accountID int64, modelName string) (*HealthCheckResult, error) {
	acc, err := h.account.GetByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("账号不存在: %w", err)
	}

	if modelName == "" {
		modelName = defaultModel(acc.Platform)
	}

	result := &HealthCheckResult{
		AccountID:   acc.ID,
		AccountName: acc.Name,
		Platform:    acc.Platform,
	}

	start := time.Now()
	resp, err := h.sendTestRequest(acc, modelName)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		h.account.UpdateStatus(acc.ID, "error", &result.Error)
		return result, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		h.account.UpdateStatus(acc.ID, "error", &result.Error)
		return result, nil
	}

	result.Healthy = true
	h.account.UpdateStatus(acc.ID, "active", nil)
	return result, nil
}

func (h *HealthCheckService) CheckAllAccounts(modelName string) ([]HealthCheckResult, error) {
	accounts, err := h.account.List()
	if err != nil {
		return nil, err
	}

	results := make([]HealthCheckResult, 0, len(accounts))
	for _, acc := range accounts {
		m := modelName
		if m == "" {
			m = defaultModel(acc.Platform)
		}
		r, err := h.CheckAccount(acc.ID, m)
		if err != nil {
			results = append(results, HealthCheckResult{
				AccountID:   acc.ID,
				AccountName: acc.Name,
				Platform:    acc.Platform,
				Error:       err.Error(),
			})
			continue
		}
		results = append(results, *r)
	}
	return results, nil
}

func (h *HealthCheckService) sendTestRequest(acc *model.Account, modelName string) (*http.Response, error) {
	switch acc.Platform {
	case "claude":
		return h.sendClaudeRequest(acc, modelName)
	case "openai":
		return h.sendOpenAIRequest(acc, modelName)
	case "gemini":
		return h.sendGeminiRequest(acc, modelName)
	default:
		return nil, fmt.Errorf("不支持的平台: %s", acc.Platform)
	}
}

func (h *HealthCheckService) sendClaudeRequest(acc *model.Account, modelName string) (*http.Response, error) {
	baseURL := getBaseURL(acc, "https://api.anthropic.com/v1")
	targetURL := baseURL + "/messages"

	body := map[string]any{
		"model":      modelName,
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "你好"}},
	}
	bodyBytes, _ := json.Marshal(body)

	headers := map[string]string{
		"Content-Type":      "application/json",
		"anthropic-version": "2023-06-01",
	}
	switch acc.Type {
	case "api_key":
		if key, ok := acc.Credentials["api_key"].(string); ok {
			headers["x-api-key"] = key
		}
	case "oauth":
		if token, ok := acc.Credentials["access_token"].(string); ok {
			headers["Authorization"] = "Bearer " + token
		}
	case "cookie":
		if sk, ok := acc.Credentials["session_key"].(string); ok {
			headers["Cookie"] = "sessionKey=" + sk
		}
	}

	return h.doRequestWithTimeout(acc, "POST", targetURL, headers, bodyBytes, 30*time.Second)
}

func (h *HealthCheckService) sendOpenAIRequest(acc *model.Account, modelName string) (*http.Response, error) {
	baseURL := getBaseURL(acc, "https://api.openai.com/v1")
	targetURL := baseURL + "/chat/completions"

	body := map[string]any{
		"model":      modelName,
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "你好"}},
	}
	bodyBytes, _ := json.Marshal(body)

	headers := map[string]string{"Content-Type": "application/json"}
	switch acc.Type {
	case "api_key":
		if key, ok := acc.Credentials["api_key"].(string); ok {
			headers["Authorization"] = "Bearer " + key
		}
	case "oauth":
		if token, ok := acc.Credentials["access_token"].(string); ok {
			headers["Authorization"] = "Bearer " + token
		}
	}

	return h.doRequestWithTimeout(acc, "POST", targetURL, headers, bodyBytes, 30*time.Second)
}

func (h *HealthCheckService) sendGeminiRequest(acc *model.Account, modelName string) (*http.Response, error) {
	baseURL := getBaseURL(acc, "https://generativelanguage.googleapis.com")
	targetURL := baseURL + "/v1beta/models/" + modelName + ":generateContent"

	if acc.Type == "api_key" {
		if key, ok := acc.Credentials["api_key"].(string); ok {
			targetURL += "?key=" + key
		}
	}

	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": "你好"}}},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	headers := map[string]string{"Content-Type": "application/json"}
	if acc.Type == "oauth" {
		if token, ok := acc.Credentials["access_token"].(string); ok {
			headers["Authorization"] = "Bearer " + token
		}
	}

	return h.doRequestWithTimeout(acc, "POST", targetURL, headers, bodyBytes, 30*time.Second)
}

func (h *HealthCheckService) doRequestWithTimeout(acc *model.Account, method, targetURL string, headers map[string]string, body []byte, timeout time.Duration) (*http.Response, error) {
	// Use a transport clone (same as GatewayService.DoRequest) but with custom timeout
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if acc.ProxyID != nil {
		ps := NewProxyService(h.db)
		p, err := ps.GetByID(*acc.ProxyID)
		if err == nil && p != nil {
			proxyURL, _ := parseProxyURL(p)
			if proxyURL != nil {
				transport.Proxy = http.ProxyURL(proxyURL)
			}
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	req, err := http.NewRequest(method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return client.Do(req)
}

func getBaseURL(account *model.Account, defaultURL string) string {
	if account.BaseURL != nil && *account.BaseURL != "" {
		return strings.TrimRight(*account.BaseURL, "/")
	}
	return defaultURL
}

func parseProxyURL(p *model.Proxy) (*url.URL, error) {
	raw := fmt.Sprintf("%s://%s:%d", p.Protocol, p.Host, p.Port)
	u, err := url.Parse(raw)
	if u != nil && p.Username != nil && *p.Username != "" {
		u.User = url.UserPassword(*p.Username, *p.Password)
	}
	return u, err
}
