package main

import (
	"context"
	"desktop-proxy/internal/database"
	"log"
	"os"
	"path/filepath"
)

// App is the main application struct bound to the Wails frontend.
type App struct {
	ctx context.Context
	db  *database.DB
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the Wails app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	baseDir := filepath.Dir(exePath)

	dbPath := filepath.Join(baseDir, "data.db")
	a.db, err = database.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized at:", dbPath)
}

// shutdown is called when the Wails app closes.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// GetAppVersion returns the application version.
func (a *App) GetAppVersion() string {
	return "0.1.0"
}
