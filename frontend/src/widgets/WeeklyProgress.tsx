import type { StudyProgress } from '@/entities/session';
import { Card } from '@/shared/ui';
import { formatDuration, formatHours } from '@/shared/lib/format';

/**
 * WeeklyProgress — the caller's progress toward the weekly study goal.
 *
 * All values are server-computed (GET /me `progress` block); this widget only
 * renders them. The bar fill is capped at 100% so an over-target week never
 * overflows the track.
 */
export interface WeeklyProgressProps {
  progress: StudyProgress;
}

export function WeeklyProgress({ progress }: WeeklyProgressProps) {
  const { weekMinutes, remainingMinutes, weeklyTargetMinutes } = progress;
  const goalReached = remainingMinutes <= 0;
  const pct =
    weeklyTargetMinutes > 0
      ? Math.min(100, Math.round((weekMinutes / weeklyTargetMinutes) * 100))
      : 0;

  return (
    <Card glass className="mb-3">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-sm font-medium text-muted">Цель недели</span>
        <span className="text-xs text-subtle">
          {formatHours(weekMinutes)} / {formatHours(weeklyTargetMinutes)} ч
        </span>
      </div>

      <div
        className="mt-3 h-2.5 overflow-hidden rounded-pill bg-surface-raised"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        aria-label="Прогресс по недельной цели"
      >
        <div
          className="h-full rounded-pill bg-accent transition-[width] duration-500 ease-out"
          style={{ width: `${pct}%` }}
        />
      </div>

      <p className="mt-2.5 text-xs text-muted">
        {goalReached ? (
          <span className="font-medium text-accent">Недельная цель достигнута 🎉</span>
        ) : (
          <>Осталось {formatDuration(remainingMinutes)} до цели</>
        )}
      </p>
    </Card>
  );
}
