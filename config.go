package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	defaultConfigFile = "/etc/game-over-man/config.json"
	defaultStateFile  = "/var/lib/game-over-man/state.json"
	defaultPruneDays  = 30
)

type teamConfig struct {
	Sport        string `json:"sport"`
	League       string `json:"league"`
	Abbreviation string `json:"abbreviation"`
}

type appConfig struct {
	Teams               []teamConfig      `json:"teams"`
	NotificationURL     string            `json:"notificationUrl"`
	NotificationMethod  string            `json:"notificationMethod"`
	NotificationHeaders map[string]string `json:"notificationHeaders"`
	StateFilePath       string            `json:"stateFilePath"`
	PruneAfterDays      int               `json:"pruneAfterDays"`
}

func loadConfig() (*appConfig, error) {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = defaultConfigFile
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
		cfg.StateFilePath = defaultStateFile
	}

	if cfg.PruneAfterDays <= 0 {
		cfg.PruneAfterDays = defaultPruneDays
	}
	if cfg.NotificationMethod == "" {
		cfg.NotificationMethod = "POST"
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
