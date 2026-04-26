package server

import (
	"bytes"
	"desktop-proxy/internal/model"
	"desktop-proxy/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleClaude(c *gin.Context) {
	apiKey, _ := c.Get("api_key")
	group, _ := c.Get("group")

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to read body"}})
		return
	}
	c.Request.Body.Close()

	var reqBody map[string]any
	json.Unmarshal(bodyBytes, &reqBody)

	requestedModel, _ := reqBody["model"].(string)
	stream := false
	if sv, ok := reqBody["stream"].(bool); ok {
		stream = sv
	}

	var groupID int64
	platform := "claude"
	if grp, ok := group.(*model.Group); ok && grp != nil {
		groupID = grp.ID
		platform = grp.Platform
	}

	ctx := &service.RequestContext{
		APIKey:         apiKey.(*model.APIKey),
		Group:          group.(*model.Group),
		Platform:       platform,
		StartTime:      time.Now(),
		RequestID:      fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Stream:         stream,
		Model:          resolveClaudeModel(requestedModel),
		RequestedModel: requestedModel,
	}

	maxRetries := s.cfg.Gateway.MaxAccountRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	resp, acc, err := s.gateway.DoWithRetry(groupID, "claude", maxRetries, func(account *model.Account) (*http.Response, error) {
		ctx.Account = account
		baseURL := getBaseURL(account, "https://api.anthropic.com/v1")
		targetURL := baseURL + "/messages"
		headers := map[string]string{
			"Content-Type":      "application/json",
			"anthropic-version": "2023-06-01",
		}

		switch account.Type {
		case "api_key":
			if key, ok := account.Credentials["api_key"].(string); ok {
				headers["x-api-key"] = key
			}
		case "oauth":
			if token, ok := account.Credentials["access_token"].(string); ok {
				headers["Authorization"] = "Bearer " + token
			}
		case "cookie":
			if sk, ok := account.Credentials["session_key"].(string); ok {
				headers["Cookie"] = "sessionKey=" + sk
			}
		}

		reqBody["model"] = ctx.Model
		modifiedBody, _ := json.Marshal(reqBody)
		return s.gateway.DoRequest(account, "POST", targetURL, headers, bytes.NewReader(modifiedBody))
	})

	if err != nil {
		log.Printf("[claude] all retries failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "api_error", "message": err.Error()}})
		return
	}
	_ = acc

	if stream {
		s.gateway.StreamResponse(c.Writer, resp, ctx)
		return
	}

	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var respObj map[string]any
	json.Unmarshal(respBody, &respObj)

	inputTokens, outputTokens, cacheCreation, cacheRead := 0, 0, 0, 0
	if usage, ok := respObj["usage"].(map[string]any); ok {
		inputTokens = service.ExtractIntField(usage, "input_tokens")
		outputTokens = service.ExtractIntField(usage, "output_tokens")
		cacheCreation = service.ExtractIntField(usage, "cache_creation_input_tokens")
		cacheRead = service.ExtractIntField(usage, "cache_read_input_tokens")
	}

	duration := time.Since(ctx.StartTime).Milliseconds()
	s.gateway.LogUsage(ctx.APIKey, ctx.Group, ctx.Account, ctx.RequestID, ctx.Model, ctx.RequestedModel,
		inputTokens, outputTokens, cacheCreation, cacheRead, duration, false, resp)

	for k, vs := range resp.Header {
		for _, v := range vs {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(respBody)
}

func resolveClaudeModel(requested string) string {
	aliases := map[string]string{
		"claude-3.5-sonnet": "claude-3-5-sonnet-20241022",
		"claude-3.5-haiku":  "claude-3-5-haiku-20241022",
		"claude-3-opus":     "claude-3-opus-20240229",
		"claude-3-haiku":    "claude-3-haiku-20240307",
		"claude-sonnet-4-6": "claude-sonnet-4-6-20250514",
		"claude-haiku-4-5":  "claude-haiku-4-5-20251001",
		"claude-opus-4-7":   "claude-opus-4-7-20250219",
	}
	if r, ok := aliases[requested]; ok {
		return r
	}
	if requested != "" {
		return requested
	}
	return "claude-sonnet-4-6-20250514"
}

func isClaudeModel(model string) bool {
	return strings.HasPrefix(model, "claude-") || strings.HasPrefix(model, "claude_")
}

func isGeminiModel(model string) bool {
	return strings.HasPrefix(model, "gemini-") || strings.HasPrefix(model, "gemini_")
}
