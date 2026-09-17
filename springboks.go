package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	springboksLeague     = "springboks"
	springboksResultsURL = "https://springboks.rugby/match-centre/results"
)

func isSpringboksLeague(league string) bool {
	return league == springboksLeague
}

// The results page has no JSON API. It embeds a schema.org ItemList of
// SportsEvent as JSON-LD in a <script type="application/ld+json"> tag.
// Games that haven't been played yet appear in the same list without
// homeTeamScore/awayTeamScore, so presence of both scores is the only
// available "completed" signal -- there is no status or game-ID field.
var ldJSONPattern = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)

type sbTeam struct {
	Name string `json:"name"`
}

type sbScore struct {
	Value int `json:"value"`
}

type sbEvent struct {
	Type          string   `json:"@type"`
	StartDate     string   `json:"startDate"`
	Competitor    []sbTeam `json:"competitor"`
	HomeTeamScore *sbScore `json:"homeTeamScore"`
	AwayTeamScore *sbScore `json:"awayTeamScore"`
}

type sbListItem struct {
	Item sbEvent `json:"item"`
}

type sbItemList struct {
	ItemListElement []sbListItem `json:"itemListElement"`
}

func fetchSpringboksResults(sport, league string) ([]gameResult, error) {
	resp, err := http.Get(springboksResultsURL) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", springboksResultsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", springboksResultsURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s response: %w", springboksResultsURL, err)
	}

	list, ok := extractResultsList(body)
	if !ok {
		return nil, fmt.Errorf("no embedded results JSON found at %s", springboksResultsURL)
	}

	var results []gameResult
	for _, li := range list.ItemListElement {
		event := li.Item
		if event.HomeTeamScore == nil || event.AwayTeamScore == nil {
			continue
		}
		if len(event.Competitor) != 2 {
			continue
		}
		home, away := event.Competitor[0].Name, event.Competitor[1].Name

		results = append(results, gameResult{
			ID:     springboksGameID(event.StartDate, home, away),
			Sport:  sport,
			League: league,
			Date:   event.StartDate,
			HomeTeam: competitor{
				Name:         home,
				Abbreviation: strings.ToUpper(home),
				Score:        event.HomeTeamScore.Value,
				IsHome:       true,
			},
			AwayTeam: competitor{
				Name:         away,
				Abbreviation: strings.ToUpper(away),
				Score:        event.AwayTeamScore.Value,
				IsHome:       false,
			},
			StatusDescription: "Full Time",
		})
	}
	return results, nil
}

// The page embeds more than one JSON-LD block (e.g. breadcrumb navigation
// is also typically an ItemList of ListItems), so a non-empty
// itemListElement isn't enough to identify the right one. Only accept a
// candidate that actually contains SportsEvent items.
func extractResultsList(page []byte) (sbItemList, bool) {
	for _, match := range ldJSONPattern.FindAllSubmatch(page, -1) {
		var candidate sbItemList
		if err := json.Unmarshal(match[1], &candidate); err != nil {
			continue
		}
		if hasSportsEvents(candidate) {
			return candidate, true
		}
	}
	return sbItemList{}, false
}

func hasSportsEvents(list sbItemList) bool {
	for _, li := range list.ItemListElement {
		if li.Item.Type == "SportsEvent" {
			return true
		}
	}
	return false
}

// Games have no provider-issued ID, so one is derived from the kickoff time
// and team names -- stable across runs since neither changes for a given game.
func springboksGameID(startDate, home, away string) string {
	slug := strings.NewReplacer(" ", "_", ":", "", "/", "-").Replace(home + "_v_" + away + "_" + startDate)
	return springboksLeague + "_" + strings.ToLower(slug)
}
