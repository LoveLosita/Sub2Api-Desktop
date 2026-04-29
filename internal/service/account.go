package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"encoding/json"
	"time"
)

type AccountService struct {
	db *database.DB
}

func NewAccountService(db *database.DB) *AccountService {
	return &AccountService{db: db}
}

func (s *AccountService) List() ([]model.Account, error) {
	rows, err := s.db.Query(`
		SELECT id, name, platform, type, credentials, extra, proxy_id,
			base_url, concurrency, priority, multiplier, status, error_message, schedulable,
			rate_limited_at, rate_limit_reset_at, overload_until, last_used_at,
			created_at, updated_at
		FROM accounts WHERE status != 'deleted' ORDER BY priority ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var a model.Account
		var creds, extra string
		var schedulable int
		err := rows.Scan(&a.ID, &a.Name, &a.Platform, &a.Type, &creds, &extra,
			&a.ProxyID, &a.BaseURL, &a.Concurrency, &a.Priority, &a.Multiplier, &a.Status, &a.ErrorMessage,
			&schedulable, &a.RateLimitedAt, &a.RateLimitResetAt, &a.OverloadUntil,
			&a.LastUsedAt, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		a.Schedulable = schedulable == 1
		json.Unmarshal([]byte(creds), &a.Credentials)
		json.Unmarshal([]byte(extra), &a.Extra)
		accounts = append(accounts, a)
	}

	// Load group associations
	for i := range accounts {
		accounts[i].GroupIDs, _ = s.getGroupIDs(accounts[i].ID)
	}
	return accounts, nil
}

func (s *AccountService) GetByID(id int64) (*model.Account, error) {
	var a model.Account
	var creds, extra string
	var schedulable int
	err := s.db.QueryRow(`
		SELECT id, name, platform, type, credentials, extra, proxy_id,
			base_url, concurrency, priority, multiplier, status, error_message, schedulable,
			rate_limited_at, rate_limit_reset_at, overload_until, last_used_at,
			created_at, updated_at
		FROM accounts WHERE id = ?`, id).Scan(
		&a.ID, &a.Name, &a.Platform, &a.Type, &creds, &extra,
		&a.ProxyID, &a.BaseURL, &a.Concurrency, &a.Priority, &a.Multiplier, &a.Status, &a.ErrorMessage,
		&schedulable, &a.RateLimitedAt, &a.RateLimitResetAt, &a.OverloadUntil,
		&a.LastUsedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.Schedulable = schedulable == 1
	json.Unmarshal([]byte(creds), &a.Credentials)
	json.Unmarshal([]byte(extra), &a.Extra)
	a.GroupIDs, _ = s.getGroupIDs(a.ID)
	return &a, nil
}

func (s *AccountService) Create(a *model.Account) error {
	creds, _ := json.Marshal(a.Credentials)
	extra, _ := json.Marshal(a.Extra)
	schedulable := 0
	if a.Schedulable {
		schedulable = 1
	}
	result, err := s.db.Exec(`
		INSERT INTO accounts (name, platform, type, credentials, extra, proxy_id,
			base_url, concurrency, priority, multiplier, status, schedulable)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Name, a.Platform, a.Type, string(creds), string(extra),
		a.ProxyID, a.BaseURL, a.Concurrency, a.Priority, a.Multiplier, a.Status, schedulable)
	if err != nil {
		return err
	}
	a.ID, _ = result.LastInsertId()
	return s.setGroups(a.ID, a.GroupIDs)
}

