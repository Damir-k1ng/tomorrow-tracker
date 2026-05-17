import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import type { FinishSessionResult, StudySession } from '@/entities/session';
import { userApi } from './userApi';

/**
 * React Query hooks for the user-facing server state.
 *
 * Reads are thin `useQuery` wrappers over a `userApi` method. The Phase 3C
 * session lifecycle adds two mutations (start/finish). There are deliberately
 * no optimistic updates: a mutation simply invalidates the affected queries on
 * success, so the UI always re-renders from authoritative backend state.
 *
 * Caching, retry and refetch policy come from the shared QueryProvider.
 */

/** Shared prefix for every page of the sessions listing. */
const SESSIONS_KEY_PREFIX = ['user', 'sessions'] as const;

/** Query keys for the user-facing server state — one source of truth. */
export const userQueryKeys = {
  me: ['user', 'me'] as const,
  sessions: (page: number) => [...SESSIONS_KEY_PREFIX, page] as const,
  leaderboard: ['user', 'leaderboard'] as const,
};

/** GET /api/v1/user/me — profile, streak, lifetime totals and progress. */
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

/**
 * POST /api/v1/user/sessions/start — open a study session.
 *
 * On success the profile query is invalidated, so the dashboard re-fetches
 * `/me` and renders the now-active session from authoritative backend state.
 */
export function useStartSessionMutation() {
  const queryClient = useQueryClient();
  return useMutation<StudySession, unknown, void>({
    mutationFn: () => userApi.startSession(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userQueryKeys.me });
    },
  });
}

/**
 * POST /api/v1/user/sessions/{id}/finish — close a study session.
 *
 * Finishing a session changes the sessions history and the weekly leaderboard,
 * so both are invalidated on success. The profile is invalidated in `onSettled`
 * — on success it reflects the finished session, and on a race ("already
 * finished") it still re-syncs the card to authoritative backend state.
 */
export function useFinishSessionMutation() {
  const queryClient = useQueryClient();
  return useMutation<FinishSessionResult, unknown, number>({
    mutationFn: (sessionId: number) => userApi.finishSession(sessionId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SESSIONS_KEY_PREFIX });
      void queryClient.invalidateQueries({ queryKey: userQueryKeys.leaderboard });
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: userQueryKeys.me });
    },
  });
}
