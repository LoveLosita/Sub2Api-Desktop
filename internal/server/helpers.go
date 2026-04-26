package server

import (
	"desktop-proxy/internal/model"
	"strings"
)

func getBaseURL(account *model.Account, defaultURL string) string {
	if account.BaseURL != nil && *account.BaseURL != "" {
		return strings.TrimRight(*account.BaseURL, "/")
	}
	return defaultURL
}
