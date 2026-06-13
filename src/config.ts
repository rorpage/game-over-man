import * as fs from 'fs';
import { AppConfig } from './types';

const DEFAULT_STATE_FILE = '/data/state.json';
const DEFAULT_PRUNE_DAYS = 30;
const DEFAULT_CONFIG_PATH = '/config/config.json';

interface RawConfig {
  teams: Array<{
    sport: string;
    league: string;
    abbreviation: string;
  }>;
  notificationUrl?: string;
  notificationMethod?: string;
  notificationHeaders?: Record<string, string>;
  stateFilePath?: string;
  pruneAfterDays?: number;
}

export function resolveConfigPath(): string {
  return process.env.CONFIG_FILE ?? DEFAULT_CONFIG_PATH;
}

export function loadConfig(configPath: string): AppConfig {
  if (!fs.existsSync(configPath)) {
    throw new Error(
      `Config file not found at ${configPath}. ` +
        'Set CONFIG_FILE env var or mount your config at /config/config.json.'
    );
  }

  let raw: RawConfig;
  try {
    raw = JSON.parse(fs.readFileSync(configPath, 'utf-8')) as RawConfig;
  } catch (err) {
    throw new Error(`Failed to parse config file at ${configPath}: ${String(err)}`);
  }

  const notificationUrl = process.env.NOTIFICATION_URL ?? raw.notificationUrl;
  if (!notificationUrl) {
    throw new Error(
      'Notification URL is required. Set NOTIFICATION_URL env var or notificationUrl in config.'
    );
  }

  if (!Array.isArray(raw.teams) || raw.teams.length === 0) {
    throw new Error('Config must include at least one team in the "teams" array.');
  }

  for (const team of raw.teams) {
    if (!team.sport || !team.league || !team.abbreviation) {
      throw new Error(
        `Each team entry must have "sport", "league", and "abbreviation". Got: ${JSON.stringify(team)}`
      );
    }
  }

  return {
    teams: raw.teams.map(t => ({
      sport: t.sport.toLowerCase(),
      league: t.league.toLowerCase(),
      abbreviation: t.abbreviation.toUpperCase(),
    })),
    notification: {
      url: notificationUrl,
      method: raw.notificationMethod,
      headers: raw.notificationHeaders,
    },
    stateFilePath: process.env.STATE_FILE ?? raw.stateFilePath ?? DEFAULT_STATE_FILE,
    pruneAfterDays: raw.pruneAfterDays ?? DEFAULT_PRUNE_DAYS,
  };
}
