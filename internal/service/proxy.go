package service

import (
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
)

type ProxyService struct {
	db *database.DB
}

func NewProxyService(db *database.DB) *ProxyService {
	return &ProxyService{db: db}
}

func (s *ProxyService) List() ([]model.Proxy, error) {
	rows, err := s.db.Query(`
		SELECT id, name, protocol, host, port, username, password, status, created_at, updated_at
		FROM proxies ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var proxies []model.Proxy
	for rows.Next() {
		var p model.Proxy
		err := rows.Scan(&p.ID, &p.Name, &p.Protocol, &p.Host, &p.Port,
			&p.Username, &p.Password, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, p)
	}
	return proxies, nil
}

func (s *ProxyService) GetByID(id int64) (*model.Proxy, error) {
	var p model.Proxy
	err := s.db.QueryRow(`
		SELECT id, name, protocol, host, port, username, password, status, created_at, updated_at
		FROM proxies WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.Protocol, &p.Host, &p.Port,
		&p.Username, &p.Password, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *ProxyService) Create(p *model.Proxy) error {
	result, err := s.db.Exec(`
		INSERT INTO proxies (name, protocol, host, port, username, password, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Protocol, p.Host, p.Port, p.Username, p.Password, p.Status)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

func (s *ProxyService) Update(p *model.Proxy) error {
	_, err := s.db.Exec(`
		UPDATE proxies SET name=?, protocol=?, host=?, port=?, username=?, password=?,
			status=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		p.Name, p.Protocol, p.Host, p.Port, p.Username, p.Password, p.Status, p.ID)
	return err
}

func (s *ProxyService) Delete(id int64) error {
	s.db.Exec("UPDATE accounts SET proxy_id = NULL WHERE proxy_id = ?", id)
	_, err := s.db.Exec("DELETE FROM proxies WHERE id = ?", id)
	return err
}

func (s *ProxyService) Test(id int64) error {
	// TODO: implement proxy connectivity test
	return nil
}
