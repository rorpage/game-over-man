package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	springboksLeague     = "springboks"
	springboksMatchesURL = "https://springboks.rugby/api/match-centre/matches"
)

func isSpringboksLeague(league string) bool {
	return league == springboksLeague
}

type sbTeam struct {
	IsHomeTeam bool   `json:"isHomeTeam"`
	Score      int    `json:"score"`
	Name       string `json:"name"`
	ImagePath  string `json:"imagePath"`
}

type sbMatch struct {
	MatchID     string   `json:"matchId"`
	UTCDate     string   `json:"utcDate"`
	StatsStatus string   `json:"statsStatus"`
	Teams       []sbTeam `json:"teams"`
}

type sbMatchesResponse struct {
	Items []sbMatch `json:"items"`
}

// Covers yesterday through the end of today (UTC) so a match that finishes
// shortly after midnight UTC, or between two polling runs, is never missed --
// the same lookback the HockeyTech provider uses.
func fetchSpringboksResults(sport, league string) ([]gameResult, error) {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -1).Format("2006-01-02")
	end := now.Format("2006-01-02") + "T23:59:59"

	q := url.Values{}
	q.Set("startDate", start)
	q.Set("endDate", end)
	q.Set("pageIndex", "0")
	q.Set("pageSize", "100") // assumes fewer than 100 matches org-wide over the 2-day window; no pagination handling
	q.Set("IsAscending", "true")
	reqURL := springboksMatchesURL + "?" + q.Encode()

	resp, err := http.Get(reqURL) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", reqURL, resp.StatusCode)
	}

	var body sbMatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding %s response: %w", reqURL, err)
	}

	var results []gameResult
	for _, m := range body.Items {
		if m.StatsStatus != "Complete" {
			continue
		}

		var home, away *sbTeam
		for i := range m.Teams {
			t := &m.Teams[i]
			if t.IsHomeTeam {
				home = t
			} else {
				away = t
			}
		}
		if home == nil || away == nil {
			continue
		}

		date := m.UTCDate
		if !strings.HasSuffix(date, "Z") {
			date += "Z"
		}

		results = append(results, gameResult{
			ID:     springboksLeague + "_" + m.MatchID,
			Sport:  sport,
			League: league,
			Date:   date,
			HomeTeam: competitor{
				Name:         home.Name,
				Abbreviation: strings.ToUpper(home.Name),
				Score:        home.Score,
				IsHome:       true,
				LogoURL:      home.ImagePath,
			},
			AwayTeam: competitor{
				Name:         away.Name,
				Abbreviation: strings.ToUpper(away.Name),
				Score:        away.Score,
				IsHome:       false,
				LogoURL:      away.ImagePath,
			},
			StatusDescription: "Full Time",
		})
	}
	return results, nil
}
