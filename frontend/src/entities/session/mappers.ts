import type {
  FinishSessionResult,
  RawFinishSessionResult,
  RawSession,
  RawSessionStreak,
  RawStudyProgress,
  SessionStreak,
  StudyProgress,
  StudySession,
} from './types';

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

/** Map the backend progress DTO (snake_case) into the app's `StudyProgress`. */
export function mapProgress(raw: RawStudyProgress): StudyProgress {
  return {
    todayMinutes: raw.today_minutes,
    weekMinutes: raw.week_minutes,
    remainingMinutes: raw.remaining_minutes,
    weeklyTargetMinutes: raw.weekly_target_minutes,
  };
}

/** Map the backend streak DTO (snake_case) into the app's `SessionStreak`. */
function mapStreak(raw: RawSessionStreak): SessionStreak {
  return {
    counted: raw.counted,
    current: raw.current,
    best: raw.best,
    continued: raw.continued,
    broken: raw.broken,
    newRecord: raw.new_record,
  };
}

/** Map the backend finish-session DTO into the app's `FinishSessionResult`. */
export function mapFinishResult(raw: RawFinishSessionResult): FinishSessionResult {
  return {
    sessionMinutes: raw.session_minutes,
    progress: mapProgress(raw.progress),
    streak: raw.streak ? mapStreak(raw.streak) : null,
  };
}
