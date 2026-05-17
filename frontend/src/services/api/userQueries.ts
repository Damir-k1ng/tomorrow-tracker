import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { userApi } from './userApi';

/**
 * React Query hooks for the user-facing server state.
 *
 * These are intentionally thin: each hook is a plain `useQuery` over a
 * `userApi` method — no mutations, no optimistic updates, no manual cache
 * graphs. Query is used strictly as lightweight server-state fetching;
 * business logic stays in the API/entity layers.
 *
 * Caching, retry and refetch policy come from the shared QueryProvider.
 */

/** Query keys for the user-facing server state — one source of truth. */
export const userQueryKeys = {
  me: ['user', 'me'] as const,
  sessions: (page: number) => ['user', 'sessions', page] as const,
  leaderboard: ['user', 'leaderboard'] as const,
};

/** GET /api/v1/user/me — profile, streak and lifetime totals. */
export function useProfileQuery() {
  return useQuery({
    queryKey: userQueryKeys.me,
    queryFn: ({ signal }) => userApi.getMe(signal),
  });
}

/**
 * GET /api/v1/user/sessions — one page of the caller's sessions.
 * `keepPreviousData` keeps the previous page visible while the next loads, so
 * the list never flashes empty when paginating.
 */
export function useSessionsQuery(page: number) {
  return useQuery({
    queryKey: userQueryKeys.sessions(page),
    queryFn: ({ signal }) => userApi.listSessions(page, signal),
    placeholderData: keepPreviousData,
  });
}

/** GET /api/v1/user/leaderboard — the weekly Top-N plus the caller's rank. */
export function useLeaderboardQuery() {
  return useQuery({
    queryKey: userQueryKeys.leaderboard,
    queryFn: ({ signal }) => userApi.getLeaderboard(signal),
  });
}
