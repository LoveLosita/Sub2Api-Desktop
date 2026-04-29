package server

import (
	"desktop-proxy/internal/config"
	"desktop-proxy/internal/database"
	"desktop-proxy/internal/service"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg       *config.Config
	db        *database.DB
	gateway   *service.GatewayService
	scheduler *service.SchedulerService
	apiKeys   *service.APIKeyService
	usage     *service.UsageService
	accounts  *service.AccountService
}

func New(cfg *config.Config, db *database.DB) *Server {
	return &Server{
		cfg:       cfg,
		db:        db,
		gateway:   service.NewGatewayService(db),
		scheduler: service.NewSchedulerService(db),
		apiKeys:   service.NewAPIKeyService(db),
		usage:     service.NewUsageService(db),
		accounts:  service.NewAccountService(db),
	}
}

func (s *Server) SetOnUsageLogged(fn func()) {
	s.gateway.SetOnUsageLogged(fn)
}

func (s *Server) Start() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Auth middleware
	r.Use(s.authMiddleware)

	// Claude Messages API
	r.POST("/v1/messages", s.handleClaude)
	r.POST("/api/v1/messages", s.handleClaude)

	// OpenAI compatible
	r.POST("/v1/chat/completions", s.handleOpenAIChat)
	r.POST("/v1/responses", s.handleOpenAIResponses)
	r.POST("/v1/images/generations", s.handleOpenAIImages)
	r.GET("/v1/models", s.handleModels)

	// Gemini native
	r.Any("/v1beta/*path", s.handleGemini)

	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	return r.Run(addr)
}

func (s *Server) authMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		// Check query param (Gemini style)
		authHeader = "Bearer " + c.Query("key")
	}
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		apiKey := authHeader[7:]
		k, g, err := s.apiKeys.ValidateKey(apiKey)
		if err == nil && k != nil {
			c.Set("api_key", k)
			c.Set("group", g)
			c.Next()
			return
		}
	}
	c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
		"type":    "authentication_error",
		"message": "Invalid API key",
	}})
	c.Abort()
}
