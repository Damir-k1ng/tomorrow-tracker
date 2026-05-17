import { CalendarCheck, Flame, Trophy } from 'lucide-react';
import { PageHeader, QueryError, Screen, StatCard } from '@/widgets';
import { Card, Skeleton } from '@/shared/ui';
import { useProfileQuery } from '@/services/api';
import type { UserProfile } from '@/entities/profile';
import { cn } from '@/shared/lib/cn';
import { dayWord, formatDate } from '@/shared/lib/format';

/**
 * User Streak — the caller's study-day streak.
 *
 * Data: GET /api/v1/user/me (current/best streak, last study day) — no extra
 * request. A streak counts a day on which the user finished a study session
 * of at least 30 minutes; miss a day and it resets.
 */
export default function StreakPage() {
  const profile = useProfileQuery();

  return (
    <Screen>
      <PageHeader title="Серия" subtitle="Дни учёбы подряд" />

      {profile.isPending && <StreakSkeleton />}

      {profile.isError && (
        <QueryError error={profile.error} onRetry={() => void profile.refetch()} />
      )}

      {profile.isSuccess && <StreakContent profile={profile.data} />}
    </Screen>
  );
}

function StreakContent({ profile }: { profile: UserProfile }) {
  const active = profile.currentStreak > 0;

  return (
    <>
      <Card glass className="mb-3 flex flex-col items-center py-8 text-center">
        <span
          className={cn(
            'flex size-16 items-center justify-center rounded-pill',
            active ? 'bg-accent/15' : 'bg-surface-raised',
          )}
        >
          <Flame
            className={cn('size-7', active ? 'text-accent' : 'text-subtle')}
            aria-hidden
          />
        </span>

        {active ? (
          <>
            <p
              className="mt-4 font-semibold tabular-nums tracking-tight text-foreground"
              style={{ fontSize: 'clamp(3rem, 2rem + 8vw, 4.5rem)', lineHeight: 1 }}
            >
              {profile.currentStreak}
            </p>
            <p className="mt-1.5 text-sm text-muted">
              {dayWord(profile.currentStreak)} подряд
            </p>
          </>
        ) : (
          <>
            <p className="mt-4 text-lg font-semibold text-foreground">Серия не начата</p>
            <p className="mt-1 max-w-xs text-sm text-muted">
              Проведи сегодня учебную сессию от 30 минут — и серия пойдёт.
            </p>
          </>
        )}
      </Card>

      <div className="mb-3 grid grid-cols-2 gap-3">
        <StatCard
          icon={Trophy}
          label="Рекорд"
          value={profile.bestStreak}
          hint={`${dayWord(profile.bestStreak)} подряд`}
        />
        <StatCard
          icon={CalendarCheck}
          label="Последняя учёба"
          value={profile.lastStudyAt ? formatDate(profile.lastStudyAt) : '—'}
          hint={profile.lastStudyAt ? 'засчитанный день' : 'пока нет'}
        />
      </div>

      <Card>
        <p className="text-sm text-muted">
          День засчитывается в серию, если ты завершил учебную сессию длиной не
          менее 30 минут. Пропустишь день — серия начнётся заново.
        </p>
      </Card>
    </>
  );
}

/** Loading placeholder shaped like the streak screen. */
function StreakSkeleton() {
  return (
    <>
      <Skeleton className="mb-3 h-[13rem] w-full rounded-card" />
      <div className="mb-3 grid grid-cols-2 gap-3">
        <Skeleton className="h-[5.5rem] w-full rounded-card" />
        <Skeleton className="h-[5.5rem] w-full rounded-card" />
      </div>
      <Skeleton className="h-[5rem] w-full rounded-card" />
    </>
  );
}
