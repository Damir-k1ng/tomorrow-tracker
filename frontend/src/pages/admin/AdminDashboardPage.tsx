import { ClipboardCheck, Flag, Flame, Timer, TrendingUp, Users } from 'lucide-react';
import { PageHeader, Screen, StatCard } from '@/widgets';
import { Card } from '@/shared/ui';

/**
 * Admin Dashboard — Phase 3A shell.
 *
 * Per the Phase 3A scope this is the dashboard SHELL only: six metric cards,
 * no charts and no live analytics. Values are placeholders; they are wired to
 * GET /api/v1/admin/stats (via adminApi) in a later phase.
 */
export default function AdminDashboardPage() {
  return (
    <Screen>
      <PageHeader title="Обзор" subtitle="Панель администратора" />

      <div className="grid grid-cols-2 gap-3">
        <StatCard icon={Users} label="Пользователи" value="—" hint="скоро" />
        <StatCard icon={Timer} label="Сессии" value="—" hint="скоро" />
        <StatCard icon={Flame} label="Активные серии" value="—" hint="скоро" />
        <StatCard icon={TrendingUp} label="Топ пользователей" value="—" hint="скоро" />
        <StatCard icon={Flag} label="Флаги" value="—" hint="скоро" />
        <StatCard icon={ClipboardCheck} label="Корректировки" value="—" hint="скоро" />
      </div>

      <Card className="mt-3">
        <p className="text-sm text-muted">
          Это базовая панель администратора. Аналитика, графики и инструменты
          модерации появятся в следующих фазах.
        </p>
      </Card>
    </Screen>
  );
}
