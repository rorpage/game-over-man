package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// errLog is for failures; the standard logger is for everything else.
// Both go to stderr, and the Seq level depends on which one is used.
var errLog = log.New(os.Stderr, "", 0)

// seqSink buffers log lines as CLEF events and sends them to Seq in a
// single request at the end of the run, since the binary is one-shot.
type seqSink struct {
	mu     sync.Mutex
	url    string
	apiKey string
	runID  string
	buf    bytes.Buffer
}

func newSeqSink() *seqSink {
	b := make([]byte, 8)
	rand.Read(b)
	// Read env here as well as in loadConfig so a config load failure can still be sent.
	return &seqSink{url: os.Getenv("SEQ_URL"), apiKey: os.Getenv("SEQ_API_KEY"), runID: hex.EncodeToString(b)}
}

func (s *seqSink) install() {
	log.SetOutput(io.MultiWriter(os.Stderr, seqLevelWriter{s, "Information"}))
	errLog.SetOutput(io.MultiWriter(os.Stderr, seqLevelWriter{s, "Error"}))
}

type seqLevelWriter struct {
	s     *seqSink
	level string
}

// Write never returns an error, because io.MultiWriter stops at the first
// failing writer and stderr output must not depend on Seq.
func (w seqLevelWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	ev := map[string]any{
		"@t":          time.Now().UTC().Format(time.RFC3339Nano),
		"@m":          msg,
		"@l":          w.level,
		"Application": "game-over-man",
		"Version":     version,
		"RunId":       w.s.runID,
	}
	if strings.HasPrefix(msg, "[") {
		if i := strings.Index(msg, "]"); i > 1 {
			ev["Module"] = msg[1:i]
		}
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return len(p), nil
	}
	w.s.mu.Lock()
	w.s.buf.Write(line)
	w.s.buf.WriteByte('\n')
	w.s.mu.Unlock()
	return len(p), nil
}

// flush reports problems on stderr only (not through log, which would write
// back into the buffer) and never fails the run.
func (s *seqSink) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.url == "" || s.buf.Len() == 0 {
		return
	}

	req, err := http.NewRequest("POST", strings.TrimRight(s.url, "/")+"/api/events/raw?clef", &s.buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[seq] could not send logs: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/vnd.serilog.clef")
	if s.apiKey != "" {
		req.Header.Set("X-Seq-ApiKey", s.apiKey)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[seq] could not send logs: %v\n", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "[seq] could not send logs: HTTP %d\n", resp.StatusCode)
	}
}
