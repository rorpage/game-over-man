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
	Score      *int   `json:"score"` // null, not 0, for a match that hasn't been played -- must stay a pointer
	Name       string `json:"name"`
	ImagePath  string `json:"imagePath"`
}

type sbMatch struct {
	MatchID     string   `json:"matchId"`
	UTCDate     string   `json:"utcDate"`
	IsLive      bool     `json:"isLive"`
	IsCancelled bool     `json:"isCancelled"`
	IsPostponed bool     `json:"isPostponed"`
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

	return parseSpringboksMatches(body, sport, league, time.Now()), nil
}

// A rugby match cannot finish sooner than this after kickoff (80 minutes of
// play plus halftime), so nothing earlier can be a final result.
const springboksMinMatchDuration = 90 * time.Minute

// statsStatus is not a reliable completion signal in this API -- it's seen
// "Live" on a finished blowout and "None" on a finished, non-blowout match in
// the same response. Scores are not reliable either: an unplayed match can
// report score 0 (not null) for both teams. A match counts as finished only
// when isLive/isCancelled/isPostponed are all false, both scores are non-null,
// and kickoff was at least springboksMinMatchDuration ago.
func parseSpringboksMatches(body sbMatchesResponse, sport, league string, now time.Time) []gameResult {
	var results []gameResult
	for _, m := range body.Items {
		if m.IsLive || m.IsCancelled || m.IsPostponed {
			continue
		}

		kickoff, err := time.Parse(time.RFC3339, strings.TrimSuffix(m.UTCDate, "Z")+"Z")
		if err != nil || now.Before(kickoff.Add(springboksMinMatchDuration)) {
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
		if home.Score == nil || away.Score == nil {
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
				Score:        *home.Score,
				IsHome:       true,
				LogoURL:      home.ImagePath,
			},
			AwayTeam: competitor{
				Name:         away.Name,
				Abbreviation: strings.ToUpper(away.Name),
				Score:        *away.Score,
				IsHome:       false,
				LogoURL:      away.ImagePath,
			},
			StatusDescription: "Full Time",
		})
	}
	return results
}
