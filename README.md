# Game Over Man

A sports score notifier. It polls the ESPN API for final scores across multiple sports and leagues, then fires a single webhook notification per game for each team you care about. No score goes unnoticed; no notification repeats.

It is a single Go binary with no runtime dependencies. You can run it directly on any Linux or macOS machine, schedule it with cron or systemd, or run it in Docker if you prefer.

## Features

- Tracks teams across NFL, NHL, NBA, MLB, AHL, MLS, college football, college basketball, and more
- One notification per completed game -- no duplicates, even across restarts
- Webhook URL configurable via environment variable or config file
- Custom HTTP headers supported (for auth tokens, Slack/Discord format requirements, etc.)
- Persistent state in a plain JSON file; old entries are pruned automatically
- Single static binary, no runtime dependencies

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
```

## Configuration

Create a config file (default location: `/etc/game-over-man/config.json`):

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

**Notification URL:** Set via the `NOTIFICATION_URL` environment variable (preferred, keeps it out of the config file) or directly in the config as `notificationUrl`. The env var takes precedence.

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

Point `notificationUrl` at your topic URL (e.g. `https://ntfy.sh/my-sports-alerts`). The full JSON payload will be the body. To show just the `summary` as a plain-text notification, you can run a small proxy or use ntfy's [message templating](https://docs.ntfy.sh).

### Discord / Slack

Use the webhook URL as `notificationUrl`. Discord expects a `content` field and Slack expects `text`; a small proxy or [n8n](https://n8n.io/) works well for reshaping the payload.

## Scheduling

The binary is a one-shot job: it runs, checks scores, and exits. You schedule it with whatever fits your setup.

### cron

The quickest option. Add a line to your crontab (`crontab -e`):

```cron
*/10 * * * * NOTIFICATION_URL=https://ntfy.sh/my-sports-alerts /usr/local/bin/game-over-man
```

Or if you prefer an env file:

```cron
*/10 * * * * env $(cat /etc/game-over-man/env | xargs) /usr/local/bin/game-over-man
```

### systemd timer (recommended for Linux servers)

systemd timers have proper log capture via `journalctl`, survive reboots cleanly with `Persistent=true`, and run as a dedicated non-root user. Ready-to-use files are in `deploy/systemd/`.

**Quick install:**

```bash
sudo bash deploy/systemd/install.sh
```

The script downloads the latest binary, creates a `game-over-man` system user, sets up `/etc/game-over-man/` and `/var/lib/game-over-man/`, and enables the timer. Then:

```bash
# Edit the notification URL
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

### Docker Compose + ofelia

[ofelia](https://github.com/mcuadros/ofelia) is a lightweight job scheduler for Docker. Use this if you're already running Docker Compose.

Copy the files from `deploy/compose/` and edit `ofelia.ini` with your paths and notification URL:

```bash
cp deploy/compose/docker-compose.yml deploy/compose/ofelia.ini ./
# Edit ofelia.ini, then:
docker compose up -d
```

Logs: `docker compose logs -f ofelia`

### Docker (one-shot)

```bash
docker run --rm \
  -v /path/to/config.json:/config/config.json:ro \
  -v /path/to/data:/data \
  -e NOTIFICATION_URL=https://ntfy.sh/my-sports-alerts \
  ghcr.io/rorpage/game-over-man:latest
```

The Docker image sets `CONFIG_FILE=/config/config.json` and `STATE_FILE=/data/state.json` as defaults so the mount points are predictable.

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: game-over-man
spec:
  schedule: "*/10 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: game-over-man
              image: ghcr.io/rorpage/game-over-man:latest
              env:
                - name: NOTIFICATION_URL
                  valueFrom:
                    secretKeyRef:
                      name: game-over-man
                      key: notificationUrl
              volumeMounts:
                - name: config
                  mountPath: /config
                  readOnly: true
                - name: state
                  mountPath: /data
          volumes:
            - name: config
              configMap:
                name: game-over-man-config
            - name: state
              persistentVolumeClaim:
                claimName: game-over-man-state
```

## How It Works

1. Load config from `CONFIG_FILE` (default: `/etc/game-over-man/config.json`)
2. Load notification state from `STATE_FILE`, pruning entries older than `pruneAfterDays`
3. For each unique sport/league in the team list, fetch today's scoreboard from the ESPN API
4. For each completed game involving a tracked team, check whether a notification was already sent
5. If not, POST the notification payload to the configured URL and record the game ID in state
6. Save state to disk

The state file is the single source of truth for idempotency. As long as it persists across runs, no game will trigger more than one notification.

## Publishing a New Version

Create a tag to trigger the GitHub Actions workflow, which builds binaries for all platforms and attaches them to a GitHub Release, and also builds and pushes a Docker image:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Binaries will be attached to the release at `https://github.com/rorpage/game-over-man/releases`.
The Docker image will be published to `ghcr.io/rorpage/game-over-man`.

To make the published Docker image publicly pullable, go to the package settings on GitHub and set visibility to Public.
