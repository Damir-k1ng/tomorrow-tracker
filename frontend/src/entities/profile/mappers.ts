import { mapSession } from '@/entities/session';
import { toRole } from '@/entities/user';
import type { RawUserProfile, UserProfile } from './types';

/** Map the backend profile DTO (snake_case) into the app's `UserProfile`. */
export function mapProfile(raw: RawUserProfile): UserProfile {
  return {
    id: raw.id,
    telegramId: raw.telegram_id,
    firstName: raw.first_name,
    role: toRole(raw.role),
    currentStreak: raw.current_streak,
    bestStreak: raw.best_streak,
    totalMinutes: raw.total_minutes,
    totalSessions: raw.total_sessions,
    lastStudyAt: raw.last_study_at,
    activeSession: raw.active_session ? mapSession(raw.active_session) : null,
  };
}
