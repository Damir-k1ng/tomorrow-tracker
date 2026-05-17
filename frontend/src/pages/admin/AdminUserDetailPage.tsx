import { useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { CalendarClock, Flame, Timer, TriangleAlert } from 'lucide-react';
import { PageHeader, QueryError, Screen, StatCard } from '@/widgets';
import { Card, EmptyState, Skeleton } from '@/shared/ui';
import { useAdminUserQuery } from '@/services/api';
import type { AdminUserDetails } from '@/entities/admin/types';
import type { StudySession } from '@/entities/session';
import { adminPaths } from '@/routes/paths';
import { useTelegramBackButton } from '@/shared/hooks/useTelegramBackButton';
import { formatDate, formatDateTime, formatDuration, formatHours } from '@/shared/lib/format';

/** Russian labels for every role. */
const ROLE_LABEL: Record<AdminUserDetails['profile']['role'], string> = {
  user: 'Пользователь',
  moderator: 'Модератор',
  admin: 'Администратор',
};

/**
 * Admin User Detail — one user's full admin profile.
 *
 * Data: GET /api/v1/admin/users/{id} — identity, streak, lifetime totals, the
 * active session (if any) and the last 20 sessions. Reached by tapping a row
 * on the Users screen; the Telegram BackButton returns to that list.
 */
export default function AdminUserDetailPage() {
  const { id } = useParams();
  const userId = Number(id);
  const navigate = useNavigate();

  const goBack = useCallback(() => navigate(adminPaths.users), [navigate]);
  useTelegramBackButton(goBack);

  const validId = Number.isInteger(userId) && userId > 0;
  const query = useAdminUserQuery(validId ? userId : 0);

  return (
    <Screen>
      <PageHeader title="Пользователь" subtitle="Профиль и сессии" />

      {!validId && (
        <EmptyState
          icon={TriangleAlert}
          title="Некорректная ссылка"
          description="Идентификатор пользователя не распознан."
        />
      )}

      {validId && query.isPending && <DetailSkeleton />}

      {validId && query.isError && (
        <QueryError error={query.error} onRetry={() => void query.refetch()} />
      )}

      {validId && query.isSuccess && <UserDetail details={query.data} />}
    </Screen>
  );
}

function UserDetail({ details }: { details: AdminUserDetails }) {
  const { profile, streak } = details;
  const handle = profile.username ? `@${profile.username}` : `Telegram ID ${profile.telegramId}`;

  return (
    <>
      <Card className="mb-3">
        <p className="text-base font-semibold text-foreground">
          {profile.firstName || 'Без имени'}
        </p>
        <p className="mt-0.5 text-sm text-muted">{handle}</p>
        <dl className="mt-3 space-y-1.5 text-xs">
          <Field label="Роль" value={ROLE_LABEL[profile.role]} />
          <Field label="Внутренний ID" value={String(profile.id)} />
          <Field label="Регистрация" value={formatDate(profile.createdAt)} />
          <Field
            label="Последняя учёба"
            value={streak.lastStudyAt ? formatDate(streak.lastStudyAt) : '—'}
          />
        </dl>
      </Card>

      <div className="mb-3 grid grid-cols-2 gap-3">
        <StatCard
          icon={Flame}
          label="Серия"
          value={streak.current}
          hint={`рекорд: ${streak.best}`}
        />
        <StatCard
          icon={Timer}
          label="Всего"
          value={formatHours(details.totalMinutes)}
          hint="часов"
        />
      </div>

      {details.activeSession && (
        <Card className="mb-3 flex items-center gap-3 border-border-strong">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-pill bg-surface-raised">
            <CalendarClock className="size-4 text-accent" aria-hidden />
          </span>
          <div>
            <p className="text-sm font-medium text-foreground">Идёт активная сессия</p>
            <p className="text-xs text-muted">
              Начата {formatDateTime(details.activeSession.startedAt)}
            </p>
          </div>
        </Card>
      )}

      <h2 className="mb-2.5 mt-5 text-sm font-medium text-muted">Последние сессии</h2>
      {details.recentSessions.length === 0 ? (
        <EmptyState
          icon={Timer}
          title="Сессий пока нет"
          description="У этого пользователя ещё нет завершённых учебных сессий."
        />
      ) : (
        <ul className="space-y-2.5">
          {details.recentSessions.map((session) => (
            <li key={session.id}>
              <SessionRow session={session} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

/** One label/value row inside the profile card. */
function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="text-subtle">{label}</dt>
      <dd className="font-medium text-foreground">{value}</dd>
    </div>
  );
}

/** One session in the user's recent-sessions list. */
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
        <p className={`mt-0.5 text-xs ${status.tone}`}>{status.text}</p>
      </div>
      <span className="shrink-0 text-base font-semibold tabular-nums text-foreground">
        {session.isActive ? '—' : formatDuration(session.durationMinutes)}
      </span>
    </Card>
  );
}

/** Loading placeholder shaped like the detail screen. */
function DetailSkeleton() {
  return (
    <>
      <Skeleton className="mb-3 h-[8.5rem] w-full rounded-card" />
      <div className="mb-3 grid grid-cols-2 gap-3">
        <Skeleton className="h-[5.5rem] w-full rounded-card" />
        <Skeleton className="h-[5.5rem] w-full rounded-card" />
      </div>
      <div className="space-y-2.5">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[4.25rem] w-full rounded-card" />
        ))}
      </div>
    </>
  );
}
