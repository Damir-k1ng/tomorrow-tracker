import { useState } from 'react';
import { Timer } from 'lucide-react';
import { PageHeader, QueryError, Screen } from '@/widgets';
import { Button, Card, EmptyState, Skeleton } from '@/shared/ui';
import { useSessionsQuery } from '@/services/api';
import type { StudySession } from '@/entities/session';
import { formatDate, formatDuration } from '@/shared/lib/format';
import { cn } from '@/shared/lib/cn';

/**
 * User Sessions — the caller's study-session history.
 *
 * Data: GET /api/v1/user/sessions, paginated newest-first. The page index is
 * local UI state; `useSessionsQuery` keeps the previous page visible while the
 * next loads, so the list never flashes empty when paginating.
 */
export default function SessionsPage() {
  const [page, setPage] = useState(1);
  const query = useSessionsQuery(page);

  return (
    <Screen>
      <PageHeader title="Сессии" subtitle="История учебных сессий" />

      {query.isPending && <SessionsSkeleton />}

      {query.isError && (
        <QueryError error={query.error} onRetry={() => void query.refetch()} />
      )}

      {query.isSuccess &&
        (query.data.items.length === 0 ? (
          <EmptyState
            icon={Timer}
            title="Пока нет сессий"
            description="Здесь появится история твоих учебных сессий, когда ты начнёшь заниматься."
          />
        ) : (
          <>
            <ul className={cn('space-y-2.5', query.isFetching && 'opacity-60')}>
              {query.data.items.map((session) => (
                <li key={session.id}>
                  <SessionRow session={session} />
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

/** One session in the history list: date on the left, duration on the right. */
function SessionRow({ session }: { session: StudySession }) {
  const status = session.isActive
    ? { text: 'Идёт сейчас', tone: 'text-accent' }
    : session.isValid
      ? { text: 'Завершена', tone: 'text-muted' }
      : { text: 'Не засчитана', tone: 'text-subtle' };

  return (
    <Card className="flex items-center justify-between gap-3 p-4">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-foreground">
          {formatDate(session.startedAt)}
        </p>
        <p className={cn('mt-0.5 text-xs', status.tone)}>{status.text}</p>
      </div>
      <span className="shrink-0 text-base font-semibold tabular-nums text-foreground">
        {session.isActive ? '—' : formatDuration(session.durationMinutes)}
      </span>
    </Card>
  );
}

/** Prev/next pager. Buttons are disabled at the bounds and while fetching. */
function Pager({
  page,
  totalPages,
  busy,
  onChange,
}: {
  page: number;
  totalPages: number;
  busy: boolean;
  onChange: (page: number) => void;
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <Button
        variant="secondary"
        size="sm"
        disabled={busy || page <= 1}
        onClick={() => onChange(page - 1)}
      >
        Назад
      </Button>
      <span className="text-xs text-muted">
        Стр. {page} из {totalPages}
      </span>
      <Button
        variant="secondary"
        size="sm"
        disabled={busy || page >= totalPages}
        onClick={() => onChange(page + 1)}
      >
        Вперёд
      </Button>
    </div>
  );
}

/** Loading placeholder shaped like the session list. */
function SessionsSkeleton() {
  return (
    <div className="space-y-2.5">
      {Array.from({ length: 5 }).map((_, i) => (
        <Skeleton key={i} className="h-[4.25rem] w-full rounded-card" />
      ))}
    </div>
  );
}
