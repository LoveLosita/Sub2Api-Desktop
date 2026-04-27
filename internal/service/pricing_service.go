package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultPricingURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

var pricingMirrors = []string{
	"https://ghfast.top/https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
	"https://ghproxy.cc/https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
}

type PricingService struct {
	db         *database.DB
	pricingURL string
}

func NewPricingService(db *database.DB, pricingURL string) *PricingService {
	return &PricingService{db: db, pricingURL: pricingURL}
}

func (s *PricingService) List() ([]model.ModelPricing, error) {
	rows, err := s.db.Query(`SELECT id, model, input_price, output_price, cache_creation_price, cache_read_price, image_price, updated_at FROM model_pricing ORDER BY model DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ModelPricing
	for rows.Next() {
		var p model.ModelPricing
		var updatedAt string
		if err := rows.Scan(&p.ID, &p.Model, &p.InputPrice, &p.OutputPrice, &p.CacheCreationPrice, &p.CacheReadPrice, &p.ImagePrice, &updatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}

	return list, nil
}

func (s *PricingService) Update(id int64, inputPrice, outputPrice, cacheCreation, cacheRead float64) error {
	_, err := s.db.Exec(`UPDATE model_pricing SET input_price=?, output_price=?, cache_creation_price=?, cache_read_price=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		inputPrice, outputPrice, cacheCreation, cacheRead, id)
	return err
}

func (s *PricingService) Seed() error {
	for m, p := range modelPricing {
		_, err := s.db.Exec(`INSERT OR IGNORE INTO model_pricing (model, input_price, output_price, cache_creation_price, cache_read_price) VALUES (?, ?, ?, ?, ?)`,
			m, p.InputPerM, p.OutputPerM, p.CacheCreationPerM, p.CacheReadPerM)
		if err != nil {
			return err
		}
	}
	return nil
}

// liteLLMEntry matches the LiteLLM pricing JSON format.
type liteLLMEntry struct {
	InputCostPerToken           *float64 `json:"input_cost_per_token"`
	OutputCostPerToken          *float64 `json:"output_cost_per_token"`
	CacheCreationInputTokenCost *float64 `json:"cache_creation_input_token_cost"`
	CacheReadInputTokenCost     *float64 `json:"cache_read_input_token_cost"`
	LitellmProvider             string   `json:"litellm_provider"`
	Mode                        string   `json:"mode"`
}

func (s *PricingService) FetchRemote() (int, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	urls := s.buildFetchURLs()
	var body []byte
	var lastErr error
	for _, u := range urls {
		resp, err := client.Get(u)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", u, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
			continue
		}
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("%s: read body: %w", u, err)
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return 0, fmt.Errorf("所有定价源均失败，最后一个错误: %w", lastErr)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}

	remoteModels := make(map[string]bool)
	count := 0
	for modelName, raw := range data {
		if modelName == "sample_spec" {
			continue
		}

		var entry liteLLMEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}

		provider := PlatformOfModel(modelName)
		if provider == "other" {
			continue
		}
		if entry.Mode != "chat" && entry.Mode != "" {
			continue
		}
		if entry.InputCostPerToken == nil || entry.OutputCostPerToken == nil {
			continue
		}

		inPerM := *entry.InputCostPerToken * 1_000_000
		outPerM := *entry.OutputCostPerToken * 1_000_000
		cacheCreatePerM := 0.0
		cacheReadPerM := 0.0
		if entry.CacheCreationInputTokenCost != nil {
			cacheCreatePerM = *entry.CacheCreationInputTokenCost * 1_000_000
		}
		if entry.CacheReadInputTokenCost != nil {
			cacheReadPerM = *entry.CacheReadInputTokenCost * 1_000_000
		}

		_, err := s.db.Exec(`INSERT INTO model_pricing (model, input_price, output_price, cache_creation_price, cache_read_price, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(model) DO UPDATE SET input_price=excluded.input_price, output_price=excluded.output_price,
				cache_creation_price=excluded.cache_creation_price, cache_read_price=excluded.cache_read_price,
				updated_at=CURRENT_TIMESTAMP`,
			modelName, inPerM, outPerM, cacheCreatePerM, cacheReadPerM)
		if err == nil {
			remoteModels[modelName] = true
			count++
		}
	}

	// Delete models not present in remote data
	if len(remoteModels) > 0 {
		placeholders := ""
		args := make([]any, 0, len(remoteModels))
		i := 0
		for m := range remoteModels {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, m)
			i++
		}
		s.db.Exec(`DELETE FROM model_pricing WHERE model NOT IN (`+placeholders+`)`, args...)
	}

	// Recalculate all usage_logs costs with updated prices
	s.recalculateUsageCosts()

	return count, nil
}

