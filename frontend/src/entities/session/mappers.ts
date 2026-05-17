import type { RawSession, StudySession } from './types';

/** Map the backend session DTO (snake_case) into the app's `StudySession`. */
export function mapSession(raw: RawSession): StudySession {
  return {
    id: raw.id,
    userId: raw.user_id,
    startedAt: raw.started_at,
    endedAt: raw.ended_at,
    durationMinutes: raw.duration_minutes,
    isActive: raw.is_active,
    isValid: raw.is_valid,
  };
}
