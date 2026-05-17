import { Timer } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { EmptyState } from '@/shared/ui';

/** Admin Sessions — Phase 3A shell. Session review & corrections arrive later. */
export default function AdminSessionsPage() {
  return (
    <Screen>
      <PageHeader title="Сессии" subtitle="Проверка и корректировка" />
      <EmptyState
        icon={Timer}
        title="Скоро"
        description="Здесь появится просмотр сессий и инструменты корректировки длительности."
      />
    </Screen>
  );
}
