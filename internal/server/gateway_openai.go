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
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleOpenAIChat(c *gin.Context) {
	apiKey, _ := c.Get("api_key")
	group, _ := c.Get("group")

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Failed to read body"}})
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

	platform := "openai"
	if isClaudeModel(requestedModel) {
		platform = "claude"
	} else if isGeminiModel(requestedModel) {
		platform = "gemini"
	}

	var groupID int64
	if grp, ok := group.(*model.Group); ok && grp != nil {
		groupID = grp.ID
	}

	ctx := &service.RequestContext{
		APIKey:         apiKey.(*model.APIKey),
		Group:          group.(*model.Group),
		Platform:       platform,
		StartTime:      time.Now(),
		RequestID:      fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Stream:         stream,
		Model:          requestedModel,
		RequestedModel: requestedModel,
	}

	maxRetries := s.cfg.Gateway.MaxAccountRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	resp, acc, err := s.gateway.DoWithRetry(groupID, platform, maxRetries, func(account *model.Account) (*http.Response, error) {
		ctx.Account = account
		switch account.Platform {
		case "claude":
			claudeBody := convertOpenAIToClaude(reqBody)
			ctx.Model = resolveClaudeModel(requestedModel)
			claudeBody["model"] = ctx.Model
			modifiedBody, _ := json.Marshal(claudeBody)
			claudeURL := getBaseURL(account, "https://api.anthropic.com/v1") + "/messages"
			return s.gateway.DoRequest(account, "POST", claudeURL, claudeHeaders(account), bytes.NewReader(modifiedBody))
		case "gemini":
			key, _ := account.Credentials["api_key"].(string)
			geminiBase := getBaseURL(account, "https://generativelanguage.googleapis.com/v1beta")
			targetURL := geminiBase + "/models/" + requestedModel + ":generateContent?key=" + key
			modifiedBody, _ := json.Marshal(reqBody)
			return s.gateway.DoRequest(account, "POST", targetURL, geminiHeaders(account), bytes.NewReader(modifiedBody))
		default:
			modifiedBody, _ := json.Marshal(reqBody)
			openaiURL := getBaseURL(account, "https://api.openai.com/v1") + "/chat/completions"
			return s.gateway.DoRequest(account, "POST", openaiURL, openaiHeaders(account), bytes.NewReader(modifiedBody))
		}
	})

	if err != nil {
		log.Printf("[openai-chat] all retries failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": err.Error()}})
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
	inputTokens, outputTokens := 0, 0
	if usage, ok := respObj["usage"].(map[string]any); ok {
		inputTokens = service.ExtractIntField(usage, "prompt_tokens")
		outputTokens = service.ExtractIntField(usage, "completion_tokens")
	}

	duration := time.Since(ctx.StartTime).Milliseconds()
	s.gateway.LogUsage(ctx.APIKey, ctx.Group, ctx.Account, ctx.RequestID, ctx.Model, ctx.RequestedModel,
		inputTokens, outputTokens, 0, 0, duration, false, resp)

	for k, vs := range resp.Header {
		for _, v := range vs {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(respBody)
}

func (s *Server) handleOpenAIResponses(c *gin.Context) {
	apiKey, _ := c.Get("api_key")
	group, _ := c.Get("group")

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Failed to read body"}})
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
	if grp, ok := group.(*model.Group); ok && grp != nil {
		groupID = grp.ID
	}

	ctx := &service.RequestContext{
		APIKey:         apiKey.(*model.APIKey),
		Group:          group.(*model.Group),
		Platform:       "openai",
		StartTime:      time.Now(),
		RequestID:      fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Stream:         stream,
		Model:          requestedModel,
		RequestedModel: requestedModel,
	}

	maxRetries := s.cfg.Gateway.MaxAccountRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	resp, acc, err := s.gateway.DoWithRetry(groupID, "openai", maxRetries, func(account *model.Account) (*http.Response, error) {
		ctx.Account = account
		modifiedBody, _ := json.Marshal(reqBody)
		responsesURL := getBaseURL(account, "https://api.openai.com/v1") + "/responses"
		return s.gateway.DoRequest(account, "POST", responsesURL, openaiHeaders(account), bytes.NewReader(modifiedBody))
	})

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": err.Error()}})
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
	inputTokens, outputTokens := 0, 0
	if usage, ok := respObj["usage"].(map[string]any); ok {
		inputTokens = service.ExtractIntField(usage, "input_tokens")
		outputTokens = service.ExtractIntField(usage, "output_tokens")
	}

	duration := time.Since(ctx.StartTime).Milliseconds()
	s.gateway.LogUsage(ctx.APIKey, ctx.Group, ctx.Account, ctx.RequestID, ctx.Model, ctx.RequestedModel,
		inputTokens, outputTokens, 0, 0, duration, false, resp)

	for k, vs := range resp.Header {
		for _, v := range vs {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(respBody)
}

func (s *Server) handleOpenAIImages(c *gin.Context) {
	apiKey, _ := c.Get("api_key")
	group, _ := c.Get("group")

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Failed to read body"}})
		return
	}
	c.Request.Body.Close()

	var reqBody map[string]any
	json.Unmarshal(bodyBytes, &reqBody)

	var groupID int64
	if grp, ok := group.(*model.Group); ok && grp != nil {
		groupID = grp.ID
	}

	ctx := &service.RequestContext{
		APIKey:    apiKey.(*model.APIKey),
		Group:     group.(*model.Group),
		Platform:  "openai",
		StartTime: time.Now(),
		RequestID: fmt.Sprintf("req_%d", time.Now().UnixNano()),
	}

	resp, _, err := s.gateway.DoWithRetry(groupID, "openai", 3, func(account *model.Account) (*http.Response, error) {
		ctx.Account = account
		modifiedBody, _ := json.Marshal(reqBody)
		imagesURL := getBaseURL(account, "https://api.openai.com/v1") + "/images/generations"
		return s.gateway.DoRequest(account, "POST", imagesURL, openaiHeaders(account), bytes.NewReader(modifiedBody))
	})

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	for k, vs := range resp.Header {
		for _, v := range vs {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(respBody)
}

func (s *Server) handleModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": []map[string]any{
		{"id": "claude-sonnet-4-6-20250514", "object": "model", "owned_by": "anthropic"},
		{"id": "claude-haiku-4-5-20251001", "object": "model", "owned_by": "anthropic"},
		{"id": "claude-3-5-sonnet-20241022", "object": "model", "owned_by": "anthropic"},
		{"id": "claude-opus-4-7-20250219", "object": "model", "owned_by": "anthropic"},
		{"id": "gpt-4o", "object": "model", "owned_by": "openai"},
		{"id": "gpt-4o-mini", "object": "model", "owned_by": "openai"},
		{"id": "o3-mini", "object": "model", "owned_by": "openai"},
		{"id": "o4-mini", "object": "model", "owned_by": "openai"},
		{"id": "gemini-2.0-flash", "object": "model", "owned_by": "google"},
		{"id": "gemini-2.5-flash-preview-05-20", "object": "model", "owned_by": "google"},
		{"id": "gemini-2.5-pro-preview-05-06", "object": "model", "owned_by": "google"},
	}})
}

