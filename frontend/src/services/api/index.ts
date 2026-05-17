/** API layer — public surface. Components import from here, never raw fetch. */
export { apiClient } from './apiClient';
export type { RequestOptions } from './apiClient';
export { ApiError, isApiError, type ApiErrorKind } from './errors';
export { authApi } from './authApi';
export { userApi, SESSIONS_PAGE_SIZE, type PageMeta, type SessionsPage } from './userApi';
export {
  useProfileQuery,
  useSessionsQuery,
  useLeaderboardQuery,
  useStartSessionMutation,
  useFinishSessionMutation,
  userQueryKeys,
} from './userQueries';
export { adminApi, ADMIN_PAGE_SIZE, type ExportRange, type AdminUserSort } from './adminApi';
export {
  useAdminStatsQuery,
  useAdminUsersQuery,
  useAdminUserQuery,
  useAuditLogsQuery,
  useAdminSessionsQuery,
  useCorrectSessionMutation,
  adminQueryKeys,
  type AdminSessionFilter,
} from './adminQueries';
