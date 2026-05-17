import { Link } from 'react-router-dom';
import { CalendarCheck, ChevronRight, Flame, LogOut, Settings as SettingsIcon, Timer } from 'lucide-react';
import { PageHeader, Screen, StatCard } from '@/widgets';
import { Button, Card, Skeleton } from '@/shared/ui';
import { useAuthStore } from '@/store';
import { useProfileQuery } from '@/services/api';
import { userPaths } from '@/routes/paths';
import { closeApp } from '@/telegram/sdk';
import { formatHours } from '@/shared/lib/format';

/**
 * User Profile — identity, lifetime study stats, and app entry points.
 *
 * Identity comes from the auth store (resolved at bootstrap); the stat tiles
 * come from GET /api/v1/user/me. The stats are secondary — if they fail to
 * load the page still works for identity and navigation.
 */
export default function ProfilePage() {
  const user = useAuthStore((s) => s.user);
  const profile = useProfileQuery();
  const initial = user?.firstName?.charAt(0).toUpperCase() ?? '?';

  return (
    <Screen>
      <PageHeader title="Профиль" />

      <Card className="flex items-center gap-4">
        <div className="flex size-14 shrink-0 items-center justify-center rounded-pill bg-accent/15 text-xl font-semibold text-accent">
          {initial}
        </div>
        <div className="min-w-0">
          <p className="truncate text-base font-medium text-foreground">
            {user?.firstName || 'Пользователь'}
          </p>
          {user?.username && (
            <p className="truncate text-sm text-muted">@{user.username}</p>
          )}
        </div>
      </Card>

      <h2 className="mb-2.5 mt-5 px-1 text-sm font-medium text-muted">Статистика</h2>
      {profile.isPending && <StatsSkeleton />}
      {profile.isError && (
        <p className="px-1 text-xs text-subtle">Не удалось загрузить статистику.</p>
      )}
      {profile.isSuccess && (
        <div className="grid grid-cols-3 gap-3">
          <StatCard
            icon={Timer}
            label="Часы"
            value={formatHours(profile.data.totalMinutes)}
            hint="всего"
          />
          <StatCard
            icon={CalendarCheck}
            label="Сессий"
            value={profile.data.totalSessions}
            hint="завершено"
          />
          <StatCard
            icon={Flame}
            label="Серия"
            value={profile.data.currentStreak}
            hint={`рекорд ${profile.data.bestStreak}`}
          />
        </div>
      )}

      <Card className="mt-5 p-0">
        <Link
          to={userPaths.settings}
          className="flex items-center gap-3 p-4 active:bg-surface-raised"
        >
          <SettingsIcon className="size-5 text-muted" aria-hidden />
          <span className="flex-1 text-sm text-foreground">Настройки</span>
          <ChevronRight className="size-4 text-subtle" aria-hidden />
        </Link>
      </Card>

      <Button variant="ghost" block className="mt-6" onClick={() => closeApp()}>
        <LogOut className="size-4" aria-hidden />
        Закрыть приложение
      </Button>
    </Screen>
  );
}

/** Loading placeholder for the three stat tiles. */
function StatsSkeleton() {
  return (
    <div className="grid grid-cols-3 gap-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <Skeleton key={i} className="h-[5.5rem] w-full rounded-card" />
      ))}
    </div>
  );
}