func openaiHeaders(account *model.Account) map[string]string {
	h := map[string]string{"Content-Type": "application/json"}
	switch account.Type {
	case "api_key":
		if key, ok := account.Credentials["api_key"].(string); ok {
			h["Authorization"] = "Bearer " + key
		}
		if org, ok := account.Credentials["organization_id"].(string); ok && org != "" {
			h["OpenAI-Organization"] = org
		}
	case "oauth":
		if token, ok := account.Credentials["access_token"].(string); ok {
			h["Authorization"] = "Bearer " + token
		}
	}
	return h
}

func claudeHeaders(account *model.Account) map[string]string {
	h := map[string]string{"Content-Type": "application/json", "anthropic-version": "2023-06-01"}
	switch account.Type {
	case "api_key":
		if key, ok := account.Credentials["api_key"].(string); ok {
			h["x-api-key"] = key
		}
	case "oauth":
		if token, ok := account.Credentials["access_token"].(string); ok {
			h["Authorization"] = "Bearer " + token
		}
	case "cookie":
		if sk, ok := account.Credentials["session_key"].(string); ok {
			h["Cookie"] = "sessionKey=" + sk
		}
	}
	return h
}

func geminiHeaders(account *model.Account) map[string]string {
	h := map[string]string{"Content-Type": "application/json"}
	if account.Type == "oauth" {
		if token, ok := account.Credentials["access_token"].(string); ok {
			h["Authorization"] = "Bearer " + token
		}
	}
	return h
}

func convertOpenAIToClaude(oai map[string]any) map[string]any {
	claude := map[string]any{"model": oai["model"], "max_tokens": 4096}
	if mt, ok := oai["max_tokens"].(float64); ok {
		claude["max_tokens"] = int(mt)
	}
	if msgs, ok := oai["messages"].([]any); ok {
		var claudeMsgs []map[string]any
		var systemPrompt string
		for _, m := range msgs {
			msg, ok := m.(map[string]any)
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			content, _ := msg["content"].(string)
			if role == "system" {
				systemPrompt = content
				continue
			}
			claudeMsgs = append(claudeMsgs, map[string]any{"role": role, "content": content})
		}
		claude["messages"] = claudeMsgs
		if systemPrompt != "" {
			claude["system"] = systemPrompt
		}
	}
	if stream, ok := oai["stream"].(bool); ok && stream {
		claude["stream"] = true
	}
	return claude
}
