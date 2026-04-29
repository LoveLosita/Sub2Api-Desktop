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

	resp, acc, err := s.gateway.DoWithRetryAnyPlatform(groupID, maxRetries, func(account *model.Account) (*http.Response, error) {
		ctx.Account = account
		switch account.Platform {
		case "claude":
			return s.sendToClaude(reqBody, ctx, account)
		case "openai":
			return s.sendClaudeRequestToOpenAI(reqBody, ctx, account, requestedModel)
		default:
			return s.sendToClaude(reqBody, ctx, account)
		}
	})

	if err != nil {
		log.Printf("[claude] all retries failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "api_error", "message": err.Error()}})
		return
	}

	switch acc.Platform {
	case "openai":
		s.handleOpenAIResponseAsClaude(c, resp, ctx, stream, requestedModel)
	default:
		s.handleClaudeResponseDirect(c, resp, ctx, stream)
	}
}

func (s *Server) sendToClaude(reqBody map[string]any, ctx *service.RequestContext, account *model.Account) (*http.Response, error) {
	baseURL := getBaseURL(account, "https://api.anthropic.com/v1")
	targetURL := baseURL + "/messages"
	headers := claudeHeaders(account)
	body := make(map[string]any, len(reqBody))
	for k, v := range reqBody {
		body[k] = v
	}
	body["model"] = ctx.Model
	modifiedBody, _ := json.Marshal(body)
	return s.gateway.DoRequest(account, "POST", targetURL, headers, bytes.NewReader(modifiedBody))
}

func (s *Server) sendClaudeRequestToOpenAI(claudeReq map[string]any, ctx *service.RequestContext, account *model.Account, requestedModel string) (*http.Response, error) {
	oaiReq := convertClaudeToOpenAIRequest(claudeReq)
	oaiReq["model"] = requestedModel
	ctx.Model = requestedModel
	modifiedBody, _ := json.Marshal(oaiReq)
	openaiURL := getBaseURL(account, "https://api.openai.com/v1") + "/chat/completions"
	log.Printf("[claude→openai] account=%s model=%s url=%s", account.Name, requestedModel, openaiURL)
	resp, err := s.gateway.DoRequest(account, "POST", openaiURL, openaiHeaders(account), bytes.NewReader(modifiedBody))
	if err != nil {
		log.Printf("[claude→openai] connection error: %v", err)
	} else {
		log.Printf("[claude→openai] status=%d", resp.StatusCode)
	}
	return resp, err
}

func (s *Server) handleClaudeResponseDirect(c *gin.Context, resp *http.Response, ctx *service.RequestContext, stream bool) {
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

func (s *Server) handleOpenAIResponseAsClaude(c *gin.Context, resp *http.Response, ctx *service.RequestContext, stream bool, model string) {
	if stream {
		converter := NewOpenAIToClaudeStreamConverter(model)
		s.gateway.StreamResponseWithConverter(c.Writer, resp, ctx, converter.Convert)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var oaiResp map[string]any
	json.Unmarshal(respBody, &oaiResp)

	// Extract usage for billing from OpenAI format
	inputTokens, outputTokens := 0, 0
	if usage, ok := oaiResp["usage"].(map[string]any); ok {
		inputTokens = service.ExtractIntField(usage, "prompt_tokens")
		outputTokens = service.ExtractIntField(usage, "completion_tokens")
	}

	duration := time.Since(ctx.StartTime).Milliseconds()
	s.gateway.LogUsage(ctx.APIKey, ctx.Group, ctx.Account, ctx.RequestID, ctx.Model, ctx.RequestedModel,
		inputTokens, outputTokens, 0, 0, duration, false, resp)

	// Convert response to Claude format
	claudeResp := convertOpenAIResponseToClaude(oaiResp, model)
	claudeBody, _ := json.Marshal(claudeResp)

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(claudeBody)
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
