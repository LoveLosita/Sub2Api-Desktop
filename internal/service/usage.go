package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
)

type UsageService struct {
	db *database.DB
}

func NewUsageService(db *database.DB) *UsageService {
	return &UsageService{db: db}
}

func (s *UsageService) Log(log *model.UsageLog) error {
	_, err := s.db.Exec(`
		INSERT INTO usage_logs (request_id, api_key_id, account_id, group_id, model,
			requested_model, input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			input_cost, output_cost, cache_creation_cost, cache_read_cost, total_cost,
			stream, duration_ms, first_token_ms, status_code, error_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.RequestID, log.APIKeyID, log.AccountID, log.GroupID, log.Model,
		log.RequestedModel, log.InputTokens, log.OutputTokens,
		log.CacheCreationTokens, log.CacheReadTokens,
		log.InputCost, log.OutputCost, log.CacheCreationCost, log.CacheReadCost,
		log.TotalCost, log.Stream, log.DurationMs, log.FirstTokenMs,
		log.StatusCode, log.ErrorType)
	return err
}

func (s *UsageService) List(limit, offset int, modelName, startDate, endDate string) ([]model.UsageLog, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if modelName != "" {
		where += " AND model = ?"
		args = append(args, modelName)
	}
	if startDate != "" {
		where += " AND created_at >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		where += " AND created_at < ?"
		args = append(args, endDate)
	}

	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM usage_logs "+where, args...).Scan(&total)

	args = append(args, limit, offset)
	rows, err := s.db.Query(`
		SELECT id, request_id, api_key_id, account_id, group_id, model, requested_model,
			input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			input_cost, output_cost, cache_creation_cost, cache_read_cost, total_cost,
			stream, duration_ms, first_token_ms, status_code, error_type, created_at
		FROM usage_logs `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.UsageLog
	for rows.Next() {
		var l model.UsageLog
		err := rows.Scan(&l.ID, &l.RequestID, &l.APIKeyID, &l.AccountID, &l.GroupID,
			&l.Model, &l.RequestedModel, &l.InputTokens, &l.OutputTokens,
			&l.CacheCreationTokens, &l.CacheReadTokens, &l.InputCost, &l.OutputCost,
			&l.CacheCreationCost, &l.CacheReadCost, &l.TotalCost, &l.Stream,
			&l.DurationMs, &l.FirstTokenMs, &l.StatusCode, &l.ErrorType, &l.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (s *UsageService) DashboardStats() (*model.DashboardStats, error) {
	stats := &model.DashboardStats{}

	s.db.QueryRow("SELECT COUNT(*) FROM usage_logs").Scan(&stats.TotalRequests)
	s.db.QueryRow("SELECT COUNT(*) FROM usage_logs WHERE date(created_at) = date('now')").Scan(&stats.TodayRequests)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) FROM usage_logs").Scan(&stats.TotalTokens)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) FROM usage_logs WHERE date(created_at) = date('now')").Scan(&stats.TodayTokens)
	s.db.QueryRow("SELECT COALESCE(SUM(total_cost), 0) FROM usage_logs").Scan(&stats.TotalCost)
	s.db.QueryRow("SELECT COALESCE(SUM(total_cost), 0) FROM usage_logs WHERE date(created_at) = date('now')").Scan(&stats.TodayCost)

	rows, err := s.db.Query(`
		SELECT model, COUNT(*) as requests, SUM(input_tokens + output_tokens) as tokens, SUM(total_cost) as cost
		FROM usage_logs GROUP BY model ORDER BY cost DESC LIMIT 10`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e model.ModelCostEntry
			rows.Scan(&e.Model, &e.Requests, &e.Tokens, &e.Cost)
			stats.ByModel = append(stats.ByModel, e)
		}
	}
	return stats, nil
}
