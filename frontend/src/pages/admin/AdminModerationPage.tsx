import { ShieldAlert } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { EmptyState } from '@/shared/ui';

/** Admin Moderation — Phase 3A shell. Anti-cheat moderation tools arrive later. */
export default function AdminModerationPage() {
  return (
    <Screen>
      <PageHeader title="Модерация" subtitle="Анти-чит и проверки" />
      <EmptyState
        icon={ShieldAlert}
        title="Скоро"
        description="Здесь появятся инструменты модерации: флаги анти-чита и проверка сессий."
      />
    </Screen>
  );
}
