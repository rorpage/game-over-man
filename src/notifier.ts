import { GameResult, NotificationConfig, NotificationPayload } from './types';

function buildPayload(game: GameResult): NotificationPayload {
  const { homeTeam, awayTeam, statusDescription } = game;
  const isDraw = homeTeam.score === awayTeam.score;

  let winner: string | null = null;
  let loser: string | null = null;
  let summary: string;

  if (isDraw) {
    summary = `Final: ${awayTeam.name} ${awayTeam.score}, ${homeTeam.name} ${homeTeam.score} -- Draw (${statusDescription})`;
  } else {
    const w = homeTeam.score > awayTeam.score ? homeTeam : awayTeam;
    const l = homeTeam.score > awayTeam.score ? awayTeam : homeTeam;
    winner = w.name;
    loser = l.name;
    summary = `Final: ${w.name} ${w.score}, ${l.name} ${l.score} (${statusDescription})`;
  }

  return { game, summary, winner, loser, isDraw };
}

export async function sendNotification(
  config: NotificationConfig,
  game: GameResult
): Promise<boolean> {
  const payload = buildPayload(game);

  try {
    const response = await fetch(config.url, {
      method: config.method ?? 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...config.headers,
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      console.error(
        `[notify] HTTP ${response.status} for game ${game.id} (${game.sport}/${game.league})`
      );
      return false;
    }

    console.log(`[notify] Sent: ${payload.summary}`);
    return true;
  } catch (err) {
    console.error(`[notify] Request failed for game ${game.id}: ${String(err)}`);
    return false;
  }
}
