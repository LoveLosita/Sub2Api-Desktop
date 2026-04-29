package service

import (
	"bufio"
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

func (g *GatewayService) SetOnUsageLogged(fn func()) {
	g.usage.OnLog = fn
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

	client := &http.Client{Transport: transport}
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
// Supports: connection errors, 429, 529, 5xx, 401/403 auto-switch.
// Failed accounts are tracked and excluded from subsequent picks.
func (g *GatewayService) DoWithRetry(groupID int64, platform string, maxRetries int,
	fn func(account *model.Account) (*http.Response, error)) (*http.Response, *model.Account, error) {

	if maxRetries <= 0 {
		maxRetries = 3
	}

	failedIDs := make(map[int64]bool)

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Pick account, skipping previously failed ones
		acc, err := g.scheduler.PickAccount(groupID, platform)
		if err != nil {
			return nil, nil, fmt.Errorf("no available account: %w", err)
		}
		if failedIDs[acc.ID] {
			// Try to find another account that hasn't failed
			altFound := false
			for i := 0; i < 5; i++ {
				acc2, err2 := g.scheduler.PickAccount(groupID, platform)
				if err2 != nil {
					break
				}
				if !failedIDs[acc2.ID] {
					acc = acc2
					altFound = true
					break
				}
			}
			if !altFound {
				return nil, nil, fmt.Errorf("all accounts exhausted (attempt %d/%d)", attempt+1, maxRetries)
			}
		}

		resp, err := fn(acc)
		if err != nil {
			log.Printf("[gateway] attempt %d: account %s connection error: %v", attempt+1, acc.Name, err)
			failedIDs[acc.ID] = true
			continue
		}

		// Rate limited — cooldown and switch
		if resp.StatusCode == 429 {
			log.Printf("[gateway] account %s rate limited (429)", acc.Name)
			resetAt := time.Now().Add(60 * time.Second)
			g.account.UpdateScheduling(acc.ID, acc.Schedulable, &resetAt, &resetAt, nil)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		// Overloaded — cooldown and switch
		if resp.StatusCode == 529 {
			log.Printf("[gateway] account %s overloaded (529)", acc.Name)
			until := time.Now().Add(30 * time.Second)
			g.account.UpdateScheduling(acc.ID, acc.Schedulable, nil, nil, &until)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		// Auth errors — switch to different account
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			log.Printf("[gateway] account %s auth error (%d), marking as error", acc.Name, resp.StatusCode)
			errMsg := fmt.Sprintf("认证失败: HTTP %d", resp.StatusCode)
			g.account.UpdateStatus(acc.ID, "error", &errMsg)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		// Server errors — retry with different account
		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			log.Printf("[gateway] account %s server error (%d), switching", acc.Name, resp.StatusCode)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		// Success or non-retryable client error (400, 404, etc.)
		g.account.MarkUsed(acc.ID)
		return resp, acc, nil
	}

	return nil, nil, fmt.Errorf("all %d attempts failed", maxRetries)
}

// DoWithRetryAnyPlatform is the same as DoWithRetry but picks accounts of any platform.
func (g *GatewayService) DoWithRetryAnyPlatform(groupID int64, maxRetries int,
	fn func(account *model.Account) (*http.Response, error)) (*http.Response, *model.Account, error) {

	if maxRetries <= 0 {
		maxRetries = 3
	}

	failedIDs := make(map[int64]bool)
	var lastErr string

	for attempt := 0; attempt < maxRetries; attempt++ {
		acc, err := g.scheduler.PickAccountAnyPlatform(groupID)
		if err != nil {
			return nil, nil, fmt.Errorf("no available account: %w", err)
		}
		if failedIDs[acc.ID] {
			altFound := false
			for i := 0; i < 5; i++ {
				acc2, err2 := g.scheduler.PickAccountAnyPlatform(groupID)
				if err2 != nil {
					break
				}
				if !failedIDs[acc2.ID] {
					acc = acc2
					altFound = true
					break
				}
			}
			if !altFound {
				return nil, nil, fmt.Errorf("all accounts exhausted (attempt %d/%d), last error: %s", attempt+1, maxRetries, lastErr)
			}
		}

		resp, err := fn(acc)
		if err != nil {
			lastErr = fmt.Sprintf("account %s (platform=%s) connection error: %v", acc.Name, acc.Platform, err)
			log.Printf("[gateway] attempt %d: %s", attempt+1, lastErr)
			failedIDs[acc.ID] = true
			continue
		}

		if resp.StatusCode == 429 {
			lastErr = fmt.Sprintf("account %s (platform=%s) rate limited (429)", acc.Name, acc.Platform)
			log.Printf("[gateway] %s", lastErr)
			resetAt := time.Now().Add(60 * time.Second)
			g.account.UpdateScheduling(acc.ID, acc.Schedulable, &resetAt, &resetAt, nil)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		if resp.StatusCode == 529 {
			lastErr = fmt.Sprintf("account %s (platform=%s) overloaded (529)", acc.Name, acc.Platform)
			log.Printf("[gateway] %s", lastErr)
			until := time.Now().Add(30 * time.Second)
			g.account.UpdateScheduling(acc.ID, acc.Schedulable, nil, nil, &until)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			lastErr = fmt.Sprintf("account %s (platform=%s) auth error (%d)", acc.Name, acc.Platform, resp.StatusCode)
			log.Printf("[gateway] %s", lastErr)
			errMsg := fmt.Sprintf("认证失败: HTTP %d", resp.StatusCode)
			g.account.UpdateStatus(acc.ID, "error", &errMsg)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			lastErr = fmt.Sprintf("account %s (platform=%s) server error (%d)", acc.Name, acc.Platform, resp.StatusCode)
			log.Printf("[gateway] %s", lastErr)
			resp.Body.Close()
			failedIDs[acc.ID] = true
			continue
		}

		g.account.MarkUsed(acc.ID)
		return resp, acc, nil
	}

	return nil, nil, fmt.Errorf("all %d attempts failed, last error: %s", maxRetries, lastErr)
}

// LogUsage records a usage log entry.
func (g *GatewayService) LogUsage(apiKey *model.APIKey, group *model.Group, account *model.Account,
	requestID, modelName, requestedModel string,
	inputTokens, outputTokens, cacheCreation, cacheRead int,
	durationMs int64, stream bool, resp *http.Response) {

	costs := CalculateCost(modelName, inputTokens, outputTokens, cacheCreation, cacheRead)

	m := account.Multiplier
	if m <= 0 {
		m = 1.0
	}
	if m != 1.0 {
		costs.InputCost *= m
		costs.OutputCost *= m
		costs.CacheCreationCost *= m
		costs.CacheReadCost *= m
	}

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

// scanEvent represents a single line read from upstream SSE.
type scanEvent struct {
	line string
	err  error
}

// StreamResponse proxies a streaming response (SSE) to the client and logs usage.
// Uses goroutine + bufio.Scanner + channel pattern (adapted from sub2api).
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
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(resp.StatusCode)

	usage := &streamUsageData{}

	// Goroutine reads upstream lines via Scanner, sends through channel.
	// This decouples upstream reading from downstream writing,
	// allowing us to drain upstream even after client disconnects.
	events := make(chan scanEvent, 32)
	done := make(chan struct{})
	go func() {
		defer close(events)
		buf := make([]byte, 64*1024)
		scanner := newLineScanner(resp.Body, buf)
		for scanner.Scan() {
			select {
			case events <- scanEvent{line: scanner.Text()}:
			case <-done:
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case events <- scanEvent{err: err}:
			case <-done:
			}
		}
	}()
	defer close(done)

	clientDisconnected := false

	// Main loop: receive lines, accumulate SSE events, forward + extract usage
	for ev := range events {
		if ev.err != nil {
			if clientDisconnected {
				log.Printf("[gateway] upstream read error after client disconnect: %v", ev.err)
			}
			break
		}

		line := ev.line

		// Extract usage from data lines
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "[DONE]" {
				extractUsageFromData(data, usage)
			}
		}

		// Write to client only if still connected
		if !clientDisconnected {
			if _, werr := fmt.Fprintln(w, line); werr != nil {
				clientDisconnected = true
				log.Printf("[gateway] client disconnected during streaming, continuing to drain upstream for billing")
			} else {
				// Empty line = end of SSE event, flush
				if strings.TrimSpace(line) == "" {
					flusher.Flush()
				}
			}
		}
	}

	// Log usage
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

	costs := CalculateCost(ctx.Model, usage.inputTokens, usage.outputTokens, usage.cacheCreation, usage.cacheRead)
	usageLog := &model.UsageLog{
		RequestID:           ctx.RequestID,
		APIKeyID:            apiKeyID,
		AccountID:           ctx.Account.ID,
		GroupID:             groupID,
		Model:               ctx.Model,
		RequestedModel:      strPtr(ctx.RequestedModel),
		InputTokens:         usage.inputTokens,
		OutputTokens:        usage.outputTokens,
		CacheCreationTokens: usage.cacheCreation,
		CacheReadTokens:     usage.cacheRead,
		InputCost:           costs.InputCost,
		OutputCost:          costs.OutputCost,
		CacheCreationCost:   costs.CacheCreationCost,
		CacheReadCost:       costs.CacheReadCost,
		TotalCost:           costs.Total(),
		Stream:              true,
		DurationMs:          intPtr(int(duration)),
		StatusCode:          intPtr(statusCode),
		ErrorType:           strPtr(errType),
	}
	if logErr := g.usage.Log(usageLog); logErr != nil {
		log.Printf("[gateway] failed to log stream usage: %v", logErr)
	}
}

// SSELineConverter converts an upstream SSE line to zero or more output lines.
type SSELineConverter func(line string) []string

// StreamResponseWithConverter proxies a streaming response with per-line SSE conversion.
func (g *GatewayService) StreamResponseWithConverter(
	w http.ResponseWriter,
	resp *http.Response,
	ctx *RequestContext,
	convertFn SSELineConverter,
) {
	defer resp.Body.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(resp.StatusCode)

	usage := &streamUsageData{}

	events := make(chan scanEvent, 32)
	done := make(chan struct{})
	go func() {
		defer close(events)
		buf := make([]byte, 64*1024)
		scanner := newLineScanner(resp.Body, buf)
		for scanner.Scan() {
			select {
			case events <- scanEvent{line: scanner.Text()}:
			case <-done:
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case events <- scanEvent{err: err}:
			case <-done:
			}
		}
	}()
	defer close(done)

	clientDisconnected := false

	for ev := range events {
		if ev.err != nil {
			if clientDisconnected {
				log.Printf("[gateway] upstream read error after client disconnect: %v", ev.err)
			}
			break
		}

		line := ev.line

		// Extract usage from raw upstream data (before conversion)
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "[DONE]" {
				extractUsageFromData(data, usage)
			}
		}

		if !clientDisconnected {
			converted := convertFn(line)
			for _, cl := range converted {
				if _, werr := fmt.Fprintln(w, cl); werr != nil {
					clientDisconnected = true
					log.Printf("[gateway] client disconnected during streaming")
					break
				}
			}
			if !clientDisconnected && strings.TrimSpace(line) == "" {
				flusher.Flush()
			}
		}
	}

	// Log usage
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

	costs := CalculateCost(ctx.Model, usage.inputTokens, usage.outputTokens, usage.cacheCreation, usage.cacheRead)
	usageLog := &model.UsageLog{
		RequestID:           ctx.RequestID,
		APIKeyID:            apiKeyID,
		AccountID:           ctx.Account.ID,
		GroupID:             groupID,
		Model:               ctx.Model,
		RequestedModel:      strPtr(ctx.RequestedModel),
		InputTokens:         usage.inputTokens,
		OutputTokens:        usage.outputTokens,
		CacheCreationTokens: usage.cacheCreation,
		CacheReadTokens:     usage.cacheRead,
		InputCost:           costs.InputCost,
		OutputCost:          costs.OutputCost,
		CacheCreationCost:   costs.CacheCreationCost,
		CacheReadCost:       costs.CacheReadCost,
		TotalCost:           costs.Total(),
		Stream:              true,
		DurationMs:          intPtr(int(duration)),
		StatusCode:          intPtr(statusCode),
		ErrorType:           strPtr(errType),
	}
	if logErr := g.usage.Log(usageLog); logErr != nil {
		log.Printf("[gateway] failed to log stream usage: %v", logErr)
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// newLineScanner creates a bufio.Scanner with a 64KB buffer for SSE line reading.
func newLineScanner(r io.Reader, buf []byte) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(buf, 1024*1024) // max line 1MB
	return s
}

type streamUsageData struct {
	inputTokens   int
	outputTokens  int
	cacheCreation int
	cacheRead     int
}

// parseSSEInt extracts an int from any JSON numeric type (adapted from sub2api).
func parseSSEInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

// extractUsageFromData parses a single SSE data payload and extracts token usage.
// Supports Claude (message_start/message_delta), OpenAI Chat (usage),
// OpenAI Responses (response.usage), and Gemini (usageMetadata).
func extractUsageFromData(data string, u *streamUsageData) {
	var obj map[string]any
	if json.Unmarshal([]byte(data), &obj) != nil {
		return
	}

	// --- Claude: message_start / message_delta ---
	// message_start → obj.message.usage (input_tokens, cache_creation/read_input_tokens)
	// message_delta → obj.usage (output_tokens)
	if msg, ok := obj["message"].(map[string]any); ok {
		if usage, ok := msg["usage"].(map[string]any); ok {
			mergeClaudeUsage(usage, u)
		}
	}
	if usage, ok := obj["usage"].(map[string]any); ok {
		mergeClaudeUsage(usage, u)
	}

	// --- OpenAI Chat Completions: obj.usage with prompt_tokens/completion_tokens ---
	if usage, ok := obj["usage"].(map[string]any); ok {
		if v, ok := parseSSEInt(usage["prompt_tokens"]); ok && v > u.inputTokens {
			u.inputTokens = v
		}
		if v, ok := parseSSEInt(usage["completion_tokens"]); ok && v > u.outputTokens {
			u.outputTokens = v
		}
		if v, ok := usage["prompt_tokens_details"].(map[string]any); ok {
			if ct, ok := parseSSEInt(v["cached_tokens"]); ok && ct > u.cacheRead {
				u.cacheRead = ct
			}
		}
	}

	// --- OpenAI Responses API: obj.response.usage ---
	if resp, ok := obj["response"].(map[string]any); ok {
		if usage, ok := resp["usage"].(map[string]any); ok {
			if v, ok := parseSSEInt(usage["input_tokens"]); ok && v > u.inputTokens {
				u.inputTokens = v
			}
			if v, ok := parseSSEInt(usage["output_tokens"]); ok && v > u.outputTokens {
				u.outputTokens = v
			}
			if details, ok := usage["input_tokens_details"].(map[string]any); ok {
				if ct, ok := parseSSEInt(details["cached_tokens"]); ok && ct > u.cacheRead {
					u.cacheRead = ct
				}
			}
		}
	}

	// --- Gemini: obj.usageMetadata ---
	if meta, ok := obj["usageMetadata"].(map[string]any); ok {
		if v, ok := parseSSEInt(meta["promptTokenCount"]); ok && v > u.inputTokens {
			u.inputTokens = v
		}
		if v, ok := parseSSEInt(meta["candidatesTokenCount"]); ok && v > u.outputTokens {
			u.outputTokens = v
		}
		if v, ok := parseSSEInt(meta["cachedContentTokenCount"]); ok && v > u.cacheRead {
			u.cacheRead = v
		}
	}
}

func mergeClaudeUsage(usage map[string]any, u *streamUsageData) {
	if v, ok := parseSSEInt(usage["input_tokens"]); ok && v > u.inputTokens {
		u.inputTokens = v
	}
	if v, ok := parseSSEInt(usage["output_tokens"]); ok && v > u.outputTokens {
		u.outputTokens = v
	}
	if v, ok := parseSSEInt(usage["cache_creation_input_tokens"]); ok && v > u.cacheCreation {
		u.cacheCreation = v
	}
	if v, ok := parseSSEInt(usage["cache_read_input_tokens"]); ok && v > u.cacheRead {
		u.cacheRead = v
	}
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
