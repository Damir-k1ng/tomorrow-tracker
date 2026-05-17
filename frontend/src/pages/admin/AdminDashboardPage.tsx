import { Clock, Flame, Timer, TrendingUp, UserPlus, Users } from 'lucide-react';
import { PageHeader, QueryError, Screen, StatCard } from '@/widgets';
import { Skeleton } from '@/shared/ui';
import { useAdminStatsQuery } from '@/services/api';
import type { AdminStats } from '@/entities/admin/types';

/**
 * Admin Dashboard — operational overview.
 *
 * Data: GET /api/v1/admin/stats. The six tiles map 1:1 onto the counters the
 * endpoint actually returns — no placeholder cards.
 */
export default function AdminDashboardPage() {
  const stats = useAdminStatsQuery();

  return (
    <Screen>
      <PageHeader title="Обзор" subtitle="Панель администратора" />

      {stats.isPending && <StatsSkeleton />}

      {stats.isError && <QueryError error={stats.error} onRetry={() => void stats.refetch()} />}

      {stats.isSuccess && <StatsGrid stats={stats.data} />}
    </Screen>
  );
}

/** Format a number for a stat tile: integers plain, fractions to one decimal. */
function formatNumber(value: number): string {
  return value.toLocaleString('ru-RU', { maximumFractionDigits: 1 });
}

function StatsGrid({ stats }: { stats: AdminStats }) {
  const newUsersHint =
    stats.newUsers7d > 0 ? `+${formatNumber(stats.newUsers7d)} за неделю` : 'за всё время';

  return (
    <div className="grid grid-cols-2 gap-3">
      <StatCard
        icon={Users}
        label="Пользователи"
        value={formatNumber(stats.totalUsers)}
        hint={newUsersHint}
      />
      <StatCard
        icon={UserPlus}
        label="Новые за 7д"
        value={formatNumber(stats.newUsers7d)}
        hint="пользователей"
      />
      <StatCard
        icon={Timer}
        label="Сессии"
        value={formatNumber(stats.completedSessions)}
        hint="завершено"
      />
      <StatCard
        icon={Clock}
        label="Активные"
        value={formatNumber(stats.activeSessions)}
        hint="идут сейчас"
      />
      <StatCard
        icon={TrendingUp}
        label="Часы учёбы"
        value={formatNumber(stats.totalStudyHours)}
        hint="всего"
      />
      <StatCard
        icon={Flame}
        label="Серии"
        value={formatNumber(stats.bestStreak)}
        hint={`рекорд · в среднем ${formatNumber(stats.averageStreak)}`}
      />
    </div>
  );
}

/** Loading placeholder shaped like the stat grid so the layout never jumps. */
function StatsSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-3">
      {Array.from({ length: 6 }).map((_, i) => (
        <Skeleton key={i} className="h-[5.5rem] w-full rounded-card" />
      ))}
    </div>
  );
}
