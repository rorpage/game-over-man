package main

import (
	"testing"
	"time"
)

// An unplayed match reports score 0 (not null) with all status flags false.
// It must not count as finished until kickoff + springboksMinMatchDuration.
func TestSpringboksUnplayedMatchNotFinished(t *testing.T) {
	zero := 0
	body := sbMatchesResponse{Items: []sbMatch{{
		MatchID: "m1",
		UTCDate: "2026-11-13T20:10:00",
		Teams: []sbTeam{
			{IsHomeTeam: true, Score: &zero, Name: "France"},
			{IsHomeTeam: false, Score: &zero, Name: "Springboks"},
		},
	}}}
	kickoff := time.Date(2026, 11, 13, 20, 10, 0, 0, time.UTC)

	if got := parseSpringboksMatches(body, "rugby", "springboks", kickoff.Add(-24*time.Hour)); len(got) != 0 {
		t.Errorf("day before kickoff: got %d results, want 0", len(got))
	}
	if got := parseSpringboksMatches(body, "rugby", "springboks", kickoff.Add(10*time.Minute)); len(got) != 0 {
		t.Errorf("10 minutes after kickoff: got %d results, want 0", len(got))
	}
	if got := parseSpringboksMatches(body, "rugby", "springboks", kickoff.Add(2*time.Hour)); len(got) != 1 {
		t.Errorf("2 hours after kickoff: got %d results, want 1", len(got))
	}
}
