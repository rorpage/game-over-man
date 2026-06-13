package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type notificationPayload struct {
	Game    gameResult `json:"game"`
	Summary string     `json:"summary"`
	Winner  *string    `json:"winner"`
	Loser   *string    `json:"loser"`
	IsDraw  bool       `json:"isDraw"`
}

func buildPayload(game gameResult) notificationPayload {
	home, away := game.HomeTeam, game.AwayTeam
	isDraw := home.Score == away.Score

	var winner, loser *string
	var summary string

	if isDraw {
		summary = fmt.Sprintf("Final: %s %d, %s %d -- Draw (%s)",
			away.Name, away.Score, home.Name, home.Score, game.StatusDescription)
	} else {
		w, l := home, away
		if away.Score > home.Score {
			w, l = away, home
		}
		wn, ln := w.Name, l.Name
		winner = &wn
		loser = &ln
		summary = fmt.Sprintf("Final: %s %d, %s %d (%s)",
			w.Name, w.Score, l.Name, l.Score, game.StatusDescription)
	}

	return notificationPayload{Game: game, Summary: summary, Winner: winner, Loser: loser, IsDraw: isDraw}
}

func sendNotification(cfg *appConfig, game gameResult) error {
	payload := buildPayload(game)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest(cfg.NotificationMethod, cfg.NotificationURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.NotificationHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", cfg.NotificationMethod, cfg.NotificationURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d", cfg.NotificationMethod, cfg.NotificationURL, resp.StatusCode)
	}

	fmt.Printf("[notify] %s\n", payload.Summary)
	return nil
}
