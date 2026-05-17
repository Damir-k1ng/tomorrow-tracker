import { Trophy } from 'lucide-react';
import { PageHeader, QueryError, Screen } from '@/widgets';
import { Card, EmptyState, Skeleton } from '@/shared/ui';
import { useLeaderboardQuery } from '@/services/api';
import type { LeaderboardEntry, LeaderboardPosition } from '@/entities/leaderboard';
import { formatHours } from '@/shared/lib/format';
import { cn } from '@/shared/lib/cn';

/**
 * User Leaderboard — the weekly study ranking.
 *
 * Data: GET /api/v1/user/leaderboard — the Top-N plus the caller's own
 * standing. When the caller is outside the Top-N, their position is shown in a
 * separate row below the list so they always see where they stand.
 */
export default function LeaderboardPage() {
  const query = useLeaderboardQuery();

  return (
    <Screen>
      <PageHeader title="Рейтинг" subtitle="Топ за неделю" />

      {query.isPending && <LeaderboardSkeleton />}

      {query.isError && (
        <QueryError error={query.error} onRetry={() => void query.refetch()} />
      )}

      {query.isSuccess &&
        (query.data.top.length === 0 ? (
          <EmptyState
            icon={Trophy}
            title="Рейтинг пуст"
            description="На этой неделе ещё никто не учился. Стань первым!"
          />
        ) : (
          <>
            <ul className="space-y-2.5">
              {query.data.top.map((entry) => (
                <li key={entry.userId}>
                  <RankRow entry={entry} />
                </li>
              ))}
            </ul>
            <SelfPosition position={query.data.me} />
          </>
        ))}
    </Screen>
  );
}

/** One ranked competitor. The caller's own row is visually highlighted. */
function RankRow({ entry }: { entry: LeaderboardEntry }) {
  return (
    <Card
      className={cn(
        'flex items-center gap-3 p-4',
        entry.isCurrent && 'border-border-strong bg-surface-raised',
      )}
    >
      <span className="w-7 shrink-0 text-center text-sm font-semibold text-muted tabular-nums">
        {entry.rank}
      </span>
      <span className="min-w-0 flex-1 truncate text-sm font-medium text-foreground">
        {entry.name}
        {entry.isCurrent && <span className="ml-1.5 text-xs text-accent">ты</span>}
      </span>
      <span className="shrink-0 text-sm font-semibold tabular-nums text-foreground">
        {formatHours(entry.minutes)} ч
      </span>
    </Card>
  );
}

/**
 * The caller's standing, shown only when they have minutes this week but are
 * outside the Top-N (when they are in the Top-N, their row is already
 * highlighted above, so this would be redundant).
 */
function SelfPosition({ position }: { position: LeaderboardPosition }) {
  if (!position.found || position.inTop) return null;

  return (
    <div className="mt-4">
      <p className="mb-2 px-1 text-xs font-medium text-subtle">Твоё место</p>
      <Card className="flex items-center gap-3 border-border-strong bg-surface-raised p-4">
        <span className="w-7 shrink-0 text-center text-sm font-semibold text-muted tabular-nums">
          {position.rank}
        </span>
        <span className="min-w-0 flex-1 truncate text-sm font-medium text-foreground">
          {position.name}
          <span className="ml-1.5 text-xs text-accent">ты</span>
        </span>
        <span className="shrink-0 text-sm font-semibold tabular-nums text-foreground">
          {formatHours(position.minutes)} ч
        </span>
      </Card>
    </div>
  );
}

/** Loading placeholder shaped like the ranking list. */
function LeaderboardSkeleton() {
  return (
    <div className="space-y-2.5">
      {Array.from({ length: 6 }).map((_, i) => (
        <Skeleton key={i} className="h-[3.75rem] w-full rounded-card" />
      ))}
    </div>
  );
}
