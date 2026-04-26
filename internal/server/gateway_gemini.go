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

func (s *Server) handleGemini(c *gin.Context) {
	apiKey, _ := c.Get("api_key")
	group, _ := c.Get("group")

	path := c.Param("path")

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Failed to read body"}})
		return
	}
	c.Request.Body.Close()

	modelName := extractGeminiModel(path)

	var groupID int64
	if grp, ok := group.(*model.Group); ok && grp != nil {
		groupID = grp.ID
	}

	ctx := &service.RequestContext{
		APIKey:         apiKey.(*model.APIKey),
		Group:          group.(*model.Group),
		Platform:       "gemini",
		StartTime:      time.Now(),
		RequestID:      fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Model:          modelName,
		RequestedModel: modelName,
	}

	maxRetries := s.cfg.Gateway.MaxAccountRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	resp, acc, err := s.gateway.DoWithRetry(groupID, "gemini", maxRetries, func(account *model.Account) (*http.Response, error) {
		ctx.Account = account

		baseURL := getBaseURL(account, "https://generativelanguage.googleapis.com")
		targetURL := baseURL + path
		if account.Type == "api_key" {
			key, _ := account.Credentials["api_key"].(string)
			if strings.Contains(targetURL, "?") {
				targetURL += "&key=" + key
			} else {
				targetURL += "?key=" + key
			}
		}

		headers := map[string]string{"Content-Type": "application/json"}
		if account.Type == "oauth" {
			if token, ok := account.Credentials["access_token"].(string); ok {
				headers["Authorization"] = "Bearer " + token
			}
		}

		return s.gateway.DoRequest(account, c.Request.Method, targetURL, headers, bytes.NewReader(bodyBytes))
	})

	if err != nil {
		log.Printf("[gemini] all retries failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	_ = acc

	if strings.Contains(path, "streamGenerateContent") {
		s.gateway.StreamResponse(c.Writer, resp, ctx)
		return
	}

	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var respObj map[string]any
	json.Unmarshal(respBody, &respObj)

	inputTokens, outputTokens := 0, 0
	if metadata, ok := respObj["usageMetadata"].(map[string]any); ok {
		inputTokens = service.ExtractIntField(metadata, "promptTokenCount")
		outputTokens = service.ExtractIntField(metadata, "candidatesTokenCount")
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

func extractGeminiModel(path string) string {
	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) >= 2 {
		modelPart := parts[1]
		if idx := strings.Index(modelPart, ":"); idx >= 0 {
			return modelPart[:idx]
		}
		return modelPart
	}
	return "gemini-2.0-flash"
}
