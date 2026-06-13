import { GameResult, TeamConfig } from './types';

const ESPN_BASE = 'http://site.api.espn.com/apis/site/v2/sports';

interface EspnTeam {
  abbreviation: string;
  displayName: string;
}

interface EspnCompetitor {
  homeAway: 'home' | 'away';
  team: EspnTeam;
  score?: string;
}

interface EspnStatusType {
  completed: boolean;
  description: string;
}

interface EspnEvent {
  id: string;
  date: string;
  status: {
    type: EspnStatusType;
  };
  competitions: Array<{
    competitors: EspnCompetitor[];
  }>;
}

interface EspnScoreboard {
  events?: EspnEvent[];
}

export async function fetchScoreboard(sport: string, league: string): Promise<GameResult[]> {
  const url = `${ESPN_BASE}/${sport}/${league}/scoreboard`;

  let data: EspnScoreboard;
  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status} ${response.statusText}`);
    }
    data = (await response.json()) as EspnScoreboard;
  } catch (err) {
    console.error(`[espn] Failed to fetch ${sport}/${league}: ${String(err)}`);
    return [];
  }

  const events = data.events ?? [];

  return events
    .filter(e => e.status.type.completed)
    .flatMap(e => {
      const game = parseEvent(e, sport, league);
      return game ? [game] : [];
    });
}

function parseEvent(event: EspnEvent, sport: string, league: string): GameResult | null {
  const competition = event.competitions[0];
  if (!competition) return null;

  const home = competition.competitors.find(c => c.homeAway === 'home');
  const away = competition.competitors.find(c => c.homeAway === 'away');
  if (!home || !away) return null;

  return {
    id: event.id,
    sport,
    league,
    date: event.date,
    homeTeam: {
      name: home.team.displayName,
      abbreviation: home.team.abbreviation.toUpperCase(),
      score: parseInt(home.score ?? '0', 10),
      isHome: true,
    },
    awayTeam: {
      name: away.team.displayName,
      abbreviation: away.team.abbreviation.toUpperCase(),
      score: parseInt(away.score ?? '0', 10),
      isHome: false,
    },
    statusDescription: event.status.type.description,
  };
}

export function isTrackedGame(game: GameResult, teams: TeamConfig[]): boolean {
  const leagueTeams = teams.filter(
    t => t.sport === game.sport && t.league === game.league
  );
  const abbrs = new Set(leagueTeams.map(t => t.abbreviation));
  return abbrs.has(game.homeTeam.abbreviation) || abbrs.has(game.awayTeam.abbreviation);
}
