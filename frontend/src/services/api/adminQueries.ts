import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import type {
  SessionCorrectionResult,
  SessionPatchInput,
} from '@/entities/admin/types';
import { adminApi, type AdminUserSort } from './adminApi';

/**
 * React Query hooks for the admin-facing server state.
 *
 * Reads are thin `useQuery` wrappers over `adminApi` methods; paginated
 * listings use `keepPreviousData` so the list never flashes empty while the
 * next page or a new filter loads. The one mutation — session correction —
 * invalidates every admin query on success, since a correction can ripple
 * into stats, the users list and a user's streak.
 */

/** Filters accepted by the admin sessions listing. */
export interface AdminSessionFilter {
  page: number;
  valid?: boolean;
  flagged?: boolean;
  userId?: number;
}

/** Query keys for the admin server state — one source of truth. */
export const adminQueryKeys = {
  root: ['admin'] as const,
  stats: ['admin', 'stats'] as const,
  users: (page: number, search: string, sort: AdminUserSort) =>
    ['admin', 'users', page, search, sort] as const,
  user: (id: number) => ['admin', 'user', id] as const,
  audit: (page: number, action: string) => ['admin', 'audit', page, action] as const,
  sessions: (filter: AdminSessionFilter) => ['admin', 'sessions', filter] as const,
};

/** GET /api/v1/admin/stats — operational counters for the admin dashboard. */
export function useAdminStatsQuery() {
  return useQuery({
    queryKey: adminQueryKeys.stats,
    queryFn: ({ signal }) => adminApi.getStats(signal),
  });
}

/** GET /api/v1/admin/users — one page of the users listing. */
export function useAdminUsersQuery(page: number, search: string, sort: AdminUserSort) {
  return useQuery({
    queryKey: adminQueryKeys.users(page, search, sort),
    queryFn: ({ signal }) => adminApi.listUsers({ page, search, sort }, signal),
    placeholderData: keepPreviousData,
  });
}

/** GET /api/v1/admin/users/{id} — one user's full admin profile. */
export function useAdminUserQuery(id: number) {
  return useQuery({
    queryKey: adminQueryKeys.user(id),
    queryFn: ({ signal }) => adminApi.getUser(id, signal),
    enabled: Number.isFinite(id) && id > 0,
  });
}

/** GET /api/v1/admin/audit-logs — one page of the immutable audit log. */
export function useAuditLogsQuery(page: number, action: string) {
  return useQuery({
    queryKey: adminQueryKeys.audit(page, action),
    queryFn: ({ signal }) => adminApi.listAuditLogs({ page, action: action || undefined }, signal),
    placeholderData: keepPreviousData,
  });
}

/** GET /api/v1/admin/sessions — one page of the sessions listing. */
export function useAdminSessionsQuery(filter: AdminSessionFilter) {
  return useQuery({
    queryKey: adminQueryKeys.sessions(filter),
    queryFn: ({ signal }) => adminApi.listSessions(filter, signal),
    placeholderData: keepPreviousData,
  });
}

/**
 * PATCH /api/v1/admin/sessions/{id} — correct a finished session.
 *
 * A correction can change session counters, a user's streak and the audit
 * log, so on success every admin query is invalidated and re-derived from
 * authoritative backend state — no optimistic updates.
 */
export function useCorrectSessionMutation() {
  const queryClient = useQueryClient();
  return useMutation<SessionCorrectionResult, unknown, { id: number; patch: SessionPatchInput }>({
    mutationFn: ({ id, patch }) => adminApi.correctSession(id, patch),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminQueryKeys.root });
    },
  });
}
