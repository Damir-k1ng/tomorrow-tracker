/**
 * User profile domain types — the read-model behind GET /api/v1/user/me.
 *
 * `UserProfile` is the app-internal (camelCase) shape; `RawUserProfile` is the
 * snake_case DTO returned by the Go backend. Mapping lives in `mappers.ts`.
 */
import type {
  RawSession,
  RawStudyProgress,
  StudyProgress,
  StudySession,
} from '@/entities/session';
import type { UserRole } from '@/entities/user';

/**
 * The caller's overview: identity, streak, lifetime totals, active session,
 * and today/week progress. This is also the Mini App's session-recovery
 * source — after a Telegram reopen the client re-fetches /me and restores
 * `activeSession` from the backend, never from a local timer.
 */
export interface UserProfile {
  id: number;
  telegramId: number;
  firstName: string;
  role: UserRole;
  currentStreak: number;
  bestStreak: number;
  totalMinutes: number;
  totalSessions: number;
  lastStudyAt: string | null; // ISO-8601, or null when no qualifying session
  activeSession: StudySession | null;
  progress: StudyProgress;
}

/** Raw profile DTO as returned by the Go backend (GET /api/v1/user/me). */
export interface RawUserProfile {
  id: number;
  telegram_id: number;
  first_name: string;
  role: string;
  current_streak: number;
  best_streak: number;
  total_minutes: number;
  total_sessions: number;
  last_study_at: string | null;
  active_session: RawSession | null;
  progress: RawStudyProgress;
}
