package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const hockeytechBase = "https://lscluster.hockeytech.com/feed/index.php"

type hockeytechLeague struct {
	key        string
	clientCode string
}

var hockeytechLeagues = map[string]hockeytechLeague{
	"pwhl": {key: "694cfeed58c932ee", clientCode: "pwhl"},
	"echl": {key: "e6219ee34f4b5200", clientCode: "echl"},
}

func isHockeytechLeague(league string) bool {
	_, ok := hockeytechLeagues[league]
	return ok
}

// HockeyTech API response shapes

type htResponse struct {
	SiteKit htSiteKit `json:"SiteKit"`
}

type htSiteKit struct {
	Scorebar []htGame `json:"Scorebar"`
}

type htGame struct {
	GameID           string `json:"game_id"`
	DatePlayed       string `json:"date_played"`
	GameStatus       string `json:"GameStatus"`
	GameStatusString string `json:"GameStatusString"`
	HomeTeam         string `json:"HomeTeam"`
	HomeTeamName     string `json:"HomeTeamName"`
	HomeGoals        string `json:"HomeGoals"`
	VisitorTeam      string `json:"VisitorTeam"`
	VisitorTeamName  string `json:"VisitorTeamName"`
	VisitorGoals     string `json:"VisitorGoals"`
	IsPlayoffGame    string `json:"IsPlayoffGame"`
}

func fetchHockeytechScoreboard(sport, league string) ([]gameResult, error) {
	ht, ok := hockeytechLeagues[league]
	if !ok {
		return nil, fmt.Errorf("unknown hockeytech league: %s", league)
	}
	url := fmt.Sprintf(
		"%s?key=%s&client_code=%s&feed=modulekit&view=scorebar&numberofdaysahead=0&numberofdaysback=1&lang=en&fmt=json",
		hockeytechBase, ht.key, ht.clientCode,
	)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}

	var body htResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding %s response: %w", url, err)
	}

	var results []gameResult
	for _, g := range body.SiteKit.Scorebar {
		// GameStatus "4" = final, "5" = unofficial final
		if g.GameStatus != "4" && g.GameStatus != "5" {
			continue
		}
		results = append(results, gameResult{
			ID:     league + "_" + g.GameID,
			Sport:  sport,
			League: league,
			Date:   g.DatePlayed,
			HomeTeam: competitor{
				Name:         g.HomeTeamName,
				Abbreviation: strings.ToUpper(g.HomeTeam),
				Score:        parseScore(g.HomeGoals),
				IsHome:       true,
			},
			AwayTeam: competitor{
				Name:         g.VisitorTeamName,
				Abbreviation: strings.ToUpper(g.VisitorTeam),
				Score:        parseScore(g.VisitorGoals),
				IsHome:       false,
			},
			StatusDescription: g.GameStatusString,
			IsPostseason:      g.IsPlayoffGame == "1",
		})
	}
	return results, nil
}
