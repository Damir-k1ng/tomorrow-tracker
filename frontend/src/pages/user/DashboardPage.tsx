import { CalendarCheck, Flame, Timer, Trophy } from 'lucide-react';
import { PageHeader, QueryError, Screen, StatCard } from '@/widgets';
import { Card, Skeleton } from '@/shared/ui';
import { useAuthStore } from '@/store';
import { useLeaderboardQuery, useProfileQuery } from '@/services/api';
import type { UserProfile } from '@/entities/profile';
import { formatDateTime, formatHours } from '@/shared/lib/format';

/**
 * User Dashboard — the caller's study overview.
 *
 * Data: GET /api/v1/user/me (profile, streak, lifetime totals) drives the
 * tiles; GET /api/v1/user/leaderboard supplies the weekly rank. The rank query
 * is secondary — if it is still loading or fails, the rank tile degrades
 * gracefully to a placeholder while the rest of the dashboard renders.
 */
export default function DashboardPage() {
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
      {profile.activeSession && (
        <Card className="mb-3 flex items-center gap-3 border-border-strong">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-pill bg-surface-raised">
            <Flame className="size-4 text-accent" aria-hidden />
          </span>
          <div>
            <p className="text-sm font-medium text-foreground">Идёт учебная сессия</p>
            <p className="text-xs text-muted">
              Начата {formatDateTime(profile.activeSession.startedAt)}
            </p>
          </div>
        </Card>
      )}

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
    <div className="grid grid-cols-2 gap-3">
      {Array.from({ length: 4 }).map((_, i) => (
        <Skeleton key={i} className="h-[5.5rem] w-full rounded-card" />
      ))}
    </div>
  );
}
