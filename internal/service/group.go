package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"encoding/json"
)

type GroupService struct {
	db *database.DB
}

func NewGroupService(db *database.DB) *GroupService {
	return &GroupService{db: db}
}

func (s *GroupService) List() ([]model.Group, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, platform, rate_multiplier, is_exclusive,
			status, model_routing, model_routing_enabled, created_at, updated_at
		FROM groups ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []model.Group
	for rows.Next() {
		var g model.Group
		var isExcl, mrEnabled int
		var routing string
		err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Platform,
			&g.RateMultiplier, &isExcl, &g.Status, &routing, &mrEnabled,
			&g.CreatedAt, &g.UpdatedAt)
		if err != nil {
			return nil, err
		}
		g.IsExclusive = isExcl == 1
		g.ModelRoutingEnabled = mrEnabled == 1
		json.Unmarshal([]byte(routing), &g.ModelRouting)
		g.AccountIDs, _ = s.getAccountIDs(g.ID)
		groups = append(groups, g)
	}
	return groups, nil
}

func (s *GroupService) GetByID(id int64) (*model.Group, error) {
	var g model.Group
	var isExcl, mrEnabled int
	var routing string
	err := s.db.QueryRow(`
		SELECT id, name, description, platform, rate_multiplier, is_exclusive,
			status, model_routing, model_routing_enabled, created_at, updated_at
		FROM groups WHERE id = ?`, id).Scan(
		&g.ID, &g.Name, &g.Description, &g.Platform,
		&g.RateMultiplier, &isExcl, &g.Status, &routing, &mrEnabled,
		&g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.IsExclusive = isExcl == 1
	g.ModelRoutingEnabled = mrEnabled == 1
	json.Unmarshal([]byte(routing), &g.ModelRouting)
	g.AccountIDs, _ = s.getAccountIDs(g.ID)
	return &g, nil
}

func (s *GroupService) Create(g *model.Group) error {
	routing, _ := json.Marshal(g.ModelRouting)
	isExcl := 0
	if g.IsExclusive {
		isExcl = 1
	}
	mrEnabled := 0
	if g.ModelRoutingEnabled {
		mrEnabled = 1
	}
	result, err := s.db.Exec(`
		INSERT INTO groups (name, description, platform, rate_multiplier, is_exclusive,
			status, model_routing, model_routing_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		g.Name, g.Description, g.Platform, g.RateMultiplier, isExcl,
		g.Status, string(routing), mrEnabled)
	if err != nil {
		return err
	}
	g.ID, _ = result.LastInsertId()
	return s.setAccounts(g.ID, g.AccountIDs)
}

func (s *GroupService) Update(g *model.Group) error {
	routing, _ := json.Marshal(g.ModelRouting)
	isExcl := 0
	if g.IsExclusive {
		isExcl = 1
	}
	mrEnabled := 0
	if g.ModelRoutingEnabled {
		mrEnabled = 1
	}
	_, err := s.db.Exec(`
		UPDATE groups SET name=?, description=?, platform=?, rate_multiplier=?,
			is_exclusive=?, status=?, model_routing=?, model_routing_enabled=?,
			updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		g.Name, g.Description, g.Platform, g.RateMultiplier, isExcl,
		g.Status, string(routing), mrEnabled, g.ID)
	if err != nil {
		return err
	}
	return s.setAccounts(g.ID, g.AccountIDs)
}

func (s *GroupService) Delete(id int64) error {
	s.db.Exec("DELETE FROM account_groups WHERE group_id = ?", id)
	_, err := s.db.Exec("DELETE FROM groups WHERE id = ?", id)
	return err
}

func (s *GroupService) getAccountIDs(groupID int64) ([]int64, error) {
	rows, err := s.db.Query("SELECT account_id FROM account_groups WHERE group_id = ?", groupID)
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

func (s *GroupService) setAccounts(groupID int64, accountIDs []int64) error {
	s.db.Exec("DELETE FROM account_groups WHERE group_id = ?", groupID)
	for _, aid := range accountIDs {
		s.db.Exec("INSERT INTO account_groups (account_id, group_id) VALUES (?, ?)", aid, groupID)
	}
	return nil
}
