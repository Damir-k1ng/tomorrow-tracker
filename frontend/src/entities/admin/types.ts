/**
 * Admin domain types — shapes for the Go Admin API (/api/v1/admin/*).
 *
 * Each domain object has an app-internal camelCase form and a snake_case
 * `Raw*` DTO matching the backend exactly. Mapping lives in `adminApi.ts`.
 */
import type { RawSession, StudySession } from '@/entities/session';
import type { UserRole } from '@/entities/user';

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

/** Pagination metadata, app-internal shape. */
export interface PageMeta {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

/** Pagination envelope shared by admin list endpoints. */
export interface Paginated<T> {
  items: T[];
  pagination: PageMeta;
}

/** One row of GET /api/v1/admin/users. */
export interface AdminUser {
  id: number;
  telegramId: number;
  username: string;
  firstName: string;
  role: UserRole;
  currentStreak: number;
  bestStreak: number;
  lastStudyAt: string | null;
  createdAt: string;
}

/** Raw admin-user DTO (snake_case). */
export interface RawAdminUser {
  id: number;
  telegram_id: number;
  username: string;
  first_name: string;
  role: string;
  current_streak: number;
  best_streak: number;
  last_study_at: string | null;
  created_at: string;
}

/** GET /api/v1/admin/users/{id} — a user's full admin profile. */
export interface AdminUserDetails {
  profile: AdminUser;
  streak: { current: number; best: number; lastStudyAt: string | null };
  totalMinutes: number;
  totalHours: number;
  activeSession: StudySession | null;
  recentSessions: StudySession[];
}

/** Raw user-details DTO (snake_case). */
export interface RawAdminUserDetails {
  profile: RawAdminUser;
  streak: { current: number; best: number; last_study_at: string | null };
  total_minutes: number;
  total_hours: number;
  active_session: RawSession | null;
  recent_sessions: RawSession[];
}

/** One row of GET /api/v1/admin/audit-logs. */
export interface AuditLogEntry {
  id: number;
  adminId: number;
  action: string;
  entityType: string | null;
  entityId: number | null;
  targetUserId: number | null;
  reason: string | null;
  createdAt: string;
}

/**
 * Raw audit-log DTO (snake_case). The before/after JSON snapshots are not
 * surfaced in the list view, so they are typed loosely and ignored by the
 * mapper.
 */
export interface RawAuditLogEntry {
  id: number;
  admin_id: number;
  action: string;
  entity_type: string | null;
  entity_id: number | null;
  target_user_id: number | null;
  before_data: unknown;
  after_data: unknown;
  reason: string | null;
  created_at: string;
}

/**
 * The fixed anti-cheat evidence vocabulary. Must stay in lockstep with the
 * backend whitelist (internal/api/request.go `allowedAntiCheatFlags`) — the
 * API rejects any flag outside this set.
 */
export const ANTI_CHEAT_FLAGS = [
  'manual_review',
  'suspicious_duration',
  'rapid_restarts',
  'overlap_detected',
  'admin_invalidated',
] as const;

export type AntiCheatFlag = (typeof ANTI_CHEAT_FLAGS)[number];

/** One row of GET /api/v1/admin/sessions — a session plus its owner identity. */
export interface AdminSession {
  id: number;
  userId: number;
  startedAt: string;
  endedAt: string | null;
  durationMinutes: number;
  isActive: boolean;
  isValid: boolean;
  antiCheatFlags: string[];
  ownerFirstName: string;
  ownerUsername: string;
}

/** Raw admin-session DTO (snake_case). */
export interface RawAdminSession {
  id: number;
  user_id: number;
  started_at: string;
  ended_at: string | null;
  duration_minutes: number;
  is_active: boolean;
  is_valid: boolean;
  anti_cheat_flags: string[] | null;
  created_at: string;
  owner_first_name: string;
  owner_username: string;
}

/** The admin-supplied correction sent to PATCH /api/v1/admin/sessions/{id}. */
export interface SessionPatchInput {
  durationMinutes: number;
  isValid: boolean;
  antiCheatFlags: string[];
  reason: string;
}

/** The outcome of a session correction, surfaced back to the admin UI. */
export interface SessionCorrectionResult {
  sessionId: number;
  streakRecomputed: boolean;
  currentStreak: number;
  bestStreak: number;
}

/** Raw session-correction DTO (snake_case). */
export interface RawSessionCorrectionResult {
  session_id: number;
  user_id: number;
  old_duration_minutes: number;
  new_duration_minutes: number;
  old_is_valid: boolean;
  new_is_valid: boolean;
  streak_recomputed: boolean;
  current_streak: number;
  best_streak: number;
}
