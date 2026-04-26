package model

import "time"

type Account struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"`
	Platform        string            `json:"platform"`
	Type            string            `json:"type"`
	Credentials     map[string]any    `json:"credentials"`
	Extra           map[string]any    `json:"extra"`
	ProxyID         *int64            `json:"proxy_id"`
	BaseURL         *string           `json:"base_url"`
	Concurrency     int               `json:"concurrency"`
	Priority        int               `json:"priority"`
	Status          string            `json:"status"`
	ErrorMessage    *string           `json:"error_message"`
	Schedulable     bool              `json:"schedulable"`
	RateLimitedAt   *time.Time        `json:"rate_limited_at"`
	RateLimitResetAt *time.Time       `json:"rate_limit_reset_at"`
	OverloadUntil   *time.Time        `json:"overload_until"`
	LastUsedAt      *time.Time        `json:"last_used_at"`
	GroupIDs        []int64           `json:"group_ids"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type Group struct {
	ID                  int64              `json:"id"`
	Name                string             `json:"name"`
	Description         *string            `json:"description"`
	Platform            string             `json:"platform"`
	RateMultiplier      float64            `json:"rate_multiplier"`
	IsExclusive         bool               `json:"is_exclusive"`
	Status              string             `json:"status"`
	ModelRouting        map[string][]int64 `json:"model_routing"`
	ModelRoutingEnabled bool               `json:"model_routing_enabled"`
	AccountIDs          []int64            `json:"account_ids"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

type APIKey struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	GroupID     *int64   `json:"group_id"`
	Status      string   `json:"status"`
	IPWhitelist []string `json:"ip_whitelist"`
	IPBlacklist []string `json:"ip_blacklist"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Group       *Group   `json:"group,omitempty"`
}

type Proxy struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Protocol  string    `json:"protocol"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  *string   `json:"username"`
	Password  *string   `json:"password"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UsageLog struct {
	ID                 int64     `json:"id"`
	RequestID          string    `json:"request_id"`
	APIKeyID           *int64    `json:"api_key_id"`
	AccountID          int64     `json:"account_id"`
	AccountName        string    `json:"account_name"`
	GroupID            *int64    `json:"group_id"`
	Model              string    `json:"model"`
	RequestedModel     *string   `json:"requested_model"`
	InputTokens        int       `json:"input_tokens"`
	OutputTokens       int       `json:"output_tokens"`
	CacheCreationTokens int      `json:"cache_creation_tokens"`
	CacheReadTokens    int       `json:"cache_read_tokens"`
	InputCost          float64   `json:"input_cost"`
	OutputCost         float64   `json:"output_cost"`
	CacheCreationCost  float64   `json:"cache_creation_cost"`
	CacheReadCost      float64   `json:"cache_read_cost"`
	TotalCost          float64   `json:"total_cost"`
	Stream             bool      `json:"stream"`
	DurationMs         *int      `json:"duration_ms"`
	FirstTokenMs       *int      `json:"first_token_ms"`
	StatusCode         *int      `json:"status_code"`
	ErrorType          *string   `json:"error_type"`
	CreatedAt          time.Time `json:"created_at"`
}

type ModelPricing struct {
	ID                int64   `json:"id"`
	Model             string  `json:"model"`
	InputPrice        float64 `json:"input_price"`
	OutputPrice       float64 `json:"output_price"`
	CacheCreationPrice float64 `json:"cache_creation_price"`
	CacheReadPrice    float64 `json:"cache_read_price"`
	ImagePrice        float64 `json:"image_price"`
}

type DashboardStats struct {
	TotalAccounts    int              `json:"total_accounts"`
	ActiveAccounts   int              `json:"active_accounts"`
	ErrorAccounts    int              `json:"error_accounts"`
	RateLimitAccounts int             `json:"rate_limit_accounts"`
	TotalRequests    int64            `json:"total_requests"`
	TodayRequests    int64            `json:"today_requests"`
	TotalTokens      int64            `json:"total_tokens"`
	TodayTokens      int64            `json:"today_tokens"`
	TotalCost        float64          `json:"total_cost"`
	TodayCost        float64          `json:"today_cost"`
	ByModel          []ModelCostEntry `json:"by_model"`
}

type ModelCostEntry struct {
	Model     string  `json:"model"`
	Requests  int64   `json:"requests"`
	Tokens    int64   `json:"tokens"`
	Cost      float64 `json:"cost"`
}

type UsageListResult struct {
	Logs  []UsageLog `json:"logs"`
	Total int        `json:"total"`
}
