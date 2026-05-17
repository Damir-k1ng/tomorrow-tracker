import { useEffect, useState } from 'react';
import { ScrollText } from 'lucide-react';
import { PageHeader, QueryError, Screen } from '@/widgets';
import { Button, Card, EmptyState, Skeleton } from '@/shared/ui';
import { useAuditLogsQuery } from '@/services/api';
import type { AuditLogEntry } from '@/entities/admin/types';
import { formatDateTime } from '@/shared/lib/format';
import { cn } from '@/shared/lib/cn';

/** Russian labels for the fixed audit-action taxonomy. */
const ACTION_LABEL: Record<string, string> = {
  PATCH_SESSION: 'Корректировка сессии',
  EXPORT_USERS: 'Экспорт пользователей',
  EXPORT_SESSIONS: 'Экспорт сессий',
};

/** Russian labels for the entity-type taxonomy. */
const ENTITY_LABEL: Record<string, string> = {
  session: 'Сессия',
  user: 'Пользователь',
  export: 'Экспорт',
};

/** Action filter chips: a label plus the backend `action` query value. */
const FILTERS: ReadonlyArray<{ label: string; value: string }> = [
  { label: 'Все', value: '' },
  { label: 'Корректировки', value: 'PATCH_SESSION' },
  { label: 'Экспорт юзеров', value: 'EXPORT_USERS' },
  { label: 'Экспорт сессий', value: 'EXPORT_SESSIONS' },
];

/**
 * Admin Audit — the immutable, append-only action log.
 *
 * Data: GET /api/v1/admin/audit-logs (newest first), paginated, with an
 * optional action filter. Changing the filter resets to page 1.
 */
export default function AdminAuditPage() {
  const [action, setAction] = useState('');
  const [page, setPage] = useState(1);
  const query = useAuditLogsQuery(page, action);

  useEffect(() => {
    setPage(1);
  }, [action]);

  return (
    <Screen>
      <PageHeader title="Аудит" subtitle="Журнал действий администраторов" />

      <FilterChips active={action} onChange={setAction} />

      {query.isPending && <AuditSkeleton />}

      {query.isError && <QueryError error={query.error} onRetry={() => void query.refetch()} />}

      {query.isSuccess &&
        (query.data.items.length === 0 ? (
          <EmptyState
            icon={ScrollText}
            title="Записей нет"
            description="Действия администраторов появятся здесь по мере их совершения."
          />
        ) : (
          <>
            <ul className={cn('space-y-2.5', query.isFetching && 'opacity-60')}>
              {query.data.items.map((entry) => (
                <li key={entry.id}>
                  <AuditRow entry={entry} />
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

/** Horizontal scroll row of action-filter chips. */
function FilterChips({ active, onChange }: { active: string; onChange: (v: string) => void }) {
  return (
    <div className="mb-4 flex gap-2 overflow-x-auto pb-1">
      {FILTERS.map((f) => (
        <button
          key={f.value}
          type="button"
          onClick={() => onChange(f.value)}
          className={cn(
            'shrink-0 rounded-pill px-3.5 py-1.5 text-xs font-medium transition-colors',
            active === f.value
              ? 'bg-accent text-accent-foreground'
              : 'bg-surface-raised text-muted hover:text-foreground',
          )}
        >
          {f.label}
        </button>
      ))}
    </div>
  );
}

/** One immutable audit-log record. */
function AuditRow({ entry }: { entry: AuditLogEntry }) {
  const actionLabel = ACTION_LABEL[entry.action] ?? entry.action;
  const entityLabel =
    entry.entityType &&
    `${ENTITY_LABEL[entry.entityType] ?? entry.entityType}${
      entry.entityId != null ? ` #${entry.entityId}` : ''
    }`;

  return (
    <Card className="p-4">
      <div className="flex items-baseline justify-between gap-3">
        <p className="text-sm font-medium text-foreground">{actionLabel}</p>
        <span className="shrink-0 text-xs text-subtle">{formatDateTime(entry.createdAt)}</span>
      </div>
      {entityLabel && <p className="mt-1 text-xs text-muted">{entityLabel}</p>}
      {entry.reason && (
        <p className="mt-1.5 text-xs text-muted">
          <span className="text-subtle">Причина: </span>
          {entry.reason}
        </p>
      )}
      <p className="mt-1.5 text-[0.7rem] text-subtle">Администратор #{entry.adminId}</p>
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

/** Loading placeholder shaped like the audit list. */
function AuditSkeleton() {
  return (
    <div className="space-y-2.5">
      {Array.from({ length: 5 }).map((_, i) => (
        <Skeleton key={i} className="h-[6rem] w-full rounded-card" />
      ))}
    </div>
  );
}
