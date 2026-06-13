package main

import (
	"fmt"
	"log"
	"os"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	log.SetFlags(0)

	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	fmt.Printf("=== Game Over Man %s ===\n", version)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("[config] %v", err)
	}
	log.Printf("[config] tracking %d team(s)", len(cfg.Teams))

	state := loadState(cfg.StateFilePath)
	state = pruneState(state, cfg.PruneAfterDays)

	seen := make(map[string]bool)
	notified := 0

	for _, t := range cfg.Teams {
		key := t.Sport + "/" + t.League
		if seen[key] {
			continue
		}
		seen[key] = true

		log.Printf("[espn] fetching %s...", key)
		games, err := fetchScoreboard(t.Sport, t.League)
		if err != nil {
			log.Printf("[espn] %v", err)
			continue
		}
		log.Printf("[espn] %d completed game(s)", len(games))

		for _, game := range games {
			if !isTrackedGame(game, cfg.Teams) {
				continue
			}
			if hasBeenNotified(state, game.ID) {
				log.Printf("[state] already notified for game %s, skipping", game.ID)
				continue
			}
			if err := sendNotification(cfg, game); err != nil {
				log.Printf("[notify] %v", err)
				continue
			}
			markNotified(&state, game.ID)
			notified++
		}
	}

	if err := saveState(cfg.StateFilePath, state); err != nil {
		log.Printf("[state] failed to save: %v", err)
		os.Exit(1)
	}

	fmt.Printf("=== Done. %d new notification(s) sent. ===\n", notified)
}
