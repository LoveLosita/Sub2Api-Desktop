package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type GatewayService struct {
	db        *database.DB
	usage     *UsageService
	account   *AccountService
	scheduler *SchedulerService
}

func NewGatewayService(db *database.DB) *GatewayService {
	return &GatewayService{
		db:        db,
		usage:     NewUsageService(db),
		account:   NewAccountService(db),
		scheduler: NewSchedulerService(db),
	}
}

// RequestContext holds state for a single proxy request (used by server package).
type RequestContext struct {
	APIKey         *model.APIKey
	Group          *model.Group
	Account        *model.Account
	Platform       string
	StartTime      time.Time
	RequestID      string
	Stream         bool
	Model          string
	RequestedModel string
}

// DoRequest executes a single upstream HTTP request with optional proxy.
func (g *GatewayService) DoRequest(account *model.Account, method, targetURL string, headers map[string]string, body io.Reader) (*http.Response, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if account.ProxyID != nil {
		ps := NewProxyService(g.db)
		p, err := ps.GetByID(*account.ProxyID)
		if err == nil && p != nil {
			proxyURL, _ := url.Parse(fmt.Sprintf("%s://%s:%d", p.Protocol, p.Host, p.Port))
			if proxyURL != nil && p.Username != nil && *p.Username != "" {
				proxyURL.User = url.UserPassword(*p.Username, *p.Password)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	client := &http.Client{Transport: transport, Timeout: 300 * time.Second}
	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return client.Do(req)
}

// DoWithRetry tries multiple accounts for a group with failover.
func (g *GatewayService) DoWithRetry(groupID int64, platform string, maxRetries int,
	fn func(account *model.Account) (*http.Response, error)) (*http.Response, *model.Account, error) {

	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		acc, err := g.scheduler.PickAccount(groupID, platform)
		if err != nil {
			return nil, nil, fmt.Errorf("no available account: %w", err)
		}

		resp, err := fn(acc)
		if err != nil {
			log.Printf("[gateway] attempt %d: request error: %v", attempt+1, err)
			g.account.MarkUsed(acc.ID)
			continue
		}

		if resp.StatusCode == 429 {
			log.Printf("[gateway] account %s rate limited", acc.Name)
			resetAt := time.Now().Add(60 * time.Second)
			g.account.UpdateScheduling(acc.ID, acc.Schedulable, &resetAt, &resetAt, nil)
			resp.Body.Close()
			continue
		}

		if resp.StatusCode == 529 {
			log.Printf("[gateway] account %s overloaded", acc.Name)
			until := time.Now().Add(30 * time.Second)
			g.account.UpdateScheduling(acc.ID, acc.Schedulable, nil, nil, &until)
			resp.Body.Close()
			continue
		}

		g.account.MarkUsed(acc.ID)
		return resp, acc, nil
	}

	return nil, nil, fmt.Errorf("all %d attempts failed", maxRetries)
}

// LogUsage records a usage log entry.
func (g *GatewayService) LogUsage(apiKey *model.APIKey, group *model.Group, account *model.Account,
	requestID, modelName, requestedModel string,
	inputTokens, outputTokens, cacheCreation, cacheRead int,
	durationMs int64, stream bool, resp *http.Response) {

	costs := CalculateCost(modelName, inputTokens, outputTokens, cacheCreation, cacheRead)

	var groupID *int64
	var apiKeyID *int64
	if group != nil {
		groupID = &group.ID
	}
	if apiKey != nil {
		apiKeyID = &apiKey.ID
	}

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	errType := ""
	if statusCode >= 400 {
		errType = http.StatusText(statusCode)
	}

	usageLog := &model.UsageLog{
		RequestID:           requestID,
		APIKeyID:            apiKeyID,
		AccountID:           account.ID,
		GroupID:             groupID,
		Model:               modelName,
		RequestedModel:      strPtr(requestedModel),
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		CacheCreationTokens: cacheCreation,
		CacheReadTokens:     cacheRead,
		InputCost:           costs.InputCost,
		OutputCost:          costs.OutputCost,
		CacheCreationCost:   costs.CacheCreationCost,
		CacheReadCost:       costs.CacheReadCost,
		TotalCost:           costs.Total(),
		Stream:              stream,
		DurationMs:          intPtr(int(durationMs)),
		StatusCode:          intPtr(statusCode),
		ErrorType:           strPtr(errType),
	}

	if err := g.usage.Log(usageLog); err != nil {
		log.Printf("[gateway] failed to log usage: %v", err)
	}
}

// StreamResponse proxies a streaming response (SSE) to the client and logs usage.
func (g *GatewayService) StreamResponse(w http.ResponseWriter, resp *http.Response, ctx *RequestContext) {
	defer resp.Body.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	var totalOutputTokens int
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
			chunk := string(buf[:n])
			if strings.Contains(chunk, "usage") {
				totalOutputTokens = extractOutputTokens(chunk, totalOutputTokens)
			}
		}
		if err != nil {
			break
		}
	}

	duration := time.Since(ctx.StartTime).Milliseconds()
	var groupID *int64
	var apiKeyID *int64
	if ctx.Group != nil {
		groupID = &ctx.Group.ID
	}
	if ctx.APIKey != nil {
		apiKeyID = &ctx.APIKey.ID
	}

	statusCode := resp.StatusCode
	errType := ""
	if statusCode >= 400 {
		errType = http.StatusText(statusCode)
	}

	costs := CalculateCost(ctx.Model, 0, totalOutputTokens, 0, 0)
	usageLog := &model.UsageLog{
		RequestID:      ctx.RequestID,
		APIKeyID:       apiKeyID,
		AccountID:      ctx.Account.ID,
		GroupID:        groupID,
		Model:          ctx.Model,
		RequestedModel: strPtr(ctx.RequestedModel),
		OutputTokens:   totalOutputTokens,
		OutputCost:     costs.OutputCost,
		TotalCost:      costs.OutputCost,
		Stream:         true,
		DurationMs:     intPtr(int(duration)),
		StatusCode:     intPtr(statusCode),
		ErrorType:      strPtr(errType),
	}
	if logErr := g.usage.Log(usageLog); logErr != nil {
		log.Printf("[gateway] failed to log stream usage: %v", logErr)
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func extractOutputTokens(chunk string, current int) int {
	for _, line := range strings.Split(chunk, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(data), &obj) != nil {
			continue
		}
		if usage, ok := obj["usage"].(map[string]any); ok {
			if ot, ok := usage["output_tokens"].(float64); ok && int(ot) > current {
				return int(ot)
			}
		}
		if msg, ok := obj["message"].(map[string]any); ok {
			if usage, ok := msg["usage"].(map[string]any); ok {
				if ot, ok := usage["output_tokens"].(float64); ok && int(ot) > current {
					return int(ot)
				}
			}
		}
	}
	return current
}

// ExtractIntField safely extracts an int from a map.
func ExtractIntField(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		case string:
			i, _ := strconv.Atoi(n)
			return i
		}
	}
	return 0
}
