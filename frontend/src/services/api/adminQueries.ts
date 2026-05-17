import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { adminApi, type AdminUserSort } from './adminApi';

/**
 * React Query hooks for the admin-facing server state.
 *
 * Thin `useQuery` wrappers over `adminApi` methods — read-only, no mutations.
 * Paginated listings use `keepPreviousData` so the list never flashes empty
 * while the next page or a new search loads. Caching/retry policy comes from
 * the shared QueryProvider.
 */

/** Query keys for the admin server state — one source of truth. */
export const adminQueryKeys = {
  stats: ['admin', 'stats'] as const,
  users: (page: number, search: string, sort: AdminUserSort) =>
    ['admin', 'users', page, search, sort] as const,
  user: (id: number) => ['admin', 'user', id] as const,
  audit: (page: number, action: string) => ['admin', 'audit', page, action] as const,
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
