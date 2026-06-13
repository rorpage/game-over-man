# Game Over Man

A sports score notifier for home servers. It polls the ESPN API for final scores across multiple sports and leagues, then fires a single webhook notification per game for each team you care about. No score goes unnoticed; no notification repeats.

It is a single Go binary with no runtime dependencies. Drop it on any Linux or macOS machine, point it at a config file, and schedule it with cron or systemd.

## Features

- Tracks teams across NFL, NHL, NBA, MLB, AHL, MLS, college football, college basketball, and more
- One notification per completed game -- no duplicates, even across restarts
- Webhook URL configurable via environment variable or config file
- Custom HTTP headers supported (for auth tokens, Slack/Discord format requirements, etc.)
- Persistent state in a plain JSON file; old entries are pruned automatically
- Single static binary, no runtime dependencies, no Docker required

## Installation

### Download a pre-built binary (recommended)

Download the appropriate binary for your platform from the [Releases](https://github.com/rorpage/game-over-man/releases) page:

| Platform | File |
|---|---|
| Linux x86-64 | `game-over-man-linux-amd64` |
| Linux ARM64 (Raspberry Pi, etc.) | `game-over-man-linux-arm64` |
| macOS Intel | `game-over-man-darwin-amd64` |
| macOS Apple Silicon | `game-over-man-darwin-arm64` |
| Windows x86-64 | `game-over-man-windows-amd64.exe` |

```bash
# Example: Linux x86-64
curl -fsSL https://github.com/rorpage/game-over-man/releases/latest/download/game-over-man-linux-amd64 \
  -o /usr/local/bin/game-over-man
chmod +x /usr/local/bin/game-over-man
```

### Build from source

Requires Go 1.22+.

```bash
git clone https://github.com/rorpage/game-over-man.git
cd game-over-man
CGO_ENABLED=0 go build -ldflags="-s -w" -o game-over-man .
sudo mv game-over-man /usr/local/bin/
```

## Configuration

Create a config file at `/etc/game-over-man/config.json`:

```bash
sudo mkdir -p /etc/game-over-man
sudo nano /etc/game-over-man/config.json
```

```json
{
  "teams": [
    { "sport": "hockey",   "league": "nhl", "abbreviation": "UTA" },
    { "sport": "hockey",   "league": "ahl", "abbreviation": "TUC" },
    { "sport": "football", "league": "nfl", "abbreviation": "KC"  }
  ]
}
```

See `config.example.json` for a more complete example with all supported fields.

### Config fields

| Field | Required | Default | Description |
|---|---|---|---|
| `teams` | Yes | -- | Array of teams to track |
| `teams[].sport` | Yes | -- | Sport category (e.g. `hockey`, `football`) |
| `teams[].league` | Yes | -- | League identifier (e.g. `nhl`, `nfl`) |
| `teams[].abbreviation` | Yes | -- | Team abbreviation as used by ESPN (e.g. `UTA`, `KC`) |
| `notificationUrl` | See note | -- | Webhook URL to POST alerts to |
| `notificationMethod` | No | `POST` | HTTP method for notifications |
| `notificationHeaders` | No | -- | Extra headers (e.g. `{"Authorization": "Bearer ..."}`) |
| `stateFilePath` | No | `/var/lib/game-over-man/state.json` | Where to persist notification state |
| `pruneAfterDays` | No | `30` | How many days to keep state entries before pruning |

**Notification URL:** Set via the `NOTIFICATION_URL` environment variable (preferred, keeps it out of the config file) or as `notificationUrl` in the config. The env var takes precedence.

## Environment Variables

| Variable | Description |
|---|---|
| `NOTIFICATION_URL` | Webhook URL (overrides `notificationUrl` in config) |
| `CONFIG_FILE` | Path to config file (default: `/etc/game-over-man/config.json`) |
| `STATE_FILE` | Path to state file (default: `/var/lib/game-over-man/state.json`) |

## Supported Leagues

| Sport | League | `sport` value | `league` value |
|---|---|---|---|
| Football | NFL | `football` | `nfl` |
| Football | College Football | `football` | `college-football` |
| Basketball | NBA | `basketball` | `nba` |
| Basketball | WNBA | `basketball` | `wnba` |
| Basketball | Men's NCAA | `basketball` | `mens-college-basketball` |
| Basketball | Women's NCAA | `basketball` | `womens-college-basketball` |
| Baseball | MLB | `baseball` | `mlb` |
| Hockey | NHL | `hockey` | `nhl` |
| Hockey | AHL | `hockey` | `ahl` |
| Soccer | MLS | `soccer` | `usa.1` |
| Soccer | NWSL | `soccer` | `usa.nwsl` |
| Soccer | Premier League | `soccer` | `eng.1` |
| Soccer | La Liga | `soccer` | `esp.1` |
| Soccer | Serie A | `soccer` | `ita.1` |
| Soccer | Bundesliga | `soccer` | `ger.1` |
| Soccer | Ligue 1 | `soccer` | `fra.1` |
| Soccer | Champions League | `soccer` | `uefa.champions` |

The ESPN API may support additional leagues. Test any `sport`/`league` pair with:

```bash
curl "http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard"
```

## Notification Payload

Each alert is an HTTP POST with `Content-Type: application/json`:

```json
{
  "game": {
    "id": "401589012",
    "sport": "hockey",
    "league": "nhl",
    "date": "2025-04-10T02:00:00Z",
    "homeTeam": { "name": "Utah Hockey Club", "abbreviation": "UTA", "score": 4, "isHome": true },
    "awayTeam": { "name": "Colorado Avalanche", "abbreviation": "COL", "score": 3, "isHome": false },
    "statusDescription": "Final/OT"
  },
  "summary": "Final: Utah Hockey Club 4, Colorado Avalanche 3 (Final/OT)",
  "winner": "Utah Hockey Club",
  "loser": "Colorado Avalanche",
  "isDraw": false
}
```

`winner` and `loser` are `null` when the game ends in a draw.

### ntfy.sh

Point `notificationUrl` at your topic URL (e.g. `https://ntfy.sh/my-sports-alerts`). The full JSON payload will be the body. To show just the `summary` as a plain-text push notification, use ntfy's [message templating](https://docs.ntfy.sh) or run a small proxy.

### Discord / Slack

Use the webhook URL as `notificationUrl`. Discord expects a `content` field and Slack expects `text`; a tool like [n8n](https://n8n.io/) works well for reshaping the payload.

## Scheduling

The binary is a one-shot job: it runs, checks scores, and exits.

### systemd timer (recommended)

systemd timers have proper log capture via `journalctl`, survive reboots cleanly with `Persistent=true`, run as a dedicated non-root user, and are standard on any modern Linux distro. Ready-to-use files are in `deploy/systemd/`.

**Quick install:**

```bash
sudo bash deploy/systemd/install.sh
```

The script downloads the latest binary from GitHub Releases (or builds from source if Go is available), creates a `game-over-man` system user, sets up `/etc/game-over-man/` and `/var/lib/game-over-man/`, and enables the timer. Then:

```bash
# Set your notification URL
sudo nano /etc/game-over-man/env

# Copy your config
sudo cp config.json /etc/game-over-man/config.json
sudo chown root:game-over-man /etc/game-over-man/config.json

# Start
sudo systemctl start game-over-man.timer
```

**Useful commands:**

```bash
systemctl status game-over-man.timer     # next scheduled run
systemctl start game-over-man.service    # run immediately
journalctl -u game-over-man -f           # follow logs
```

**Changing the schedule:** Edit `/etc/systemd/system/game-over-man.timer`, update `OnCalendar`, then:

```bash
sudo systemctl daemon-reload && sudo systemctl restart game-over-man.timer
```

Common values:

```ini
OnCalendar=*:0/10    # every 10 minutes (default)
OnCalendar=*:0/5     # every 5 minutes
OnCalendar=hourly    # once per hour
```

### cron

Add a line to your crontab with `crontab -e`:

```cron
*/10 * * * * NOTIFICATION_URL=https://ntfy.sh/my-sports-alerts /usr/local/bin/game-over-man
```

Or if you prefer an env file:

```cron
*/10 * * * * env $(cat /etc/game-over-man/env | xargs) /usr/local/bin/game-over-man
```

Logs go to syslog (`journalctl -t game-over-man` or `/var/log/syslog`).

## How It Works

1. Load config from `CONFIG_FILE` (default: `/etc/game-over-man/config.json`)
2. Load notification state from `STATE_FILE`, pruning entries older than `pruneAfterDays`
3. For each unique sport/league in the team list, fetch today's scoreboard from the ESPN API
4. For each completed game involving a tracked team, check whether a notification was already sent
5. If not, POST the notification payload to the configured URL and record the game ID in state
6. Save state to disk

The state file is the single source of truth for idempotency. As long as it persists across runs, no game will ever trigger more than one notification.

## Versioning

Releases are created automatically on every push to `main`. The version follows CalVer: `YYYY.MM.DD.N` where N is the GitHub Actions run number (e.g. `2026.06.13.4`). No manual tagging required.

The version is embedded in the binary at build time:

```bash
game-over-man --version
# 2026.06.13.4
```

When running locally from source, `--version` prints `dev`.

The latest release is always available at `https://github.com/rorpage/game-over-man/releases/latest`.
