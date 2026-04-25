package service

import "sync"

type CostBreakdown struct {
	InputCost         float64
	OutputCost        float64
	CacheCreationCost float64
	CacheReadCost     float64
}

func (c CostBreakdown) Total() float64 {
	return c.InputCost + c.OutputCost + c.CacheCreationCost + c.CacheReadCost
}

type modelPrice struct {
	InputPerM         float64
	OutputPerM        float64
	CacheCreationPerM float64
	CacheReadPerM     float64
}

var (
	pricingMu    sync.RWMutex
	modelPricing = map[string]modelPrice{
		// Claude
		"claude-opus-4-20250514":          {15, 75, 18.75, 1.875},
		"claude-opus-4-7-20250219":        {15, 75, 18.75, 1.875},
		"claude-sonnet-4-20250514":        {3, 15, 3.75, 0.375},
		"claude-sonnet-4-6-20250514":      {3, 15, 3.75, 0.375},
		"claude-haiku-4-5-20251001":       {0.80, 4, 1, 0.08},
		"claude-3-5-sonnet-20241022":      {3, 15, 3.75, 0.30},
		"claude-3-5-haiku-20241022":       {0.80, 4, 1, 0.08},
		"claude-3-opus-20240229":          {15, 75, 18.75, 1.50},
		"claude-3-haiku-20240307":         {0.25, 1.25, 0.30, 0.03},
		// OpenAI
		"gpt-4o":                          {2.50, 10, 0, 1.25},
		"gpt-4o-mini":                     {0.15, 0.60, 0, 0.075},
		"gpt-4o-2024-11-20":              {2.50, 10, 0, 1.25},
		"gpt-4-turbo":                     {10, 30, 0, 5},
		"gpt-4":                           {30, 60, 0, 15},
		"gpt-3.5-turbo":                   {0.50, 1.50, 0, 0.25},
		"o1":                              {15, 60, 0, 7.50},
		"o1-mini":                         {3, 12, 0, 1.50},
		"o3-mini":                         {1.10, 4.40, 0, 0.55},
		"o4-mini":                         {1.10, 4.40, 0, 0.55},
		// Gemini
		"gemini-2.5-pro-preview-05-06":    {1.25, 10, 0, 0.315},
		"gemini-2.5-flash-preview-05-20":  {0.15, 0.60, 0, 0.0375},
		"gemini-2.0-flash":                {0.10, 0.40, 0, 0.025},
		"gemini-1.5-pro":                  {1.25, 5, 0, 0.315},
		"gemini-1.5-flash":                {0.075, 0.30, 0, 0.01875},
	}
)

func CalculateCost(modelName string, inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int) CostBreakdown {
	pricingMu.RLock()
	price, ok := modelPricing[modelName]
	pricingMu.RUnlock()

	if !ok {
		// Default: assume cheap model pricing
		price = modelPrice{InputPerM: 3, OutputPerM: 15}
	}

	return CostBreakdown{
		InputCost:         float64(inputTokens) * price.InputPerM / 1_000_000,
		OutputCost:        float64(outputTokens) * price.OutputPerM / 1_000_000,
		CacheCreationCost: float64(cacheCreationTokens) * price.CacheCreationPerM / 1_000_000,
		CacheReadCost:     float64(cacheReadTokens) * price.CacheReadPerM / 1_000_000,
	}
}
