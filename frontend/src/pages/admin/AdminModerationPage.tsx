import { useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import { AdminSessionCard, Pager, PageHeader, QueryError, Screen } from '@/widgets';
import { Card, EmptyState, Skeleton } from '@/shared/ui';
import { useAdminSessionsQuery } from '@/services/api';
import { cn } from '@/shared/lib/cn';

/**
 * Admin Moderation — the anti-cheat review queue.
 *
 * Data: GET /api/v1/admin/sessions?flagged=true — only sessions carrying
 * anti-cheat evidence. Each row expands into the same correction form as the
 * Sessions screen, so a moderator can adjust validity or flags in place.
 * An empty queue is the healthy state.
 */
export default function AdminModerationPage() {
  const [page, setPage] = useState(1);
  const query = useAdminSessionsQuery({ page, flagged: true });

  return (
    <Screen>
      <PageHeader title="Модерация" subtitle="Анти-чит и проверки" />

      {query.isPending && <ModerationSkeleton />}

      {query.isError && <QueryError error={query.error} onRetry={() => void query.refetch()} />}

      {query.isSuccess &&
        (query.data.items.length === 0 ? (
          <EmptyState
            icon={ShieldCheck}
            title="Очередь пуста"
            description="Сессий с флагами анти-чита нет — проверять нечего."
          />
        ) : (
          <>
            <Card className="mb-3 bg-surface-raised">
              <p className="text-xs text-muted">
                Сессии с флагами анти-чита. Открой сессию, чтобы скорректировать
                длительность, статус или флаги.
              </p>
            </Card>
            <ul className={cn('space-y-2.5', query.isFetching && 'opacity-60')}>
              {query.data.items.map((session) => (
                <li key={session.id}>
                  <AdminSessionCard session={session} />
                </li>
              ))}
            </ul>
            {query.data.pagination.totalPages > 1 && (
              <Pager
                page={page}
                totalPages={query.data.pagination.totalPages}
                busy={query.isFetching}
                onChange={setPage}
              />
            )}
          </>
        ))}
    </Screen>
  );
}

/** Loading placeholder shaped like the moderation queue. */
function ModerationSkeleton() {
  return (
    <div className="space-y-2.5">
      {Array.from({ length: 4 }).map((_, i) => (
        <Skeleton key={i} className="h-[5.5rem] w-full rounded-card" />
      ))}
    </div>
  );
}
