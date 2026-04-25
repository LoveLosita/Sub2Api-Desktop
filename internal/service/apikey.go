package service

import (
	"crypto/rand"
	"database/sql"
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"encoding/json"
	"fmt"
)

type APIKeyService struct {
	db *database.DB
}

func NewAPIKeyService(db *database.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

func (s *APIKeyService) List() ([]model.APIKey, error) {
	rows, err := s.db.Query(`
		SELECT k.id, k.name, k.key, k.group_id, k.status, k.ip_whitelist, k.ip_blacklist,
			k.last_used_at, k.created_at, k.updated_at
		FROM api_keys k ORDER BY k.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		var wl, bl string
		err := rows.Scan(&k.ID, &k.Name, &k.Key, &k.GroupID, &k.Status,
			&wl, &bl, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(wl), &k.IPWhitelist)
		json.Unmarshal([]byte(bl), &k.IPBlacklist)
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *APIKeyService) GetByID(id int64) (*model.APIKey, error) {
	var k model.APIKey
	var wl, bl string
	err := s.db.QueryRow(`
		SELECT id, name, key, group_id, status, ip_whitelist, ip_blacklist,
			last_used_at, created_at, updated_at
		FROM api_keys WHERE id = ?`, id).Scan(
		&k.ID, &k.Name, &k.Key, &k.GroupID, &k.Status,
		&wl, &bl, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(wl), &k.IPWhitelist)
	json.Unmarshal([]byte(bl), &k.IPBlacklist)
	return &k, nil
}

func (s *APIKeyService) GetByKey(key string) (*model.APIKey, error) {
	var k model.APIKey
	var wl, bl string
	err := s.db.QueryRow(`
		SELECT id, name, key, group_id, status, ip_whitelist, ip_blacklist,
			last_used_at, created_at, updated_at
		FROM api_keys WHERE key = ? AND status = 'active'`, key).Scan(
		&k.ID, &k.Name, &k.Key, &k.GroupID, &k.Status,
		&wl, &bl, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(wl), &k.IPWhitelist)
	json.Unmarshal([]byte(bl), &k.IPBlacklist)
	return &k, nil
}

func (s *APIKeyService) Create(k *model.APIKey) error {
	if k.Key == "" {
		key, err := generateKey()
		if err != nil {
			return err
		}
		k.Key = key
	}
	wl, _ := json.Marshal(k.IPWhitelist)
	bl, _ := json.Marshal(k.IPBlacklist)
	result, err := s.db.Exec(`
		INSERT INTO api_keys (name, key, group_id, status, ip_whitelist, ip_blacklist)
		VALUES (?, ?, ?, ?, ?, ?)`,
		k.Name, k.Key, k.GroupID, k.Status, string(wl), string(bl))
	if err != nil {
		return err
	}
	k.ID, _ = result.LastInsertId()
	return nil
}

func (s *APIKeyService) Delete(id int64) error {
	_, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	return err
}

func (s *APIKeyService) MarkUsed(id int64) error {
	_, err := s.db.Exec("UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func generateKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("sk-%x", b), nil
}

// ValidateKey validates an API key and returns the associated group.
func (s *APIKeyService) ValidateKey(key string) (*model.APIKey, *model.Group, error) {
	k, err := s.GetByKey(key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("invalid API key")
		}
		return nil, nil, err
	}
	if k.GroupID == nil {
		return k, nil, nil
	}

	// Load group
	gs := NewGroupService(s.db)
	g, err := gs.GetByID(*k.GroupID)
	if err != nil {
		return k, nil, nil
	}
	k.Group = g
	s.MarkUsed(k.ID)
	return k, g, nil
}
