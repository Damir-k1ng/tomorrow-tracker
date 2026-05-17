import type { Leaderboard, RawLeaderboard } from './types';

/** Map the backend leaderboard DTO (snake_case) into the app's `Leaderboard`. */
export function mapLeaderboard(raw: RawLeaderboard): Leaderboard {
  return {
    top: raw.top.map((e) => ({
      rank: e.rank,
      userId: e.user_id,
      name: e.name,
      minutes: e.minutes,
      isCurrent: e.is_current,
    })),
    me: {
      found: raw.me.found,
      rank: raw.me.rank,
      minutes: raw.me.minutes,
      name: raw.me.name,
      inTop: raw.me.in_top,
    },
  };
}
