/**
 * Weekly leaderboard domain types — the read-model behind
 * GET /api/v1/user/leaderboard.
 *
 * `Leaderboard` is the app-internal (camelCase) shape; the `Raw*` interfaces
 * are the snake_case DTOs returned by the Go backend. Mapping lives in
 * `mappers.ts`.
 */

/** One row of the weekly Top-N. */
export interface LeaderboardEntry {
  rank: number;
  userId: number;
  name: string;
  minutes: number;
  isCurrent: boolean; // true when this row is the requesting user
}

/** The caller's own standing this week. `found` is false when they have no
 *  recorded minutes in the current week. */
export interface LeaderboardPosition {
  found: boolean;
  rank: number;
  minutes: number;
  name: string;
  inTop: boolean;
}

/** The full leaderboard screen payload: the Top-N plus the caller's position. */
export interface Leaderboard {
  top: LeaderboardEntry[];
  me: LeaderboardPosition;
}

export interface RawLeaderboardEntry {
  rank: number;
  user_id: number;
  name: string;
  minutes: number;
  is_current: boolean;
}

export interface RawLeaderboardPosition {
  found: boolean;
  rank: number;
  minutes: number;
  name: string;
  in_top: boolean;
}

export interface RawLeaderboard {
  top: RawLeaderboardEntry[];
  me: RawLeaderboardPosition;
}
