import type { LucideIcon } from 'lucide-react';
import { CalendarRange, Info, Moon, Smartphone, UserRound } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { Card } from '@/shared/ui';
import { useTelegramStore } from '@/store';
import { useProfileQuery } from '@/services/api';
import { cn } from '@/shared/lib/cn';

/**
 * User Settings — a small informational screen. The app is single-purpose and
 * dark-only, so there is little to toggle; this surfaces the app context
 * (theme, platform, weekly goal) and credits.
 */
export default function SettingsPage() {
  const platform = useTelegramStore((s) => s.platform);
  const profile = useProfileQuery();
  const weeklyGoal = profile.data
    ? `${Math.round(profile.data.progress.weeklyTargetMinutes / 60)} ч`
    : '—';

  return (
    <Screen>
      <PageHeader title="Настройки" />

      <h2 className="mb-2.5 px-1 text-sm font-medium text-muted">Приложение</h2>
      <Card className="p-0">
        <Row icon={Moon} label="Тема" value="Тёмная" bordered />
        <Row icon={Smartphone} label="Платформа" value={platform} bordered />
        <Row icon={CalendarRange} label="Недельная цель" value={weeklyGoal} />
      </Card>

      <h2 className="mb-2.5 mt-5 px-1 text-sm font-medium text-muted">О приложении</h2>
      <Card className="p-0">
        <Row icon={Info} label="Приложение" value="Tomorrow Tracker" bordered />
        <Row icon={UserRound} label="Разработчик" value="@King_traff" />
      </Card>

      <p className="mt-5 px-1 text-xs text-subtle">
        Tomorrow Tracker — трекер учебных часов для Tomorrow School Astana.
      </p>
    </Screen>
  );
}

/** One label/value settings row. */
function Row({
  icon: Icon,
  label,
  value,
  bordered,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  bordered?: boolean;
}) {
  return (
    <div className={cn('flex items-center gap-3 p-4', bordered && 'border-b border-border')}>
      <Icon className="size-5 shrink-0 text-muted" aria-hidden />
      <span className="flex-1 text-sm text-foreground">{label}</span>
      <span className="text-sm text-subtle">{value}</span>
    </div>
  );
}
