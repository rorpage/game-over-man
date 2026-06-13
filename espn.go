package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const espnBase = "http://site.api.espn.com/apis/site/v2/sports"

type competitor struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Score        int    `json:"score"`
	IsHome       bool   `json:"isHome"`
}

type gameResult struct {
	ID                string     `json:"id"`
	Sport             string     `json:"sport"`
	League            string     `json:"league"`
	Date              string     `json:"date"`
	HomeTeam          competitor `json:"homeTeam"`
	AwayTeam          competitor `json:"awayTeam"`
	StatusDescription string     `json:"statusDescription"`
}

// ESPN API response shapes

type espnScoreboard struct {
	Events []espnEvent `json:"events"`
}

type espnEvent struct {
	ID           string           `json:"id"`
	Date         string           `json:"date"`
	Status       espnStatus       `json:"status"`
	Competitions []espnCompetition `json:"competitions"`
}

type espnStatus struct {
	Type espnStatusType `json:"type"`
}

type espnStatusType struct {
	Completed   bool   `json:"completed"`
	Description string `json:"description"`
}

type espnCompetition struct {
	Competitors []espnCompetitor `json:"competitors"`
}

type espnCompetitor struct {
	HomeAway string   `json:"homeAway"`
	Team     espnTeam `json:"team"`
	Score    string   `json:"score"`
}

type espnTeam struct {
	Abbreviation string `json:"abbreviation"`
	DisplayName  string `json:"displayName"`
}

func fetchScoreboard(sport, league string) ([]gameResult, error) {
	url := fmt.Sprintf("%s/%s/%s/scoreboard", espnBase, sport, league)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}

	var sb espnScoreboard
	if err := json.NewDecoder(resp.Body).Decode(&sb); err != nil {
		return nil, fmt.Errorf("decoding %s response: %w", url, err)
	}

	var results []gameResult
	for _, event := range sb.Events {
		if !event.Status.Type.Completed {
			continue
		}
		if game, ok := parseEvent(event, sport, league); ok {
			results = append(results, game)
		}
	}
	return results, nil
}

func parseEvent(event espnEvent, sport, league string) (gameResult, bool) {
	if len(event.Competitions) == 0 {
		return gameResult{}, false
	}
	comp := event.Competitions[0]

	var home, away *espnCompetitor
	for i := range comp.Competitors {
		c := &comp.Competitors[i]
		switch c.HomeAway {
		case "home":
			home = c
		case "away":
			away = c
		}
	}
	if home == nil || away == nil {
		return gameResult{}, false
	}

	return gameResult{
		ID:     event.ID,
		Sport:  sport,
		League: league,
		Date:   event.Date,
		HomeTeam: competitor{
			Name:         home.Team.DisplayName,
			Abbreviation: strings.ToUpper(home.Team.Abbreviation),
			Score:        parseScore(home.Score),
			IsHome:       true,
		},
		AwayTeam: competitor{
			Name:         away.Team.DisplayName,
			Abbreviation: strings.ToUpper(away.Team.Abbreviation),
			Score:        parseScore(away.Score),
			IsHome:       false,
		},
		StatusDescription: event.Status.Type.Description,
	}, true
}

func parseScore(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func isTrackedGame(game gameResult, teams []teamConfig) bool {
	for _, t := range teams {
		if t.Sport != game.Sport || t.League != game.League {
			continue
		}
		if t.Abbreviation == game.HomeTeam.Abbreviation || t.Abbreviation == game.AwayTeam.Abbreviation {
			return true
		}
	}
	return false
}
