import { loadConfig, resolveConfigPath } from './config';
import { fetchScoreboard, isTrackedGame } from './espn';
import { sendNotification } from './notifier';
import { loadState, saveState, pruneState, hasBeenNotified, markNotified } from './state';

async function main(): Promise<void> {
  console.log('=== Game Over Man - Sports Score Notifier ===');

  const configPath = resolveConfigPath();
  console.log(`[config] Loading from ${configPath}`);
  const config = loadConfig(configPath);
  console.log(`[config] Tracking ${config.teams.length} team(s)`);

  let state = loadState(config.stateFilePath);
  state = pruneState(state, config.pruneAfterDays);

  const leagues = new Set(config.teams.map(t => `${t.sport}/${t.league}`));
  let notifiedCount = 0;

  for (const leagueKey of leagues) {
    const slashIndex = leagueKey.indexOf('/');
    const sport = leagueKey.slice(0, slashIndex);
    const league = leagueKey.slice(slashIndex + 1);

    console.log(`[espn] Fetching ${sport}/${league}...`);
    const games = await fetchScoreboard(sport, league);
    console.log(`[espn] ${games.length} completed game(s) found`);

    for (const game of games) {
      if (!isTrackedGame(game, config.teams)) continue;

      if (hasBeenNotified(state, game.id)) {
        console.log(`[state] Already notified for game ${game.id}, skipping`);
        continue;
      }

      const success = await sendNotification(config.notification, game);
      if (success) {
        markNotified(state, game.id);
        notifiedCount++;
      }
    }
  }

  saveState(config.stateFilePath, state);
  console.log(`=== Done. ${notifiedCount} new notification(s) sent. ===`);
}

main().catch(err => {
  console.error('[fatal]', err instanceof Error ? err.message : String(err));
  process.exit(1);
});
