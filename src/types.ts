export interface TeamConfig {
  sport: string;
  league: string;
  abbreviation: string;
}

export interface NotificationConfig {
  url: string;
  method?: string;
  headers?: Record<string, string>;
}

export interface AppConfig {
  teams: TeamConfig[];
  notification: NotificationConfig;
  stateFilePath: string;
  pruneAfterDays: number;
}

export interface Competitor {
  name: string;
  abbreviation: string;
  score: number;
  isHome: boolean;
}

export interface GameResult {
  id: string;
  sport: string;
  league: string;
  date: string;
  homeTeam: Competitor;
  awayTeam: Competitor;
  statusDescription: string;
}

export interface NotificationPayload {
  game: GameResult;
  summary: string;
  winner: string | null;
  loser: string | null;
  isDraw: boolean;
}

export interface StateEntry {
  gameId: string;
  notifiedAt: string;
}

export interface AppState {
  notifiedGames: StateEntry[];
}
