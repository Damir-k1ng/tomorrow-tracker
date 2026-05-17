import { useEffect, useState } from 'react';
import { Timer } from 'lucide-react';
import { AdminSessionCard, Pager, PageHeader, QueryError, Screen } from '@/widgets';
import { EmptyState, Skeleton } from '@/shared/ui';
import { useAdminSessionsQuery } from '@/services/api';
import { cn } from '@/shared/lib/cn';

/** Validity filter: a label plus the optional `valid` query value. */
const FILTERS: ReadonlyArray<{ label: string; valid: boolean | undefined }> = [
  { label: 'Все', valid: undefined },
  { label: 'Засчитанные', valid: true },
  { label: 'Незасчитанные', valid: false },
];

/**
 * Admin Sessions — review and correct study sessions.
 *
 * Data: GET /api/v1/admin/sessions (newest first), paginated, with a validity
 * filter. Each finished session expands into a correction form
 * (PATCH /admin/sessions/{id}); active sessions are read-only.
 */
export default function AdminSessionsPage() {
  const [filterIndex, setFilterIndex] = useState(0);
  const [page, setPage] = useState(1);
  const query = useAdminSessionsQuery({ page, valid: FILTERS[filterIndex].valid });

  useEffect(() => {
    setPage(1);
  }, [filterIndex]);

  return (
    <Screen>
      <PageHeader title="Сессии" subtitle="Проверка и корректировка" />

      <div className="mb-4 flex gap-2">
        {FILTERS.map((f, i) => (
          <button
            key={f.label}
            type="button"
            onClick={() => setFilterIndex(i)}
            className={cn(
              'rounded-pill px-3.5 py-1.5 text-xs font-medium transition-colors',
              i === filterIndex
                ? 'bg-accent text-accent-foreground'
                : 'bg-surface-raised text-muted hover:text-foreground',
            )}
          >
            {f.label}
          </button>
        ))}
      </div>

      {query.isPending && <SessionsSkeleton />}

      {query.isError && <QueryError error={query.error} onRetry={() => void query.refetch()} />}

      {query.isSuccess &&
        (query.data.items.length === 0 ? (
          <EmptyState
            icon={Timer}
            title="Сессий нет"
            description="С выбранным фильтром сессии не найдены."
          />
        ) : (
          <>
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

/** Loading placeholder shaped like the session list. */
function SessionsSkeleton() {
  return (
    <div className="space-y-2.5">
      {Array.from({ length: 5 }).map((_, i) => (
        <Skeleton key={i} className="h-[5.5rem] w-full rounded-card" />
      ))}
    </div>
  );
}
