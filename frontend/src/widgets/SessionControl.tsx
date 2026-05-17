import { useCallback } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { CheckCircle2, Flame, Play, Square } from 'lucide-react';
import type { FinishSessionResult, StudySession } from '@/entities/session';
import { useFinishSessionMutation, useStartSessionMutation } from '@/services/api';
import { isApiError } from '@/services/api';
import { Button, Card } from '@/shared/ui';
import { useElapsedSeconds } from '@/shared/hooks/useElapsedSeconds';
import { formatDuration, formatElapsed } from '@/shared/lib/format';
import { haptics } from '@/telegram/sdk';

/**
 * SessionControl — the daily-use study loop on the dashboard.
 *
 * Two states, driven entirely by `activeSession` (from GET /me):
 *   - none active  → a "start session" call to action
 *   - one active   → a running-timer card with a "finish" action
 *
 * The timer is presentation-only (see useElapsedSeconds); the backend computes
 * the recorded duration. Mutations carry no optimistic state — the card
 * re-renders from the refreshed profile after start/finish.
 */
export interface SessionControlProps {
  activeSession: StudySession | null;
}

export function SessionControl({ activeSession }: SessionControlProps) {
  const start = useStartSessionMutation();
  const finish = useFinishSessionMutation();

  const handleStart = useCallback(() => {
    // Clear any prior finish summary before opening a fresh session.
    finish.reset();
    start.mutate(undefined, {
      onSuccess: () => haptics.notify('success'),
      onError: () => haptics.notify('error'),
    });
  }, [start, finish]);

  const handleFinish = useCallback(() => {
    if (!activeSession) return;
    finish.mutate(activeSession.id, {
      onSuccess: () => haptics.notify('success'),
      onError: () => haptics.notify('error'),
    });
  }, [finish, activeSession]);

  if (activeSession) {
    return (
      <ActiveSessionCard
        session={activeSession}
        onFinish={handleFinish}
        finishing={finish.isPending}
        error={finish.isError ? errorText(finish.error) : null}
      />
    );
  }

  return (
    <StartSessionCard
      onStart={handleStart}
      starting={start.isPending}
      error={start.isError ? errorText(start.error) : null}
      lastResult={finish.isSuccess ? finish.data : null}
    />
  );
}

/** The running-session card: live timer + finish action. */
function ActiveSessionCard({
  session,
  onFinish,
  finishing,
  error,
}: {
  session: StudySession;
  onFinish: () => void;
  finishing: boolean;
  error: string | null;
}) {
  const elapsed = useElapsedSeconds(session.startedAt);
  const startedTime = new Date(session.startedAt).toLocaleTimeString('ru-RU', {
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <Card className="mb-3 border-border-strong bg-surface-raised">
      <div className="flex items-center gap-2">
        <span className="relative flex size-2.5" aria-hidden>
          <span className="absolute inline-flex size-full animate-ping rounded-pill bg-accent opacity-60" />
          <span className="relative inline-flex size-2.5 rounded-pill bg-accent" />
        </span>
        <span className="text-xs font-medium uppercase tracking-wide text-accent">
          Идёт сессия
        </span>
      </div>

      <p
        className="mt-3 font-semibold tabular-nums tracking-tight text-foreground"
        style={{ fontSize: 'clamp(2.75rem, 2rem + 6vw, 4rem)', lineHeight: 1 }}
        aria-live="polite"
      >
        {formatElapsed(elapsed)}
      </p>
      <p className="mt-1.5 text-xs text-subtle">Начата в {startedTime}</p>

      <Button
        variant="danger"
        block
        className="mt-4"
        loading={finishing}
        onClick={onFinish}
      >
        {!finishing && <Square className="size-4" aria-hidden />}
        Завершить сессию
      </Button>

      {error && <ErrorLine message={error} />}
    </Card>
  );
}

/** The idle card: start a session, plus the previous session's summary. */
function StartSessionCard({
  onStart,
  starting,
  error,
  lastResult,
}: {
  onStart: () => void;
  starting: boolean;
  error: string | null;
  lastResult: FinishSessionResult | null;
}) {
  return (
    <Card className="mb-3">
      <div className="flex items-center gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-pill bg-surface-raised">
          <Play className="size-4 text-accent" aria-hidden />
        </span>
        <div>
          <p className="text-sm font-semibold text-foreground">Готов учиться?</p>
          <p className="text-xs text-muted">Запусти таймер учебной сессии</p>
        </div>
      </div>

      <Button block className="mt-4" loading={starting} onClick={onStart}>
        {!starting && <Play className="size-4" aria-hidden />}
        Начать сессию
      </Button>

      {error && <ErrorLine message={error} />}

      <AnimatePresence>
        {lastResult && !error && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="overflow-hidden"
          >
            <LastSessionSummary result={lastResult} />
          </motion.div>
        )}
      </AnimatePresence>
    </Card>
  );
}

/** Compact summary shown after a session is finished. */
function LastSessionSummary({ result }: { result: FinishSessionResult }) {
  return (
    <div className="mt-4 flex items-start gap-2.5 rounded-control bg-surface-raised p-3">
      <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-accent" aria-hidden />
      <div className="text-xs">
        <p className="font-medium text-foreground">
          Сессия завершена — {formatDuration(result.sessionMinutes)}
        </p>
        {result.streak?.counted && (
          <p className="mt-0.5 flex items-center gap-1 text-muted">
            <Flame className="size-3 text-accent" aria-hidden />
            {result.streak.broken
              ? `Новая серия: ${result.streak.current}`
              : `Серия: ${result.streak.current} подряд`}
          </p>
        )}
      </div>
    </div>
  );
}

/** Inline mutation-error line. */
function ErrorLine({ message }: { message: string }) {
  return (
    <p className="mt-2.5 text-xs text-danger" role="alert">
      {message}
    </p>
  );
}

/** Resolve a mutation error into a user-facing Russian message. */
function errorText(error: unknown): string {
  return isApiError(error) ? error.userMessage : 'Что-то пошло не так. Попробуйте ещё раз';
}
