# AGENTS.md

Context for AI agents (Claude Code, Copilot, etc.) working in this repository.

## What this project does

Game Over Man is a one-shot Go binary for home servers. It queries the ESPN public scoreboard API for any sport/league you configure, finds completed games involving tracked teams, and sends a single webhook notification per game. Idempotency is maintained via a JSON state file. It runs directly on Linux or macOS -- no Docker, no runtime dependencies.

## Repository layout

```
main.go       -- entry point; orchestrates config, ESPN fetch, notify, state
config.go     -- config types, loading from JSON + env var overrides
espn.go       -- ESPN API fetch, response parsing, team matching
notifier.go   -- builds notification payload, POSTs to webhook URL
state.go      -- reads/writes/prunes the state file

go.mod                -- module definition; no external dependencies
config.example.json   -- copy to config.json (gitignored) to run locally

.github/workflows/
  publish.yml   -- on v* tag: cross-compiles binaries for 5 platforms, creates GitHub Release

deploy/
  systemd/
    game-over-man.service  -- oneshot unit; runs binary as game-over-man user
    game-over-man.timer    -- fires every 10 minutes, Persistent=true
    env.example            -- template for /etc/game-over-man/env (holds NOTIFICATION_URL)
    install.sh             -- downloads binary (or builds from source), creates user/dirs, enables timer
```

## Key design decisions

- **Single static binary, no runtime dependencies.** Go's standard library handles HTTP and JSON. CGO is disabled so the binary runs on any Linux/macOS without glibc or other shared libraries.
- **No Docker.** This project targets home server use. Docker adds path, permission, and user complexity that the Go binary approach eliminates entirely.
- **One-shot execution.** The binary runs, checks scores, and exits. Scheduling is the caller's responsibility (cron or systemd timer).
- **State file for idempotency.** `state.json` records notified game IDs with timestamps. Entries older than `pruneAfterDays` (default 30) are pruned on each run. If a notification POST fails, the game ID is not recorded, so it will be retried on the next run.
- **Config-file-first, env-var override.** `NOTIFICATION_URL`, `CONFIG_FILE`, and `STATE_FILE` env vars override their config file equivalents. Keep the notification URL in an env var to avoid committing secrets.
- **Case-normalized inputs.** Sport and league values are lowercased; abbreviations are uppercased on config load so comparisons are always case-insensitive.

## Default file paths

| Purpose | Default |
|---|---|
| Config | `/etc/game-over-man/config.json` |
| State | `/var/lib/game-over-man/state.json` |

## ESPN API

Base URL: `http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard`

This is an unofficial but stable ESPN endpoint. It returns today's games with scores and status. The `status.type.completed` boolean determines whether a game is final. `status.type.description` carries strings like `"Final"`, `"Final/OT"`, `"Final/SO"`.

Known working sport/league pairs: `football/nfl`, `football/college-football`, `basketball/nba`, `basketball/wnba`, `basketball/mens-college-basketball`, `basketball/womens-college-basketball`, `baseball/mlb`, `hockey/nhl`, `hockey/ahl`, `soccer/usa.1`, `soccer/usa.nwsl`, `soccer/eng.1`, `soccer/esp.1`, `soccer/ita.1`, `soccer/ger.1`, `soccer/fra.1`, `soccer/uefa.champions`.

## Adding a new league

1. Verify the endpoint: `curl "http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard"`
2. Add entries to the `teams` array in config
3. Update the Supported Leagues table in README.md and the known list above

## Notification payload shape

```go
type notificationPayload struct {
    Game    gameResult `json:"game"`
    Summary string     `json:"summary"`
    Winner  *string    `json:"winner"` // nil on draw
    Loser   *string    `json:"loser"`  // nil on draw
    IsDraw  bool       `json:"isDraw"`
}
```

## Development workflow

```bash
go build ./...              # compile
go vet ./...                # static analysis
go build -o game-over-man . && CONFIG_FILE=config.json NOTIFICATION_URL=http://localhost:3001 ./game-over-man
```

To test locally without a real webhook, run a listener in another terminal:

```bash
python3 -m http.server 3001
```

## Versioning

Releases use CalVer: `YYYY.MM.DD.N` where N is `github.run_number`. The version is embedded at build time via `-ldflags "-X main.version=..."` into the `version` variable in `main.go`. Running from source prints `dev`. No manual tagging; every push to main creates a release.

## GitHub Actions

`.github/workflows/publish.yml` triggers on every push to `main`. It computes the CalVer string, cross-compiles for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and windows/amd64 (with the version embedded via ldflags), then creates a GitHub Release tagged with the CalVer string using `softprops/action-gh-release` with `make_latest: true`.

## Scheduling

Two scheduling options are documented in README.md:
- **systemd timer** -- recommended; dedicated user; logs via `journalctl`; deploy files in `deploy/systemd/`
- **cron** -- simpler; one crontab line; logs go to syslog

When changing the default schedule (currently every 10 minutes), update `OnCalendar` in `deploy/systemd/game-over-man.timer` and the examples in README.md.

## Style conventions

- Standard Go idioms; run `go vet` before committing
- No external dependencies -- standard library only
- Log lines are prefixed with `[module]` (e.g. `[espn]`, `[notify]`, `[state]`, `[config]`)
- Do not write comments that explain what the code does -- only add one when the WHY is non-obvious
- No em dashes anywhere in code or documentation

## Files to keep updated

When making changes, keep README.md and AGENTS.md in sync:
- New leagues -> Supported Leagues table in README.md and known list in AGENTS.md
- New config fields -> Config fields table in README.md and this file
- New env vars -> Environment Variables table in README.md and this file
- Architectural changes -> Key design decisions section in this file
