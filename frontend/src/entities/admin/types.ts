/**
 * Admin domain types — shapes for the existing Go Admin API.
 *
 * Phase 3A wires the typed layer; the Admin dashboard renders shell cards only
 * (no charts, no live analytics yet).
 */

/** GET /api/v1/admin/stats — operational counters. */
export interface AdminStats {
  totalUsers: number;
  activeSessions: number;
  totalStudyHours: number;
  completedSessions: number;
  bestStreak: number;
  averageStreak: number;
  newUsers7d: number;
}

/** Raw stats DTO (snake_case) as returned by the backend. */
export interface RawAdminStats {
  total_users: number;
  active_sessions: number;
  total_study_hours: number;
  completed_sessions: number;
  best_streak: number;
  average_streak: number;
  new_users_7d: number;
}

/** Pagination envelope shared by admin list endpoints. */
export interface Paginated<T> {
  items: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}
