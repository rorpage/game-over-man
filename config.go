package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultPruneDays = 30

func userAppDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "game-over-man")
}

type teamConfig struct {
	Sport          string `json:"sport"`
	League         string `json:"league"`
	Abbreviation   string `json:"abbreviation"`
	PostseasonOnly bool   `json:"postseasonOnly"`
}

type appConfig struct {
	Teams                []teamConfig      `json:"teams"`
	NotificationURL      string            `json:"notificationUrl"`
	NotificationMethod   string            `json:"notificationMethod"`
	NotificationHeaders  map[string]string `json:"notificationHeaders"`
	NotificationType     string            `json:"notificationType"`
	NotificationTemplate string            `json:"notificationTemplate"`
	StateFilePath        string            `json:"stateFilePath"`
	PruneAfterDays       int               `json:"pruneAfterDays"`
}

func loadConfig() (*appConfig, error) {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = filepath.Join(userAppDir(), "config.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg appConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if url := os.Getenv("NOTIFICATION_URL"); url != "" {
		cfg.NotificationURL = url
	}
	if cfg.NotificationURL == "" {
		return nil, fmt.Errorf("notification URL required: set NOTIFICATION_URL env var or notificationUrl in config")
	}

	if sf := os.Getenv("STATE_FILE"); sf != "" {
		cfg.StateFilePath = sf
	}
	if cfg.StateFilePath == "" {
		cfg.StateFilePath = filepath.Join(userAppDir(), "state.json")
	}

	if cfg.PruneAfterDays <= 0 {
		cfg.PruneAfterDays = defaultPruneDays
	}
	if cfg.NotificationMethod == "" {
		cfg.NotificationMethod = "POST"
	}
	if cfg.NotificationType == "" {
		cfg.NotificationType = "webhook"
	}
	switch cfg.NotificationType {
	case "webhook", "slack", "discord", "template":
	default:
		return nil, fmt.Errorf("invalid notificationType %q: must be webhook, slack, discord, or template", cfg.NotificationType)
	}
	if cfg.NotificationType == "template" && cfg.NotificationTemplate == "" {
		return nil, fmt.Errorf("notificationTemplate is required when notificationType is \"template\"")
	}

	if len(cfg.Teams) == 0 {
		return nil, fmt.Errorf("no teams configured in %s", path)
	}
	for i := range cfg.Teams {
		cfg.Teams[i].Sport = strings.ToLower(cfg.Teams[i].Sport)
		cfg.Teams[i].League = strings.ToLower(cfg.Teams[i].League)
		cfg.Teams[i].Abbreviation = strings.ToUpper(cfg.Teams[i].Abbreviation)
	}

	return &cfg, nil
}
