package main

import (
	"context"
	"desktop-proxy/internal/config"
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"desktop-proxy/internal/service"
	"log"
	"os"
	"path/filepath"
)

// App is the main application struct bound to the Wails frontend.
type App struct {
	ctx    context.Context
	db     *database.DB
	cfg    *config.Config
	baseDir string

	accounts *service.AccountService
	groups   *service.GroupService
	proxies  *service.ProxyService
	apiKeys  *service.APIKeyService
	usage    *service.UsageService
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	a.baseDir = filepath.Dir(exePath)

	// Load config
	a.cfg, err = config.Load(a.baseDir)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Init database
	dbPath := filepath.Join(a.baseDir, a.cfg.Database.Path)
	a.db, err = database.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Init services
	a.accounts = service.NewAccountService(a.db)
	a.groups = service.NewGroupService(a.db)
	a.proxies = service.NewProxyService(a.db)
	a.apiKeys = service.NewAPIKeyService(a.db)
	a.usage = service.NewUsageService(a.db)

	log.Println("Desktop Proxy started. DB:", dbPath)
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// ===== General =====

func (a *App) GetAppVersion() string {
	return "0.1.0"
}

func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// ===== Dashboard =====

func (a *App) GetDashboardStats() (*model.DashboardStats, error) {
	stats, err := a.usage.DashboardStats()
	if err != nil {
		return nil, err
	}
	total, active, errored, rateLimited := 0, 0, 0, 0
	total, active, errored, rateLimited, _ = a.accounts.Stats()
	stats.TotalAccounts = total
	stats.ActiveAccounts = active
	stats.ErrorAccounts = errored
	stats.RateLimitAccounts = rateLimited
	return stats, nil
}

// ===== Accounts =====

func (a *App) ListAccounts() ([]model.Account, error) {
	return a.accounts.List()
}

func (a *App) GetAccount(id int64) (*model.Account, error) {
	return a.accounts.GetByID(id)
}

func (a *App) CreateAccount(account model.Account) error {
	return a.accounts.Create(&account)
}

func (a *App) UpdateAccount(account model.Account) error {
	return a.accounts.Update(&account)
}

func (a *App) DeleteAccount(id int64) error {
	return a.accounts.Delete(id)
}

// ===== Groups =====

func (a *App) ListGroups() ([]model.Group, error) {
	return a.groups.List()
}

func (a *App) GetGroup(id int64) (*model.Group, error) {
	return a.groups.GetByID(id)
}

func (a *App) CreateGroup(group model.Group) error {
	return a.groups.Create(&group)
}

func (a *App) UpdateGroup(group model.Group) error {
	return a.groups.Update(&group)
}

func (a *App) DeleteGroup(id int64) error {
	return a.groups.Delete(id)
}

// ===== Proxies =====

func (a *App) ListProxies() ([]model.Proxy, error) {
	return a.proxies.List()
}

func (a *App) GetProxy(id int64) (*model.Proxy, error) {
	return a.proxies.GetByID(id)
}

func (a *App) CreateProxy(proxy model.Proxy) error {
	return a.proxies.Create(&proxy)
}

func (a *App) UpdateProxy(proxy model.Proxy) error {
	return a.proxies.Update(&proxy)
}

func (a *App) DeleteProxy(id int64) error {
	return a.proxies.Delete(id)
}

// ===== API Keys =====

func (a *App) ListAPIKeys() ([]model.APIKey, error) {
	return a.apiKeys.List()
}

func (a *App) CreateAPIKey(key model.APIKey) error {
	return a.apiKeys.Create(&key)
}

func (a *App) DeleteAPIKey(id int64) error {
	return a.apiKeys.Delete(id)
}

// ===== Usage =====

func (a *App) ListUsage(limit, offset int, modelName, startDate, endDate string) ([]model.UsageLog, int, error) {
	return a.usage.List(limit, offset, modelName, startDate, endDate)
}
