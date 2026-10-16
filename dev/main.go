// Command dev runs game-over-man from source against a local webhook listener.
// Run it from the repo root: go run ./dev
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
)

const (
	configPath = "config.json"
	statePath  = "state.json"
)

const defaultConfig = `{
  "teams": [
    { "sport": "hockey", "league": "nhl",        "abbreviation": "*" },
    { "sport": "rugby",  "league": "springboks", "abbreviation": "*" }
  ]
}
`

func main() {
	fresh := flag.Bool("fresh", false, "delete state.json first so already-notified games are sent again")
	flag.Parse()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
			log.Fatalf("[dev] writing %s: %v", configPath, err)
		}
		log.Printf("[dev] created %s with a default test config; edit it to change leagues", configPath)
	}

	if *fresh {
		if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
			log.Fatalf("[dev] removing %s: %v", statePath, err)
		}
		log.Printf("[dev] removed %s", statePath)
	}

	// Port 0 picks a free port, so nothing else on the machine can conflict.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("[dev] starting listener: %v", err)
	}
	var received atomic.Int32
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var pretty bytes.Buffer
		if json.Indent(&pretty, body, "", "  ") == nil {
			body = pretty.Bytes()
		}
		n := received.Add(1)
		fmt.Printf("\n--- notification %d (%s %s) ---\n%s\n\n", n, r.Method, r.Header.Get("Content-Type"), body)
	}))

	url := "http://" + ln.Addr().String()
	log.Printf("[dev] listener on %s", url)

	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"CONFIG_FILE="+configPath,
		"STATE_FILE="+statePath,
		"NOTIFICATION_URL="+url,
	)
	runErr := cmd.Run()

	log.Printf("[dev] done: %d notification(s) received", received.Load())
	if runErr != nil {
		log.Fatalf("[dev] game-over-man failed: %v", runErr)
	}
}
