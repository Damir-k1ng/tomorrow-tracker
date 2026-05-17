import { Users } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { EmptyState } from '@/shared/ui';

/** Admin Users — Phase 3A shell. The user list & details arrive in a later phase. */
export default function AdminUsersPage() {
  return (
    <Screen>
      <PageHeader title="Пользователи" subtitle="Управление пользователями" />
      <EmptyState
        icon={Users}
        title="Скоро"
        description="Здесь появится список пользователей с поиском, ролями и профилями."
      />
    </Screen>
  );
}
