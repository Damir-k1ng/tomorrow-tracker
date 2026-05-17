import { Flame } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { EmptyState } from '@/shared/ui';

/** User Streak — Phase 3A shell. Streak history & insights arrive later. */
export default function StreakPage() {
  return (
    <Screen>
      <PageHeader title="Серия" subtitle="Дни учёбы подряд" />
      <EmptyState
        icon={Flame}
        title="Скоро"
        description="Здесь появится твоя текущая и рекордная серия учебных дней."
      />
    </Screen>
  );
}
