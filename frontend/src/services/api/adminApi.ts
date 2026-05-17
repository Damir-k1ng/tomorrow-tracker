import type { AdminStats, RawAdminStats } from '@/entities/admin/types';
import { apiClient } from './apiClient';

function mapStats(r: RawAdminStats): AdminStats {
  return {
    totalUsers: r.total_users,
    activeSessions: r.active_sessions,
    totalStudyHours: r.total_study_hours,
    completedSessions: r.completed_sessions,
    bestStreak: r.best_streak,
    averageStreak: r.average_streak,
    newUsers7d: r.new_users_7d,
  };
}

/** YYYY-MM-DD range parameters required by the export endpoints. */
export interface ExportRange {
  from: string;
  to: string;
}

/**
 * Admin API — typed wrapper for the existing Go Admin endpoints. Phase 3A uses
 * `getStats` for the dashboard shell and the export calls for the Export UX;
 * list/audit/correction wrappers are added when their screens are built.
 */
export const adminApi = {
  /** GET /api/v1/admin/stats — operational counters. */
  async getStats(signal?: AbortSignal): Promise<AdminStats> {
    return mapStats(await apiClient.get<RawAdminStats>('/admin/stats', { signal }));
  },

  /**
   * GET /api/v1/admin/export/users — backend-generated CSV. The frontend only
   * streams the bytes; it never builds CSV client-side.
   */
  exportUsers(range: ExportRange): Promise<Blob> {
    return apiClient.download(`/admin/export/users?from=${range.from}&to=${range.to}`);
  },

  /** GET /api/v1/admin/export/sessions — backend-generated CSV. */
  exportSessions(range: ExportRange): Promise<Blob> {
    return apiClient.download(`/admin/export/sessions?from=${range.from}&to=${range.to}`);
  },
};
