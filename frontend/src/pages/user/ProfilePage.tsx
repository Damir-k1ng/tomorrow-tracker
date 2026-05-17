import { Link } from 'react-router-dom';
import { ChevronRight, LogOut, Settings as SettingsIcon } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { Button, Card } from '@/shared/ui';
import { useAuthStore } from '@/store';
import { userPaths } from '@/routes/paths';
import { closeApp } from '@/telegram/sdk';

/** User Profile — Phase 3A shell. Shows the resolved identity + entry points. */
export default function ProfilePage() {
  const user = useAuthStore((s) => s.user);
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

      <Card className="mt-3 p-0">
        <Link
          to={userPaths.settings}
          className="flex items-center gap-3 p-4 active:bg-surface-raised"
        >
          <SettingsIcon className="size-5 text-muted" aria-hidden />
          <span className="flex-1 text-sm text-foreground">Настройки</span>
          <ChevronRight className="size-4 text-subtle" aria-hidden />
        </Link>
      </Card>

      <Button
        variant="ghost"
        block
        className="mt-6"
        onClick={() => closeApp()}
      >
        <LogOut className="size-4" aria-hidden />
        Закрыть приложение
      </Button>
    </Screen>
  );
}
