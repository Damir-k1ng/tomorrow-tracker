import { ScrollText } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { EmptyState } from '@/shared/ui';

/** Admin Audit — Phase 3A shell. The immutable audit log viewer arrives later. */
export default function AdminAuditPage() {
  return (
    <Screen>
      <PageHeader title="Аудит" subtitle="Журнал действий администраторов" />
      <EmptyState
        icon={ScrollText}
        title="Скоро"
        description="Здесь появится неизменяемый журнал аудита всех действий администраторов."
      />
    </Screen>
  );
}
