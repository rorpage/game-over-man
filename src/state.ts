import * as fs from 'fs';
import * as path from 'path';
import { AppState, StateEntry } from './types';

export function loadState(stateFilePath: string): AppState {
  if (!fs.existsSync(stateFilePath)) {
    return { notifiedGames: [] };
  }

  try {
    return JSON.parse(fs.readFileSync(stateFilePath, 'utf-8')) as AppState;
  } catch {
    console.warn(`[state] Could not parse state file at ${stateFilePath}, starting fresh.`);
    return { notifiedGames: [] };
  }
}

export function saveState(stateFilePath: string, state: AppState): void {
  const dir = path.dirname(stateFilePath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
  fs.writeFileSync(stateFilePath, JSON.stringify(state, null, 2));
}

export function pruneState(state: AppState, pruneAfterDays: number): AppState {
  const cutoff = new Date();
  cutoff.setDate(cutoff.getDate() - pruneAfterDays);

  const before = state.notifiedGames.length;
  const pruned = state.notifiedGames.filter(
    (entry: StateEntry) => new Date(entry.notifiedAt) > cutoff
  );

  if (pruned.length < before) {
    console.log(`[state] Pruned ${before - pruned.length} old state entries (older than ${pruneAfterDays} days).`);
  }

  return { notifiedGames: pruned };
}

export function hasBeenNotified(state: AppState, gameId: string): boolean {
  return state.notifiedGames.some((entry: StateEntry) => entry.gameId === gameId);
}

export function markNotified(state: AppState, gameId: string): void {
  state.notifiedGames.push({ gameId, notifiedAt: new Date().toISOString() });
}