func (s *AccountService) Update(a *model.Account) error {
	creds, _ := json.Marshal(a.Credentials)
	extra, _ := json.Marshal(a.Extra)
	schedulable := 0
	if a.Schedulable {
		schedulable = 1
	}
	_, err := s.db.Exec(`
		UPDATE accounts SET name=?, platform=?, type=?, credentials=?, extra=?,
			proxy_id=?, base_url=?, concurrency=?, priority=?, multiplier=?, status=?, error_message=?,
			schedulable=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		a.Name, a.Platform, a.Type, string(creds), string(extra),
		a.ProxyID, a.BaseURL, a.Concurrency, a.Priority, a.Multiplier, a.Status, a.ErrorMessage,
		schedulable, a.ID)
	if err != nil {
		return err
	}
	return s.setGroups(a.ID, a.GroupIDs)
}

func (s *AccountService) Delete(id int64) error {
	s.db.Exec("DELETE FROM account_groups WHERE account_id = ?", id)
	_, err := s.db.Exec("UPDATE accounts SET status='deleted', updated_at=CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (s *AccountService) UpdateStatus(id int64, status string, errMsg *string) error {
	_, err := s.db.Exec("UPDATE accounts SET status=?, error_message=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		status, errMsg, id)
	return err
}

func (s *AccountService) UpdateScheduling(id int64, schedulable bool, rateLimitedAt, rateLimitResetAt, overloadUntil *time.Time) error {
	sc := 0
	if schedulable {
		sc = 1
	}
	_, err := s.db.Exec(`
		UPDATE accounts SET schedulable=?, rate_limited_at=?, rate_limit_reset_at=?,
			overload_until=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		sc, rateLimitedAt, rateLimitResetAt, overloadUntil, id)
	return err
}

func (s *AccountService) MarkUsed(id int64) error {
	_, err := s.db.Exec("UPDATE accounts SET last_used_at=CURRENT_TIMESTAMP WHERE id=?", id)
	return err
}

func (s *AccountService) GetSchedulableForGroup(groupID int64) ([]model.Account, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.name, a.platform, a.type, a.credentials, a.extra, a.proxy_id,
			a.base_url, a.concurrency, a.priority, a.multiplier, a.status, a.error_message, a.schedulable,
			a.rate_limited_at, a.rate_limit_reset_at, a.overload_until, a.last_used_at,
			a.created_at, a.updated_at
		FROM accounts a
		INNER JOIN account_groups ag ON a.id = ag.account_id
		WHERE ag.group_id = ? AND a.schedulable = 1 AND a.status = 'active'
		ORDER BY a.priority ASC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	now := time.Now()
	for rows.Next() {
		var a model.Account
		var creds, extra string
		var schedulable int
		err := rows.Scan(&a.ID, &a.Name, &a.Platform, &a.Type, &creds, &extra,
			&a.ProxyID, &a.BaseURL, &a.Concurrency, &a.Priority, &a.Multiplier, &a.Status, &a.ErrorMessage,
			&schedulable, &a.RateLimitedAt, &a.RateLimitResetAt, &a.OverloadUntil,
			&a.LastUsedAt, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		a.Schedulable = schedulable == 1
		json.Unmarshal([]byte(creds), &a.Credentials)
		json.Unmarshal([]byte(extra), &a.Extra)

		// Skip rate-limited or overloaded accounts
		if a.RateLimitResetAt != nil && now.Before(*a.RateLimitResetAt) {
			continue
		}
		if a.OverloadUntil != nil && now.Before(*a.OverloadUntil) {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (s *AccountService) getGroupIDs(accountID int64) ([]int64, error) {
	rows, err := s.db.Query("SELECT group_id FROM account_groups WHERE account_id = ?", accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *AccountService) setGroups(accountID int64, groupIDs []int64) error {
	s.db.Exec("DELETE FROM account_groups WHERE account_id = ?", accountID)
	for _, gid := range groupIDs {
		s.db.Exec("INSERT INTO account_groups (account_id, group_id) VALUES (?, ?)", accountID, gid)
	}
	return nil
}

// Stats returns account counts by status.
func (s *AccountService) Stats() (total, active, errored, rateLimited int, err error) {
	err = s.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE status != 'deleted'").Scan(&total)
	if err != nil {
		return
	}
	s.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE status = 'active'").Scan(&active)
	s.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE status = 'error'").Scan(&errored)
	var rlCount int
	s.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE rate_limit_reset_at > CURRENT_TIMESTAMP").Scan(&rlCount)
	rateLimited = rlCount
	return
}
