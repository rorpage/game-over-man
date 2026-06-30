package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
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

	if isDraw && (home.IsWinner || away.IsWinner) {
		// Scores are tied but ESPN reports a winner (e.g. penalty shootout or shootout).
		// Don't report this as a draw — surface the actual winner.
		isDraw = false
		w, l := home, away
		if away.IsWinner {
			w, l = away, home
		}
		wn, ln := w.Name, l.Name
		winner = &wn
		loser = &ln
		summary = fmt.Sprintf("%s wins (%d-%d) (%s %s)",
			w.Name, w.Score, l.Score, strings.ToUpper(game.League), game.StatusDescription)
	} else if isDraw {
		summary = fmt.Sprintf("%s %d, %s %d -- Draw (%s %s)",
			away.Name, away.Score, home.Name, home.Score, strings.ToUpper(game.League), game.StatusDescription)
	} else {
		w, l := home, away
		if away.Score > home.Score {
			w, l = away, home
		}
		wn, ln := w.Name, l.Name
		winner = &wn
		loser = &ln
		summary = fmt.Sprintf("%s %d, %s %d (%s %s)",
			w.Name, w.Score, l.Name, l.Score, strings.ToUpper(game.League), game.StatusDescription)
	}

	return notificationPayload{Game: game, Summary: summary, Winner: winner, Loser: loser, IsDraw: isDraw}
}

func sportEmoji(sport, league string) string {
	switch strings.ToLower(league) {
	case "nhl", "pwhl", "echl", "nwhl":
		return "🏒"
	case "nfl":
		return "🏈"
	case "mlb":
		return "⚾"
	case "nba", "nba_g_league", "wnba":
		return "🏀"
	case "mls":
		return "⚽"
	}
	switch strings.ToLower(sport) {
	case "hockey":
		return "🏒"
	case "football":
		return "🏈"
	case "baseball":
		return "⚾"
	case "basketball":
		return "🏀"
	case "soccer":
		return "⚽"
	}
	return "🏅"
}

func buildSlackBlocks(payload notificationPayload) ([]byte, error) {
	game := payload.Game
	emoji := sportEmoji(game.Sport, game.League)

	header := fmt.Sprintf("%s  %s", emoji, strings.ToUpper(game.League))
	if game.IsPostseason {
		header += " · Playoffs"
	}
	header += " · " + game.StatusDescription

	awayWon := !payload.IsDraw && payload.Winner != nil && *payload.Winner == game.AwayTeam.Name
	homeWon := !payload.IsDraw && payload.Winner != nil && *payload.Winner == game.HomeTeam.Name

	teamLabel := func(score int, abbrev, name, side string, winner bool) string {
		label := fmt.Sprintf("%d  %s  %s  (%s)", score, abbrev, name, side)
		if winner {
			label = "*" + label + "*"
		}
		return label
	}

	teamBlock := func(label, logoURL, altText string) map[string]any {
		elements := []any{map[string]any{"type": "mrkdwn", "text": label}}
		if logoURL != "" {
			elements = append([]any{map[string]any{
				"type":      "image",
				"image_url": logoURL,
				"alt_text":  altText,
			}}, elements...)
		}
		return map[string]any{"type": "context", "elements": elements}
	}

	blocks := []any{
		map[string]any{
			"type": "header",
			"text": map[string]any{"type": "plain_text", "text": header, "emoji": true},
		},
		teamBlock(
			teamLabel(game.AwayTeam.Score, game.AwayTeam.Abbreviation, game.AwayTeam.Name, "Away", awayWon),
			game.AwayTeam.LogoURL, game.AwayTeam.Name,
		),
		teamBlock(
			teamLabel(game.HomeTeam.Score, game.HomeTeam.Abbreviation, game.HomeTeam.Name, "Home", homeWon),
			game.HomeTeam.LogoURL, game.HomeTeam.Name,
		),
	}

	return json.Marshal(map[string]any{"text": payload.Summary, "blocks": blocks})
}

func buildBody(cfg *appConfig, payload notificationPayload) ([]byte, error) {
	switch cfg.NotificationType {
	case "slack":
		return buildSlackBlocks(payload)
	case "discord":
		return json.Marshal(map[string]string{"content": payload.Summary})
	case "template":
		tmpl, err := template.New("notification").Parse(cfg.NotificationTemplate)
		if err != nil {
			return nil, fmt.Errorf("parsing notification template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, payload); err != nil {
			return nil, fmt.Errorf("executing notification template: %w", err)
		}
		return buf.Bytes(), nil
	default: // "webhook"
		return json.Marshal(payload)
	}
}

func sendNotification(cfg *appConfig, game gameResult) error {
	payload := buildPayload(game)

	body, err := buildBody(cfg, payload)
	if err != nil {
		return fmt.Errorf("building notification body: %w", err)
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
