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

/**
 * The caller's study progress for the current local day and week. All values
 * are server-computed in the configured timezone — the Mini App renders them
 * but never derives its own. Present on GET /api/v1/user/me and on the
 * finish-session response.
 */
export interface StudyProgress {
  todayMinutes: number;
  weekMinutes: number;
  remainingMinutes: number;
  weeklyTargetMinutes: number;
}

/** Raw progress DTO as returned by the Go backend. */
export interface RawStudyProgress {
  today_minutes: number;
  week_minutes: number;
  remaining_minutes: number;
  weekly_target_minutes: number;
}

/**
 * The streak outcome of finishing a session. `counted` is false when the
 * session was too short to affect the streak.
 */
export interface SessionStreak {
  counted: boolean;
  current: number;
  best: number;
  continued: boolean;
  broken: boolean;
  newRecord: boolean;
}

/** Raw streak DTO as returned by the Go backend. */
export interface RawSessionStreak {
  counted: boolean;
  current: number;
  best: number;
  continued: boolean;
  broken: boolean;
  new_record: boolean;
}

/**
 * The result of finishing a study session: the server-computed length, the
 * refreshed progress totals, and the streak evaluation. `streak` is null when
 * the streak update itself failed (non-fatal — the session was still saved).
 */
export interface FinishSessionResult {
  sessionMinutes: number;
  progress: StudyProgress;
  streak: SessionStreak | null;
}

/** Raw finish-session DTO as returned by the Go backend. */
export interface RawFinishSessionResult {
  session_minutes: number;
  progress: RawStudyProgress;
  streak: RawSessionStreak | null;
}
