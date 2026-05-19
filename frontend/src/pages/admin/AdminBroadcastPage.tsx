import { useState } from 'react';
import { CheckCircle2, Megaphone, Send, TriangleAlert, Users } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { Button, Card } from '@/shared/ui';
import {
  isApiError,
  useAdminStatsQuery,
  useBroadcastsQuery,
  useStartBroadcastMutation,
  type Broadcast,
} from '@/services/api';
import { haptics } from '@/telegram/sdk';

/** Telegram's plain-text message ceiling — mirrors the backend limit. */
const MAX_CHARS = 4096;

/**
 * Admin Broadcast — compose a message and fan it out to every user.
 *
 * The send runs in the background on the server; this page reads the broadcast
 * list (which auto-polls while one is running) to show live "sent N/total"
 * progress. Sending is irreversible, so it always goes through an explicit
 * confirmation step.
 */
export default function AdminBroadcastPage() {
  const [text, setText] = useState('');
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const stats = useAdminStatsQuery();
  const broadcasts = useBroadcastsQuery();
  const startMutation = useStartBroadcastMutation();

  const recipients = stats.data?.totalUsers ?? 0;
  const trimmed = text.trim();
  const tooLong = text.length > MAX_CHARS;

  const running = broadcasts.data?.find((b) => b.status === 'running') ?? null;
  const history = (broadcasts.data ?? []).filter((b) => b.status !== 'running');
  const canSend = trimmed.length > 0 && !tooLong && running === null && !startMutation.isPending;

  async function send() {
    setError(null);
    try {
      await startMutation.mutateAsync(trimmed);
      haptics.notify('success');
      setText('');
      setConfirming(false);
    } catch (err) {
      haptics.notify('error');
      setConfirming(false);
      setError(isApiError(err) ? err.userMessage : 'Не удалось запустить рассылку');
    }
  }

  return (
    <Screen>
      <PageHeader title="Рассылка" subtitle="Сообщение всем пользователям бота" />

      {running && (
        <div className="mb-3">
          <BroadcastProgress broadcast={running} />
        </div>
      )}

      <Card>
        <label htmlFor="broadcast-text" className="text-sm font-medium text-foreground">
          Текст сообщения
        </label>
        <textarea
          id="broadcast-text"
          value={text}
          onChange={(e) => {
            setText(e.target.value);
            setConfirming(false);
          }}
          rows={5}
          placeholder="Введите сообщение, которое получат все пользователи…"
          disabled={running !== null}
          className="mt-2 w-full resize-none rounded-control border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-subtle focus-visible:border-border-strong focus-visible:outline-none disabled:opacity-50"
        />
        <div className="mt-1.5 flex items-center justify-between text-xs">
          <span className="flex items-center gap-1 text-muted">
            <Users className="size-3.5" aria-hidden />
            Получателей: {recipients}
          </span>
          <span className={tooLong ? 'text-danger' : 'text-subtle'}>
            {text.length} / {MAX_CHARS}
          </span>
        </div>

        {running ? (
          <p className="mt-4 text-xs text-muted">
            Дождитесь завершения текущей рассылки, чтобы запустить новую.
          </p>
        ) : confirming ? (
          <div className="mt-4 rounded-control bg-danger/12 p-3">
            <p className="text-sm text-foreground">
              Отправить сообщение {recipients}&nbsp;пользователям? Остановить рассылку
              после запуска нельзя.
            </p>
            <div className="mt-3 flex gap-2">
              <Button size="sm" loading={startMutation.isPending} onClick={() => void send()}>
                <Send className="size-4" aria-hidden />
                Отправить
              </Button>
              <Button
                variant="secondary"
                size="sm"
                disabled={startMutation.isPending}
                onClick={() => setConfirming(false)}
              >
                Отмена
              </Button>
            </div>
          </div>
        ) : (
          <Button
            block
            size="sm"
            className="mt-4"
            disabled={!canSend}
            onClick={() => setConfirming(true)}
          >
            <Megaphone className="size-4" aria-hidden />
            Отправить всем
          </Button>
        )}

        {error && (
          <div className="mt-3 flex items-center gap-2 rounded-control bg-danger/12 px-3 py-2 text-sm text-danger">
            <TriangleAlert className="size-4 shrink-0" aria-hidden />
            {error}
          </div>
        )}
      </Card>

      {history.length > 0 && (
        <div className="mt-5">
          <h2 className="mb-2 text-sm font-medium text-muted">История рассылок</h2>
          <div className="space-y-2">
            {history.map((b) => (
              <BroadcastHistoryRow key={b.id} broadcast={b} />
            ))}
          </div>
        </div>
      )}
    </Screen>
  );
}

/** Live progress card for the broadcast currently being delivered. */
function BroadcastProgress({ broadcast }: { broadcast: Broadcast }) {
  const processed = broadcast.sentCount + broadcast.failedCount;
  const total = broadcast.totalRecipients;
  const pct = total > 0 ? Math.min(100, Math.round((processed / total) * 100)) : 0;

  return (
    <Card>
      <div className="flex items-center gap-2">
        <Megaphone className="size-4 animate-pulse text-accent" aria-hidden />
        <p className="text-sm font-medium text-foreground">Рассылка идёт…</p>
      </div>
      <p className="mt-1 text-xs text-muted line-clamp-2">{broadcast.message}</p>

      <div className="mt-3 h-2 w-full overflow-hidden rounded-pill bg-surface-raised">
        <div
          className="h-full rounded-pill bg-accent transition-[width] duration-500"
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className="mt-2 text-xs text-muted">
        Отправлено {broadcast.sentCount} из {total}
        {broadcast.failedCount > 0 && (
          <span className="text-danger"> · ошибок {broadcast.failedCount}</span>
        )}
      </p>
    </Card>
  );
}

/** One finished broadcast in the history list. */
function BroadcastHistoryRow({ broadcast }: { broadcast: Broadcast }) {
  const date = new Date(broadcast.createdAt).toLocaleString('ru-RU', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
  return (
    <Card>
      <div className="flex items-start gap-2">
        <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-accent" aria-hidden />
        <div className="min-w-0 flex-1">
          <p className="line-clamp-2 text-sm text-foreground">{broadcast.message}</p>
          <p className="mt-1 text-xs text-muted">
            Отправлено {broadcast.sentCount} из {broadcast.totalRecipients}
            {broadcast.failedCount > 0 && `, ошибок ${broadcast.failedCount}`}
            {' · '}
            {date}
          </p>
        </div>
      </div>
    </Card>
  );
}
