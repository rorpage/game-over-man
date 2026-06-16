# AGENTS.md

Context for AI agents (Claude Code, Copilot, etc.) working in this repository.

## What this project does

Game Over Man is a one-shot Go binary for home servers. It queries ESPN or HockeyTech (selected automatically by league) for any sport/league you configure, finds completed games involving tracked teams, and sends a single webhook notification per game. Idempotency is maintained via a JSON state file. It runs directly on Linux or macOS -- no Docker, no runtime dependencies.

## Repository layout

```
main.go          -- entry point; orchestrates config, fetch (ESPN or HockeyTech), notify, state
config.go        -- config types, loading from JSON + env var overrides
espn.go          -- ESPN API fetch, response parsing, team matching
hockeytech.go    -- HockeyTech API fetch and response parsing (PWHL, ECHL, and others)
notifier.go      -- builds notification payload (webhook/slack/discord/template), POSTs to webhook URL
state.go         -- reads/writes/prunes the state file

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
- **Notification type presets.** `notificationType` controls the outgoing payload shape: `webhook` (default, full JSON), `slack` (`{"text": "..."}`), `discord` (`{"content": "..."}`), or `template` (arbitrary Go `text/template` string rendered against the full payload). This lets you target common platforms without an intermediary.
- **Wildcard and postseason-only entries.** Setting `abbreviation` to `"*"` matches every team in that sport/league. Setting `postseasonOnly: true` suppresses notifications for regular-season games. Combine them to follow playoffs league-wide without knowing the teams in advance. Postseason detection uses the top-level `season.type` field from the ESPN scoreboard response (3 = postseason).

## Config fields

| Field | Required | Default | Description |
|---|---|---|---|
| `teams` | Yes | -- | Array of teams to track |
| `teams[].sport` | Yes | -- | Sport category (e.g. `hockey`, `football`) |
| `teams[].league` | Yes | -- | League identifier (e.g. `nhl`, `nfl`) |
| `teams[].abbreviation` | Yes | -- | Team abbreviation as used by the data provider (e.g. `CHI`, `IND`), or `"*"` to match every team in the league. Use `"*"` first to discover abbreviations from notification payloads if unsure. |
| `teams[].postseasonOnly` | No | `false` | When `true`, skip games that are not part of the postseason/playoffs |
| `notificationUrl` | See note | -- | Webhook URL to POST alerts to |
| `notificationMethod` | No | `POST` | HTTP method for notifications |
| `notificationHeaders` | No | -- | Extra headers (e.g. `{"Authorization": "******"}`) |
| `notificationType` | No | `webhook` | Payload format: `webhook`, `slack`, `discord`, or `template` |
| `notificationTemplate` | If `template` | -- | Go template string used when `notificationType` is `template` |
| `stateFilePath` | No | `/var/lib/game-over-man/state.json` | Where to persist notification state |
| `pruneAfterDays` | No | `30` | How many days to keep state entries before pruning |

## Default file paths

| Purpose | Default |
|---|---|
| Config | `/etc/game-over-man/config.json` |
| State | `/var/lib/game-over-man/state.json` |

## Shared utilities

`parseScore` is defined in `espn.go` and used by `hockeytech.go`. This is intentional -- all files are in the same `main` package and share a flat namespace, so there is no real coupling between files. Do not extract single small helpers into a `util.go` preemptively; only do so if several shared helpers accumulate and a dedicated file becomes clearly worthwhile.

## Provider selection

The API provider is selected automatically based on league name. `isHockeytechLeague` in `hockeytech.go` checks the `hockeytechLeagues` registry. If the league is not in that registry, ESPN is used. No config field is needed.

## ESPN API

Base URL: `http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard`

This is an unofficial but stable ESPN endpoint. It returns today's games with scores and status. The `status.type.completed` boolean determines whether a game is final. `status.type.description` carries strings like `"Final"`, `"Final/OT"`, `"Final/SO"`. The top-level `season.type` integer indicates the season phase: `2` = regular season, `3` = postseason. This is used to populate `gameResult.IsPostseason` and evaluate `postseasonOnly` config entries.

Known working sport/league pairs: `football/nfl`, `football/college-football`, `basketball/nba`, `basketball/wnba`, `basketball/mens-college-basketball`, `basketball/womens-college-basketball`, `baseball/mlb`, `hockey/nhl`, `hockey/ahl`, `soccer/usa.1`, `soccer/usa.nwsl`, `soccer/eng.1`, `soccer/esp.1`, `soccer/ita.1`, `soccer/ger.1`, `soccer/fra.1`, `soccer/uefa.champions`, `soccer/fifa.world`.

## HockeyTech API

Base URL: `https://lscluster.hockeytech.com/feed/index.php`

Key query parameters: `key`, `client_code`, `feed=modulekit`, `view=scorebar`, `numberofdaysahead=0`, `numberofdaysback=1`, `lang=en`, `fmt=json`.

Response shape: `SiteKit.Scorebar[]` array. `GameStatus` "4" = final, "5" = unofficial final. `IsPlayoffGame` "1" = postseason. Game IDs are prefixed with the league name in the state file (e.g. `pwhl_12345`) to avoid collisions across leagues.

Known leagues (in `hockeytechLeagues` map in `hockeytech.go`):

| League | `league` value | `client_code` | `key` |
|---|---|---|---|
| PWHL | `pwhl` | `pwhl` | `694cfeed58c932ee` |
| ECHL | `echl` | `echl` | `e6219ee34f4b5200` |

To add more HockeyTech leagues, add an entry to `hockeytechLeagues` in `hockeytech.go`. The key and client_code for any HockeyTech-powered league are embedded in that league's official website/app JavaScript.

## Adding a new ESPN league

1. Verify the endpoint: `curl "http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard"`
2. Add entries to the `teams` array in config
3. Update the Supported Leagues table in README.md and the known list above

## Adding a new HockeyTech league

1. Find the `key` and `client_code` from the league's official website JS
2. Add an entry to `hockeytechLeagues` in `hockeytech.go`
3. Update the Supported Leagues table in README.md and the HockeyTech known leagues table above

## Notification payload shape

```go
type gameResult struct {
    ID                string     `json:"id"`
    Sport             string     `json:"sport"`
    League            string     `json:"league"`
    Date              string     `json:"date"`
    HomeTeam          competitor `json:"homeTeam"`
    AwayTeam          competitor `json:"awayTeam"`
    StatusDescription string     `json:"statusDescription"`
    IsPostseason      bool       `json:"isPostseason"`
}

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

## Working with the developer

Push back when a suggestion would make the code worse -- don't just agree. If the developer proposes something that adds unnecessary complexity, breaks a design principle, or has a better alternative, say so directly and explain why. The goal is a dialog, not compliance. It is fine (expected, even) to advocate for a different approach before implementing anything.

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
