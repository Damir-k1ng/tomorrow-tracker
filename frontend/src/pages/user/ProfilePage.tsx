import { Link } from 'react-router-dom';
import { ChevronRight, LogOut, Settings as SettingsIcon } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { Button, Card, Skeleton } from '@/shared/ui';
import { useAuthStore } from '@/store';
import { useProfileQuery } from '@/services/api';
import type { UserProfile } from '@/entities/profile';
import { userPaths } from '@/routes/paths';
import { closeApp } from '@/telegram/sdk';
import { formatHours } from '@/shared/lib/format';

/**
 * User Profile — identity, lifetime study stats, and app entry points.
 *
 * Identity comes from the auth store (resolved at bootstrap); the stat strip
 * comes from GET /api/v1/user/me. The stats are secondary — if they fail to
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
      {profile.isPending && <Skeleton className="h-[5rem] w-full rounded-card" />}
      {profile.isError && (
        <p className="px-1 text-xs text-subtle">Не удалось загрузить статистику.</p>
      )}
      {profile.isSuccess && <StatStrip profile={profile.data} />}

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

/**
 * StatStrip — three lifetime metrics in one row, divided by hairlines. A
 * single compact card (no per-tile icon chrome) so all three read clearly
 * even on a 320px screen.
 */
function StatStrip({ profile }: { profile: UserProfile }) {
  return (
    <Card className="flex divide-x divide-border p-0">
      <Stat value={formatHours(profile.totalMinutes)} label="Часы" />
      <Stat value={profile.totalSessions} label="Сессии" />
      <Stat value={profile.currentStreak} label="Серия" />
    </Card>
  );
}

/** One column of the stat strip. */
function Stat({ value, label }: { value: React.ReactNode; label: string }) {
  return (
    <div className="min-w-0 flex-1 px-2 py-4 text-center">
      <p className="truncate text-xl font-semibold tabular-nums tracking-tight text-foreground">
        {value}
      </p>
      <p className="mt-1 text-xs text-muted">{label}</p>
    </div>
  );
}