func (s *PricingService) Reset() error {
	s.db.Exec(`DELETE FROM model_pricing`)
	return s.Seed()
}

func (s *PricingService) buildFetchURLs() []string {
	urls := make([]string, 0, 1+len(pricingMirrors))
	if s.pricingURL != "" {
		urls = append(urls, s.pricingURL)
	}
	urls = append(urls, defaultPricingURL)
	urls = append(urls, pricingMirrors...)
	return urls
}

// recalculateUsageCosts recalculates cost fields in usage_logs based on current model_pricing.
func (s *PricingService) recalculateUsageCosts() {
	// Load all pricing into memory
	rows, err := s.db.Query(`SELECT model, input_price, output_price, cache_creation_price, cache_read_price FROM model_pricing`)
	if err != nil {
		return
	}
	prices := make(map[string][4]float64) // [input, output, cache_create, cache_read] per million tokens
	for rows.Next() {
		var model string
		var p [4]float64
		if rows.Scan(&model, &p[0], &p[1], &p[2], &p[3]) == nil {
			prices[model] = p
		}
	}
	rows.Close()

	// Iterate usage_logs and recalculate
	logs, err := s.db.Query(`SELECT id, model, input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens FROM usage_logs`)
	if err != nil {
		return
	}
	type logEntry struct {
		id                         int64
		model                      string
		inputTokens                int
		outputTokens               int
		cacheCreationTokens        int
		cacheReadTokens            int
	}
	var entries []logEntry
	for logs.Next() {
		var e logEntry
		if logs.Scan(&e.id, &e.model, &e.inputTokens, &e.outputTokens, &e.cacheCreationTokens, &e.cacheReadTokens) == nil {
			entries = append(entries, e)
		}
	}
	logs.Close()

	stmt, err := s.db.Prepare(`UPDATE usage_logs SET input_cost=?, output_cost=?, cache_creation_cost=?, cache_read_cost=?, total_cost=? WHERE id=?`)
	if err != nil {
		return
	}
	defer stmt.Close()

	for _, e := range entries {
		p, ok := prices[e.model]
		if !ok {
			continue
		}
		regularInput := e.inputTokens - e.cacheCreationTokens - e.cacheReadTokens
		if regularInput < 0 {
			regularInput = 0
		}
		inCost := float64(regularInput) * p[0] / 1_000_000
		outCost := float64(e.outputTokens) * p[1] / 1_000_000
		cacheCreateCost := float64(e.cacheCreationTokens) * p[2] / 1_000_000
		cacheReadCost := float64(e.cacheReadTokens) * p[3] / 1_000_000
		stmt.Exec(inCost, outCost, cacheCreateCost, cacheReadCost, inCost+outCost+cacheCreateCost+cacheReadCost, e.id)
	}
}

func (s *PricingService) GetPrice(modelName string) *modelPrice {
	var p model.ModelPricing
	err := s.db.QueryRow(`SELECT input_price, output_price, cache_creation_price, cache_read_price FROM model_pricing WHERE model=?`, modelName).Scan(
		&p.InputPrice, &p.OutputPrice, &p.CacheCreationPrice, &p.CacheReadPrice)
	if err != nil {
		return nil
	}
	return &modelPrice{
		InputPerM:         p.InputPrice,
		OutputPerM:        p.OutputPrice,
		CacheCreationPerM: p.CacheCreationPrice,
		CacheReadPerM:     p.CacheReadPrice,
	}
}

func PlatformOfModel(modelName string) string {
	if strings.HasPrefix(modelName, "claude-") || strings.HasPrefix(modelName, "claude_") {
		return "claude"
	}
	if strings.HasPrefix(modelName, "gpt-") || strings.HasPrefix(modelName, "o1") || strings.HasPrefix(modelName, "o3") || strings.HasPrefix(modelName, "o4") || strings.HasPrefix(modelName, "chatgpt") {
		return "openai"
	}
	if strings.HasPrefix(modelName, "gemini-") || strings.HasPrefix(modelName, "gemini_") {
		return "gemini"
	}
	return "other"
}
