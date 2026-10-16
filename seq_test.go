package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSeqSink(t *testing.T) {
	var got []map[string]any
	var path, query, contentType, apiKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		contentType, apiKey = r.Header.Get("Content-Type"), r.Header.Get("X-Seq-ApiKey")
		dec := json.NewDecoder(r.Body)
		for {
			var ev map[string]any
			if dec.Decode(&ev) != nil {
				break
			}
			got = append(got, ev)
		}
		w.WriteHeader(http.StatusCreated)
	}))

	s := newSeqSink()
	s.url, s.apiKey = srv.URL+"/", "secret"
	fmt.Fprintln(seqLevelWriter{s, "Information"}, "[espn] fetching football/nfl...")
	fmt.Fprintln(seqLevelWriter{s, "Error"}, "[notify] POST failed")
	s.flush()

	if path != "/api/events/raw" || query != "clef" {
		t.Errorf("request URL = %s?%s", path, query)
	}
	if contentType != "application/vnd.serilog.clef" || apiKey != "secret" {
		t.Errorf("headers: Content-Type=%q X-Seq-ApiKey=%q", contentType, apiKey)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0]["@m"] != "[espn] fetching football/nfl..." || got[0]["@l"] != "Information" || got[0]["Module"] != "espn" {
		t.Errorf("event 0 = %v", got[0])
	}
	if got[1]["@l"] != "Error" || got[1]["Module"] != "notify" || got[1]["RunId"] != s.runID {
		t.Errorf("event 1 = %v", got[1])
	}

	// An unreachable Seq must only print a warning, not panic or exit.
	srv.Close()
	down := newSeqSink()
	down.url = srv.URL
	fmt.Fprintln(seqLevelWriter{down, "Information"}, "[config] tracking 1 team(s)")
	down.flush()
}
