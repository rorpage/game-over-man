# AGENTS.md

Context for AI agents (Claude Code, Copilot, etc.) working in this repository.

## What this project does

Game Over Man is a one-shot TypeScript app that runs in a Docker container. It queries the ESPN public scoreboard API for any sport/league you configure, finds completed games involving tracked teams, and sends a single webhook notification per game. Idempotency is maintained via a JSON state file on a mounted volume.

## Repository layout

```
src/
  index.ts      -- entry point; orchestrates config, ESPN fetch, notify, state
  config.ts     -- loads and validates config.json + env vars
  espn.ts       -- ESPN API fetch and parsing
  notifier.ts   -- builds payload and POSTs to webhook URL
  state.ts      -- reads/writes/prunes the state file
  types.ts      -- shared TypeScript interfaces

config.example.json   -- copy to config.json (gitignored) to run locally
Dockerfile            -- multi-stage Alpine build; no runtime npm deps
.github/workflows/
  publish.yml         -- builds and pushes to ghcr.io/rorpage/game-over-man on push to main or tag
```

## Key design decisions

- **No runtime npm dependencies.** Node 20's built-in `fetch` handles all HTTP. The dist folder is the complete runtime artifact.
- **One-shot execution.** The container runs, checks scores, and exits. Scheduling (cron, k8s CronJob, etc.) is the caller's responsibility.
- **State file for idempotency.** `state.json` records notified game IDs with timestamps. Entries older than `pruneAfterDays` (default 30) are removed on each run. If a notification POST fails, the game is not recorded, so it will be retried on the next run.
- **Config-file-first, env-var override.** `NOTIFICATION_URL`, `CONFIG_FILE`, and `STATE_FILE` env vars override their config file equivalents. Keep the notification URL in an env var to avoid committing secrets.
- **Case-normalized inputs.** Sport and league values are lowercased; abbreviations are uppercased during config load so comparisons are always case-insensitive.

## ESPN API

Base URL: `http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard`

This is an unofficial but stable ESPN endpoint. It returns today's games with scores and status. The `status.type.completed` boolean determines whether a game is final. `status.type.description` carries strings like `"Final"`, `"Final/OT"`, `"Final/SO"`.

Known working sport/league pairs: `football/nfl`, `football/college-football`, `basketball/nba`, `basketball/wnba`, `basketball/mens-college-basketball`, `basketball/womens-college-basketball`, `baseball/mlb`, `hockey/nhl`, `hockey/ahl`, `soccer/usa.1`, `soccer/usa.nwsl`, `soccer/eng.1`, `soccer/esp.1`, `soccer/ita.1`, `soccer/ger.1`, `soccer/fra.1`, `soccer/uefa.champions`.

## Adding a new league

1. Verify the ESPN endpoint returns data: `curl "http://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard"`
2. Add entries to the `teams` array in config using the correct `sport` and `league` values
3. Update the "Supported Leagues" table in README.md

## Notification payload shape

```typescript
{
  game: GameResult;       // full game details including scores and teams
  summary: string;        // human-readable one-liner, e.g. "Final: UTA 4, COL 3 (Final/OT)"
  winner: string | null;  // null on draw
  loser: string | null;   // null on draw
  isDraw: boolean;
}
```

## Development workflow

```bash
npm install
npm run build        # compiles src/ -> dist/
npm run typecheck    # tsc --noEmit, no output files
node dist/index.js   # needs CONFIG_FILE and NOTIFICATION_URL set
```

To test locally without a real webhook, use [httpbin](https://httpbin.org/post) or run a local listener:

```bash
npx -y http-echo-server 3001 &
CONFIG_FILE=config.json NOTIFICATION_URL=http://localhost:3001 node dist/index.js
```

## Docker

```bash
docker build -t game-over-man .
docker run --rm \
  -v $(pwd)/config.json:/config/config.json:ro \
  -v $(pwd)/data:/data \
  -e NOTIFICATION_URL=https://ntfy.sh/my-topic \
  game-over-man
```

The `/data` volume must be persistent across runs for idempotency to work.

## GitHub Actions

`.github/workflows/publish.yml` triggers on push to `main` or any `v*` tag. It logs into `ghcr.io` using `GITHUB_TOKEN` (no extra secrets needed), builds the Docker image, and pushes with tags: `latest` (main only), branch name, version tag, and `sha-<shortsha>`. GHA cache is used to speed up repeated builds.

To make the published image publicly pullable, go to the package settings on GitHub and set visibility to Public.

## Style conventions

- TypeScript strict mode is on; avoid `any`
- No runtime dependencies -- if you need HTTP, use built-in `fetch`; if you need file I/O, use built-in `fs`
- Log lines are prefixed with `[module]` (e.g. `[espn]`, `[notify]`, `[state]`, `[config]`)
- Do not add error handling for scenarios that cannot occur; trust TypeScript types
- Do not write comments that explain what the code does -- only write them when the WHY is non-obvious
- No em dashes anywhere in code or documentation

## Files to keep updated

When making changes, keep README.md and AGENTS.md in sync:
- New leagues -> update the Supported Leagues table in README.md and the known list in AGENTS.md
- New config fields -> update Config fields table in README.md and this file
- New env vars -> update Environment Variables table in README.md and this file
- Architectural changes -> update the Key design decisions section in this file
