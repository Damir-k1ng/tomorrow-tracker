import { Info, Moon } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { Card } from '@/shared/ui';
import { useTelegramStore } from '@/store';

/** User Settings — Phase 3A shell. Reachable from the Profile screen. */
export default function SettingsPage() {
  const platform = useTelegramStore((s) => s.platform);

  return (
    <Screen>
      <PageHeader title="Настройки" />

      <Card className="p-0">
        <div className="flex items-center gap-3 border-b border-border p-4">
          <Moon className="size-5 text-muted" aria-hidden />
          <span className="flex-1 text-sm text-foreground">Тема</span>
          <span className="text-sm text-subtle">Тёмная</span>
        </div>
        <div className="flex items-center gap-3 p-4">
          <Info className="size-5 text-muted" aria-hidden />
          <span className="flex-1 text-sm text-foreground">Платформа</span>
          <span className="text-sm text-subtle">{platform}</span>
        </div>
      </Card>

      <p className="mt-4 px-1 text-xs text-subtle">
        Tomorrow Tracker — базовая версия. Больше настроек появится позже.
      </p>
    </Screen>
  );
}
