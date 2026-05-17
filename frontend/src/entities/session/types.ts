/**
 * Study-session domain types.
 *
 * `StudySession` is the app-internal (camelCase) shape; `RawSession` is the
 * snake_case DTO returned by the Go backend. Mapping lives in `mappers.ts`.
 */

/** A single study session, in app-internal shape. */
export interface StudySession {
  id: number;
  userId: number;
  startedAt: string; // ISO-8601
  endedAt: string | null; // null while the session is running
  durationMinutes: number;
  isActive: boolean;
  isValid: boolean;
}

/** Raw session DTO as returned by the Go backend (GET /api/v1/user/*). */
export interface RawSession {
  id: number;
  user_id: number;
  started_at: string;
  ended_at: string | null;
  duration_minutes: number;
  is_active: boolean;
  is_valid: boolean;
}
