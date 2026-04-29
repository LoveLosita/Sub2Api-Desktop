package main

import (
	"context"
	"desktop-proxy/internal/config"
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/model"
	"desktop-proxy/internal/server"
	"desktop-proxy/internal/service"
	"fmt"
	"log"
	"os"
	"path/filepath"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct bound to the Wails frontend.
type App struct {
	ctx     context.Context
	db      *database.DB
	cfg     *config.Config
	baseDir string
	proxy   *server.Server

	accounts   *service.AccountService
	groups     *service.GroupService
	proxies    *service.ProxyService
	apiKeys    *service.APIKeyService
	usage      *service.UsageService
	pricing    *service.PricingService
	healthCheck *service.HealthCheckService
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		return
	}
	a.baseDir = filepath.Dir(exePath)

	// Load config
	a.cfg, err = config.Load(a.baseDir)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		wailsRuntime.MessageDialog(ctx, wailsRuntime.MessageDialogOptions{
			Type:    wailsRuntime.ErrorDialog,
			Title:   "启动失败",
			Message: "加载配置失败: " + err.Error(),
		})
		return
	}

	// Init database
	dbPath := filepath.Join(a.baseDir, a.cfg.Database.Path)
	a.db, err = database.Init(dbPath)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		wailsRuntime.MessageDialog(ctx, wailsRuntime.MessageDialogOptions{
			Type:    wailsRuntime.ErrorDialog,
			Title:   "启动失败",
			Message: "初始化数据库失败: " + err.Error(),
		})
		return
	}

	// Init services
	a.accounts = service.NewAccountService(a.db)
	a.groups = service.NewGroupService(a.db)
	a.proxies = service.NewProxyService(a.db)
	a.apiKeys = service.NewAPIKeyService(a.db)
	a.usage = service.NewUsageService(a.db)

	a.pricing = service.NewPricingService(a.db, a.cfg.Gateway.PricingURL)
	a.pricing.Seed()
	service.SetPricingService(a.pricing)

	a.healthCheck = service.NewHealthCheckService(a.db)

	// Start proxy server in background
	a.proxy = server.New(a.cfg, a.db)
	a.proxy.SetOnUsageLogged(func() {
		wailsRuntime.EventsEmit(a.ctx, "usage:logged")
	})
	go func() {
		addr := fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.Port)
		log.Printf("Proxy server starting on %s", addr)
		if err := a.proxy.Start(); err != nil {
			log.Printf("Proxy server error: %v", err)
		}
	}()

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

func (a *App) GetProxyAddr() string {
	if a.cfg == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.Port)
}

// ===== Dashboard =====

func (a *App) GetDashboardStats(since string) (*model.DashboardStats, error) {
	if a.usage == nil || a.accounts == nil {
		return &model.DashboardStats{}, nil
	}
	stats, err := a.usage.DashboardStats(since)
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
	if a.accounts == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.accounts.List()
}

func (a *App) GetAccount(id int64) (*model.Account, error) {
	if a.accounts == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.accounts.GetByID(id)
}

func (a *App) CreateAccount(account model.Account) error {
	if a.accounts == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.accounts.Create(&account)
}

func (a *App) UpdateAccount(account model.Account) error {
	if a.accounts == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.accounts.Update(&account)
}

func (a *App) DeleteAccount(id int64) error {
	if a.accounts == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.accounts.Delete(id)
}

func (a *App) HealthCheckAccount(id int64, model string) (*service.HealthCheckResult, error) {
	if a.healthCheck == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.healthCheck.CheckAccount(id, model)
}

func (a *App) HealthCheckAllAccounts(model string) ([]service.HealthCheckResult, error) {
	if a.healthCheck == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.healthCheck.CheckAllAccounts(model)
}

// ===== Groups =====

func (a *App) ListGroups() ([]model.Group, error) {
	if a.groups == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.groups.List()
}

func (a *App) GetGroup(id int64) (*model.Group, error) {
	if a.groups == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.groups.GetByID(id)
}

func (a *App) CreateGroup(group model.Group) error {
	if a.groups == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.groups.Create(&group)
}

func (a *App) UpdateGroup(group model.Group) error {
	if a.groups == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.groups.Update(&group)
}

func (a *App) DeleteGroup(id int64) error {
	if a.groups == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.groups.Delete(id)
}

// ===== Proxies =====

func (a *App) ListProxies() ([]model.Proxy, error) {
	if a.proxies == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.proxies.List()
}

func (a *App) GetProxy(id int64) (*model.Proxy, error) {
	if a.proxies == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.proxies.GetByID(id)
}

func (a *App) CreateProxy(proxy model.Proxy) error {
	if a.proxies == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.proxies.Create(&proxy)
}

func (a *App) UpdateProxy(proxy model.Proxy) error {
	if a.proxies == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.proxies.Update(&proxy)
}

func (a *App) DeleteProxy(id int64) error {
	if a.proxies == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.proxies.Delete(id)
}

// ===== API Keys =====

func (a *App) ListAPIKeys() ([]model.APIKey, error) {
	if a.apiKeys == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.apiKeys.List()
}

func (a *App) CreateAPIKey(key model.APIKey) error {
	if a.apiKeys == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.apiKeys.Create(&key)
}

func (a *App) DeleteAPIKey(id int64) error {
	if a.apiKeys == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.apiKeys.Delete(id)
}

// ===== Usage =====

func (a *App) ListUsage(limit, offset int, modelName, startDate, endDate string) (*model.UsageListResult, error) {
	if a.usage == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.usage.List(limit, offset, modelName, startDate, endDate)
}

func (a *App) ListUsageModels() ([]string, error) {
	if a.usage == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.usage.ListModels()
}

// ===== Pricing =====

func (a *App) ListPricing() ([]model.ModelPricing, error) {
	if a.pricing == nil {
		return nil, fmt.Errorf("服务未初始化")
	}
	return a.pricing.List()
}

func (a *App) UpdatePricing(id int64, inputPrice, outputPrice, cacheCreation, cacheRead float64) error {
	if a.pricing == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.pricing.Update(id, inputPrice, outputPrice, cacheCreation, cacheRead)
}

func (a *App) ResetPricing() error {
	if a.pricing == nil {
		return fmt.Errorf("服务未初始化")
	}
	return a.pricing.Reset()
}

func (a *App) FetchRemotePricing() (int, error) {
	if a.pricing == nil {
		return 0, fmt.Errorf("服务未初始化")
	}
	return a.pricing.FetchRemote()
}
