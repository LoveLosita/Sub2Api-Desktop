package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type PricingService struct {
	db *database.DB
}

func NewPricingService(db *database.DB) *PricingService {
	return &PricingService{db: db}
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
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}

	count := 0
	for modelName, raw := range data {
		if modelName == "sample_spec" {
			continue
		}

		var entry liteLLMEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}

		// Only keep claude/openai/gemini models with valid chat mode
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
			count++
		}
	}

	return count, nil
}

func (s *PricingService) Reset() error {
	s.db.Exec(`DELETE FROM model_pricing`)
	return s.Seed()
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
