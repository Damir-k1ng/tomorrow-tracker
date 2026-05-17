/** API layer — public surface. Components import from here, never raw fetch. */
export { apiClient } from './apiClient';
export type { RequestOptions } from './apiClient';
export { ApiError, isApiError, type ApiErrorKind } from './errors';
export { authApi } from './authApi';
export { userApi, SESSIONS_PAGE_SIZE, type PageMeta, type SessionsPage } from './userApi';
export { useProfileQuery, useSessionsQuery, useLeaderboardQuery, userQueryKeys } from './userQueries';
export { adminApi, type ExportRange } from './adminApi';
