import { CalendarCheck, Flame, Timer, Trophy } from 'lucide-react';
import {
  PageHeader,
  QueryError,
  Screen,
  SessionControl,
  StatCard,
  WeeklyProgress,
} from '@/widgets';
import { Skeleton } from '@/shared/ui';
import { useAuthStore } from '@/store';
import { useLeaderboardQuery, useProfileQuery } from '@/services/api';
import { useReopenRefetch } from '@/shared/hooks/useReopenRefetch';
import type { UserProfile } from '@/entities/profile';
import { formatHours } from '@/shared/lib/format';

/**
 * User Dashboard — the caller's study overview and daily-use loop.
 *
 * Data: GET /api/v1/user/me drives the session control, weekly progress and
 * the stat tiles; GET /api/v1/user/leaderboard supplies the weekly rank. The
 * rank query is secondary — if it is still loading or fails, the rank tile
 * degrades gracefully while the rest of the dashboard renders.
 *
 * Recovery: useReopenRefetch re-fetches /me whenever the Mini App returns to
 * visibility, so an active session is always restored from backend state
 * after a Telegram close/reopen.
 */
export default function DashboardPage() {
  useReopenRefetch();

  const firstName = useAuthStore((s) => s.user?.firstName ?? '');
  const profile = useProfileQuery();
  const leaderboard = useLeaderboardQuery();

  const rankValue = leaderboard.isPending
    ? '…'
    : leaderboard.data?.me.found
      ? `#${leaderboard.data.me.rank}`
      : '—';

  return (
    <Screen>
      <PageHeader
        title={firstName ? `Привет, ${firstName}` : 'Привет'}
        subtitle="Сводка по твоей учёбе"
      />

      {profile.isPending && <DashboardSkeleton />}

      {profile.isError && (
        <QueryError error={profile.error} onRetry={() => void profile.refetch()} />
      )}

      {profile.isSuccess && <DashboardContent profile={profile.data} rankValue={rankValue} />}
    </Screen>
  );
}

function DashboardContent({
  profile,
  rankValue,
}: {
  profile: UserProfile;
  rankValue: string;
}) {
  return (
    <>
      <SessionControl activeSession={profile.activeSession} />
      <WeeklyProgress progress={profile.progress} />

      <div className="grid grid-cols-2 gap-3">
        <StatCard
          icon={Flame}
          label="Серия"
          value={profile.currentStreak}
          hint={`рекорд: ${profile.bestStreak}`}
        />
        <StatCard
          icon={Timer}
          label="Всего"
          value={formatHours(profile.totalMinutes)}
          hint="часов"
        />
        <StatCard
          icon={CalendarCheck}
          label="Сессий"
          value={profile.totalSessions}
          hint="завершено"
        />
        <StatCard icon={Trophy} label="Место" value={rankValue} hint="за неделю" />
      </div>
    </>
  );
}

/** Loading placeholder shaped like the dashboard so the layout never jumps. */
function DashboardSkeleton() {
  return (
    <>
      <Skeleton className="mb-3 h-[11rem] w-full rounded-card" />
      <Skeleton className="mb-3 h-[6.5rem] w-full rounded-card" />
      <div className="grid grid-cols-2 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[5.5rem] w-full rounded-card" />
        ))}
      </div>
    </>
  );
}
