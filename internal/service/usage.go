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

func (s *UsageService) List(limit, offset int, modelName, startDate, endDate string) (*model.UsageListResult, error) {
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
		SELECT ul.id, ul.request_id, ul.api_key_id, ul.account_id, CASE WHEN a.status = 'deleted' THEN '账号已删除' ELSE COALESCE(a.name, '') END,
			ul.group_id, ul.model, ul.requested_model,
			ul.input_tokens, ul.output_tokens, ul.cache_creation_tokens, ul.cache_read_tokens,
			ul.input_cost, ul.output_cost, ul.cache_creation_cost, ul.cache_read_cost, ul.total_cost,
			ul.stream, ul.duration_ms, ul.first_token_ms, ul.status_code, ul.error_type, ul.created_at
		FROM usage_logs ul LEFT JOIN accounts a ON ul.account_id = a.id
		`+where+` ORDER BY ul.created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.UsageLog
	for rows.Next() {
		var l model.UsageLog
		err := rows.Scan(&l.ID, &l.RequestID, &l.APIKeyID, &l.AccountID, &l.AccountName,
			&l.GroupID, &l.Model, &l.RequestedModel, &l.InputTokens, &l.OutputTokens,
			&l.CacheCreationTokens, &l.CacheReadTokens, &l.InputCost, &l.OutputCost,
			&l.CacheCreationCost, &l.CacheReadCost, &l.TotalCost, &l.Stream,
			&l.DurationMs, &l.FirstTokenMs, &l.StatusCode, &l.ErrorType, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return &model.UsageListResult{Logs: logs, Total: total}, nil
}

func (s *UsageService) ListModels() ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT model FROM usage_logs ORDER BY model")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []string
	for rows.Next() {
		var m string
		rows.Scan(&m)
		models = append(models, m)
	}
	return models, nil
}

func (s *UsageService) DashboardStats(since string) (*model.DashboardStats, error) {
	stats := &model.DashboardStats{}

	where := ""
	args := []any{}
	if since != "" {
		where = " WHERE created_at >= ?"
		args = append(args, since)
	}

	s.db.QueryRow("SELECT COUNT(*) FROM usage_logs"+where, args...).Scan(&stats.TotalRequests)
	s.db.QueryRow("SELECT COUNT(*) FROM usage_logs WHERE date(created_at, '+8 hours') = date('now', '+8 hours')").Scan(&stats.TodayRequests)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens + output_tokens), 0) FROM usage_logs"+where, args...).Scan(&stats.TotalTokens)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens + output_tokens), 0) FROM usage_logs WHERE date(created_at, '+8 hours') = date('now', '+8 hours')").Scan(&stats.TodayTokens)
	s.db.QueryRow("SELECT COALESCE(SUM(total_cost), 0) FROM usage_logs"+where, args...).Scan(&stats.TotalCost)
	s.db.QueryRow("SELECT COALESCE(SUM(total_cost), 0) FROM usage_logs WHERE date(created_at, '+8 hours') = date('now', '+8 hours')").Scan(&stats.TodayCost)

	rows, err := s.db.Query(`
		SELECT model, COUNT(*) as requests, SUM(input_tokens + output_tokens) as tokens, SUM(total_cost) as cost
		FROM usage_logs`+where+` GROUP BY model ORDER BY cost DESC LIMIT 10`, args...)
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
