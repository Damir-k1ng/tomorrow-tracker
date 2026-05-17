import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ChevronRight, Flame, Search, Users } from 'lucide-react';
import { PageHeader, QueryError, Screen } from '@/widgets';
import { Button, Card, EmptyState, Skeleton } from '@/shared/ui';
import { useAdminUsersQuery } from '@/services/api';
import type { AdminUser } from '@/entities/admin/types';
import { adminUserDetailPath } from '@/routes/paths';
import { useDebounce } from '@/shared/hooks/useDebounce';
import { cn } from '@/shared/lib/cn';

/** Russian labels for the non-default roles shown as a badge. */
const ROLE_BADGE: Partial<Record<AdminUser['role'], string>> = {
  admin: 'Админ',
  moderator: 'Модератор',
};

/**
 * Admin Users — paginated, searchable user directory.
 *
 * Data: GET /api/v1/admin/users (newest first). Search is debounced so one
 * request fires after the admin stops typing; changing the query resets to
 * page 1. Tapping a row opens that user's detail screen.
 */
export default function AdminUsersPage() {
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebounce(search.trim(), 350);
  const query = useAdminUsersQuery(page, debouncedSearch, '-created_at');
  const navigate = useNavigate();

  // A new search term restarts pagination from the first page.
  useEffect(() => {
    setPage(1);
  }, [debouncedSearch]);

  return (
    <Screen>
      <PageHeader title="Пользователи" subtitle="Управление пользователями" />

      <SearchField value={search} onChange={setSearch} />

      {query.isPending && <UsersSkeleton />}

      {query.isError && <QueryError error={query.error} onRetry={() => void query.refetch()} />}

      {query.isSuccess &&
        (query.data.items.length === 0 ? (
          <EmptyState
            icon={Users}
            title="Никого не найдено"
            description={
              debouncedSearch
                ? 'Попробуй изменить поисковый запрос.'
                : 'Пользователи появятся здесь, как только начнут пользоваться ботом.'
            }
          />
        ) : (
          <>
            <ul className={cn('space-y-2.5', query.isFetching && 'opacity-60')}>
              {query.data.items.map((user) => (
                <li key={user.id}>
                  <UserRow user={user} onOpen={() => navigate(adminUserDetailPath(user.id))} />
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

/** Debounced search box. */
function SearchField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="relative mb-4">
      <Search
        className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-subtle"
        aria-hidden
      />
      <input
        type="search"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="Поиск по имени или username"
        aria-label="Поиск пользователей"
        className={cn(
          'h-11 w-full rounded-control border border-border bg-surface pl-10 pr-4',
          'text-sm text-foreground placeholder:text-subtle',
          'focus-visible:border-border-strong focus-visible:outline-none',
        )}
      />
    </div>
  );
}

/** One user in the directory list. */
function UserRow({ user, onOpen }: { user: AdminUser; onOpen: () => void }) {
  const badge = ROLE_BADGE[user.role];
  const handle = user.username ? `@${user.username}` : `ID ${user.telegramId}`;

  return (
    <Card interactive onClick={onOpen} className="flex items-center gap-3 p-4">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-medium text-foreground">
            {user.firstName || 'Без имени'}
          </p>
          {badge && (
            <span className="shrink-0 rounded-pill bg-surface-raised px-2 py-0.5 text-[0.7rem] font-medium text-accent">
              {badge}
            </span>
          )}
        </div>
        <p className="mt-0.5 truncate text-xs text-muted">{handle}</p>
      </div>
      <span className="flex shrink-0 items-center gap-1 text-sm font-semibold tabular-nums text-foreground">
        <Flame className="size-3.5 text-accent" aria-hidden />
        {user.currentStreak}
      </span>
      <ChevronRight className="size-4 shrink-0 text-subtle" aria-hidden />
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

/** Loading placeholder shaped like the user list. */
function UsersSkeleton() {
  return (
    <div className="space-y-2.5">
      {Array.from({ length: 6 }).map((_, i) => (
        <Skeleton key={i} className="h-[4.25rem] w-full rounded-card" />
      ))}
    </div>
  );
}
