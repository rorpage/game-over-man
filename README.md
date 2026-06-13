# Game Over Man

A containerized sports score notifier. It polls the ESPN API for final scores across multiple sports and leagues, then fires a single webhook notification per game for each team you care about. No score goes unnoticed; no notification repeats.

## Features

- Tracks teams across NFL, NHL, NBA, MLB, AHL, MLS, college football, college basketball, and more
- One notification per completed game -- no duplicates, even across container restarts
- Webhook URL configurable via environment variable or config file
- Custom HTTP headers supported (for auth tokens, Slack/Discord format requirements, etc.)
- Persistent state via a mounted volume; old entries are pruned automatically
- Tiny Alpine-based image with no runtime npm dependencies

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

The ESPN API may support additional leagues. Use the `sport`/`league` path segments from `http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard` to add more.

## Configuration

Copy `config.example.json` to `config.json` (which is gitignored) and edit it:

```json
{
  "teams": [
    { "sport": "hockey",   "league": "nhl", "abbreviation": "UTA" },
    { "sport": "hockey",   "league": "ahl", "abbreviation": "TUC" },
    { "sport": "football", "league": "nfl", "abbreviation": "KC"  }
  ],
  "notificationUrl": "https://ntfy.sh/my-sports-alerts",
  "stateFilePath": "/data/state.json",
  "pruneAfterDays": 30
}
```

### Config fields

| Field | Required | Default | Description |
|---|---|---|---|
| `teams` | Yes | -- | Array of teams to track |
| `teams[].sport` | Yes | -- | Sport category (e.g. `hockey`, `football`) |
| `teams[].league` | Yes | -- | League identifier (e.g. `nhl`, `nfl`) |
| `teams[].abbreviation` | Yes | -- | Team abbreviation as used by ESPN (e.g. `UTA`, `KC`) |
| `notificationUrl` | See note | -- | Webhook URL to POST alerts to |
| `notificationMethod` | No | `POST` | HTTP method for notifications |
| `notificationHeaders` | No | -- | Extra headers to include (e.g. `Authorization`) |
| `stateFilePath` | No | `/data/state.json` | Where to persist notification state |
| `pruneAfterDays` | No | `30` | How many days to keep state entries before pruning |

**Notification URL note:** Set via the `NOTIFICATION_URL` environment variable (preferred, keeps it out of the config file) or directly in the config as `notificationUrl`. The env var takes precedence.

## Notification Payload

Each notification is an HTTP POST with `Content-Type: application/json`:

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

### ntfy.sh

For [ntfy.sh](https://ntfy.sh), you can point `notificationUrl` directly at your topic URL. The notification body will be the full JSON payload; use ntfy's click/action features or a small proxy to reformat the `summary` field as plain text if needed.

### Discord

Use a Discord webhook URL as `notificationUrl` and add a `notificationHeaders` entry. Discord expects `content` as the message field; a lightweight proxy or a tool like [n8n](https://n8n.io/) works well for reshaping the payload.

## Running with Docker

### One-shot (recommended with cron or a k8s CronJob)

```bash
docker run --rm \
  -v /path/to/config.json:/config/config.json:ro \
  -v /path/to/data:/data \
  -e NOTIFICATION_URL=https://ntfy.sh/my-sports-alerts \
  ghcr.io/rorpage/game-over-man:latest
```

### cron example (every 10 minutes)

```cron
*/10 * * * * docker run --rm -v /opt/gom/config.json:/config/config.json:ro -v /opt/gom/data:/data -e NOTIFICATION_URL=https://ntfy.sh/my-sports-alerts ghcr.io/rorpage/game-over-man:latest
```

### Docker Compose

```yaml
services:
  game-over-man:
    image: ghcr.io/rorpage/game-over-man:latest
    volumes:
      - ./config.json:/config/config.json:ro
      - gom-data:/data
    environment:
      - NOTIFICATION_URL=https://ntfy.sh/my-sports-alerts
    restart: "no"

volumes:
  gom-data:
```

Pair it with a cron-based restart or use a tool like [ofelia](https://github.com/mcuadros/ofelia) to schedule it.

## Environment Variables

| Variable | Description |
|---|---|
| `NOTIFICATION_URL` | Webhook URL (overrides `notificationUrl` in config) |
| `CONFIG_FILE` | Path to config file (default: `/config/config.json`) |
| `STATE_FILE` | Path to state file (default: `/data/state.json`) |

## Building Locally

```bash
npm install
npm run build
node dist/index.js
```

Or with Docker:

```bash
docker build -t game-over-man .
docker run --rm \
  -v $(pwd)/config.json:/config/config.json:ro \
  -v $(pwd)/data:/data \
  -e NOTIFICATION_URL=https://ntfy.sh/my-sports-alerts \
  game-over-man
```

## How It Works

1. Load config from `/config/config.json` (or `CONFIG_FILE`)
2. Load notification state from `/data/state.json` (or `STATE_FILE`), pruning old entries
3. For each unique sport/league in the team list, fetch today's scoreboard from the ESPN API
4. For each completed game involving a tracked team, check whether a notification was already sent
5. If not, POST the notification payload to the configured URL and record the game ID in state
6. Save updated state to disk

The state file is the single source of truth for idempotency. Mounting `/data` as a persistent volume ensures notifications survive container restarts.

## Publishing a New Version

Push to `main` or create a version tag to trigger the GitHub Actions workflow:

```bash
git tag v1.2.0
git push origin v1.2.0
```

The image will be published to `ghcr.io/rorpage/game-over-man` with `latest`, the branch name, and the tag.
