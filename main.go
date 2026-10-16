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

	seq := newSeqSink()
	seq.install()

	log.Printf("=== Game Over Man %s ===", version)

	cfg, err := loadConfig()
	if err != nil {
		errLog.Printf("[config] %v", err)
		seq.flush()
		os.Exit(1)
	}
	seq.url, seq.apiKey = cfg.SeqURL, cfg.SeqAPIKey
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

		var games []gameResult
		var err error
		if isHockeytechLeague(t.League) {
			log.Printf("[hockeytech] fetching %s...", t.League)
			games, err = fetchHockeytechScoreboard(t.Sport, t.League)
			if err != nil {
				errLog.Printf("[hockeytech] %v", err)
				continue
			}
			log.Printf("[hockeytech] %d completed game(s)", len(games))
		} else if isSpringboksLeague(t.League) {
			log.Printf("[springboks] fetching results...")
			games, err = fetchSpringboksResults(t.Sport, t.League)
			if err != nil {
				errLog.Printf("[springboks] %v", err)
				continue
			}
			log.Printf("[springboks] %d completed game(s)", len(games))
		} else {
			log.Printf("[espn] fetching %s...", key)
			games, err = fetchScoreboard(t.Sport, t.League)
			if err != nil {
				errLog.Printf("[espn] %v", err)
				continue
			}
			log.Printf("[espn] %d completed game(s)", len(games))
		}

		for _, game := range games {
			if !isTrackedGame(game, cfg.Teams) {
				continue
			}
			if hasBeenNotified(state, game.ID) {
				log.Printf("[state] already notified for game %s, skipping", game.ID)
				continue
			}
			if err := sendNotification(cfg, game); err != nil {
				errLog.Printf("[notify] %v", err)
				continue
			}
			markNotified(&state, game.ID)
			notified++
		}
	}

	if err := saveState(cfg.StateFilePath, state); err != nil {
		errLog.Printf("[state] failed to save: %v", err)
		seq.flush()
		os.Exit(1)
	}

	log.Printf("=== Done. %d new notification(s) sent. ===", notified)
	seq.flush()
}
