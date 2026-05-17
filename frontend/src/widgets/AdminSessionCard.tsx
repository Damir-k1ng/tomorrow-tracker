import { useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { ChevronDown, Flag } from 'lucide-react';
import { ANTI_CHEAT_FLAGS, type AdminSession } from '@/entities/admin/types';
import { useCorrectSessionMutation } from '@/services/api';
import { isApiError } from '@/services/api';
import { Button, Card } from '@/shared/ui';
import { cn } from '@/shared/lib/cn';
import { formatDate, formatDuration } from '@/shared/lib/format';
import { haptics } from '@/telegram/sdk';

/** Per-session anti-cheat cap (minutes) — mirrors the backend MaxSessionMinutes. */
const MAX_DURATION = 720;

/** Russian labels for the anti-cheat evidence vocabulary. */
const FLAG_LABEL: Record<string, string> = {
  manual_review: 'Ручная проверка',
  suspicious_duration: 'Подозрит. длительность',
  rapid_restarts: 'Частые рестарты',
  overlap_detected: 'Пересечение сессий',
  admin_invalidated: 'Аннулировано админом',
};

/**
 * AdminSessionCard — one session row that expands into a correction form.
 *
 * Shared by the Sessions and Moderation screens. Active sessions are shown
 * read-only (the backend rejects correcting a running session); finished
 * sessions expand into a PATCH /admin/sessions/{id} form.
 */
export interface AdminSessionCardProps {
  session: AdminSession;
}

export function AdminSessionCard({ session }: AdminSessionCardProps) {
  const [expanded, setExpanded] = useState(false);
  const owner = session.ownerFirstName || (session.ownerUsername ? `@${session.ownerUsername}` : 'Без имени');
  const correctable = !session.isActive;

  return (
    <Card className="p-0">
      <button
        type="button"
        disabled={!correctable}
        onClick={() => setExpanded((v) => !v)}
        className={cn(
          'flex w-full items-center gap-3 p-4 text-left',
          correctable && 'cursor-pointer',
        )}
      >
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-foreground">{owner}</p>
          <p className="mt-0.5 text-xs text-muted">{formatDate(session.startedAt)}</p>
          <SessionStatus session={session} />
        </div>
        <span className="shrink-0 text-base font-semibold tabular-nums text-foreground">
          {session.isActive ? '—' : formatDuration(session.durationMinutes)}
        </span>
        {correctable && (
          <ChevronDown
            className={cn(
              'size-4 shrink-0 text-subtle transition-transform',
              expanded && 'rotate-180',
            )}
            aria-hidden
          />
        )}
      </button>

      <AnimatePresence initial={false}>
        {expanded && correctable && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="overflow-hidden"
          >
            <CorrectionForm session={session} onDone={() => setExpanded(false)} />
          </motion.div>
        )}
      </AnimatePresence>
    </Card>
  );
}

/** Status line + flag chips shown under a session. */
function SessionStatus({ session }: { session: AdminSession }) {
  const status = session.isActive
    ? { text: 'Идёт сейчас', tone: 'text-accent' }
    : session.isValid
      ? { text: 'Засчитана', tone: 'text-muted' }
      : { text: 'Не засчитана', tone: 'text-subtle' };

  return (
    <div className="mt-1 flex flex-wrap items-center gap-1.5">
      <span className={cn('text-xs', status.tone)}>{status.text}</span>
      {session.antiCheatFlags.length > 0 && (
        <span className="inline-flex items-center gap-1 rounded-pill bg-surface-raised px-2 py-0.5 text-[0.7rem] font-medium text-danger">
          <Flag className="size-3" aria-hidden />
          {session.antiCheatFlags.length}
        </span>
      )}
    </div>
  );
}

/** The PATCH /admin/sessions/{id} correction form. Mounted fresh on expand,
 * so its fields always initialise from the current session. */
function CorrectionForm({ session, onDone }: { session: AdminSession; onDone: () => void }) {
  const [duration, setDuration] = useState(String(session.durationMinutes));
  const [isValid, setIsValid] = useState(session.isValid);
  const [flags, setFlags] = useState<string[]>(session.antiCheatFlags);
  const [reason, setReason] = useState('');
  const correct = useCorrectSessionMutation();

  const durationNum = Number(duration);
  const durationOk =
    duration.trim() !== '' &&
    Number.isInteger(durationNum) &&
    durationNum >= 0 &&
    durationNum <= MAX_DURATION;
  const reasonOk = reason.trim().length > 0;
  const canSave = durationOk && reasonOk && !correct.isPending;

  function toggleFlag(flag: string) {
    setFlags((prev) => (prev.includes(flag) ? prev.filter((f) => f !== flag) : [...prev, flag]));
  }

  function handleSave() {
    if (!canSave) return;
    correct.mutate(
      {
        id: session.id,
        patch: {
          durationMinutes: durationNum,
          isValid,
          antiCheatFlags: flags,
          reason: reason.trim(),
        },
      },
      {
        onSuccess: () => {
          haptics.notify('success');
          onDone();
        },
        onError: () => haptics.notify('error'),
      },
    );
  }

  const errorText = correct.isError
    ? isApiError(correct.error)
      ? correct.error.userMessage
      : 'Не удалось сохранить корректировку'
    : null;

  return (
    // A recessed editing well — visually distinct from the row it expands from.
    <div className="space-y-4 border-t border-border bg-background/50 p-4">
      <Field label="Длительность, мин">
        <input
          type="number"
          inputMode="numeric"
          min={0}
          max={MAX_DURATION}
          value={duration}
          onChange={(e) => setDuration(e.target.value)}
          className={inputClass}
          aria-label="Длительность в минутах"
        />
        {!durationOk && (
          <p className="mt-1 text-[0.7rem] text-danger">От 0 до {MAX_DURATION} минут.</p>
        )}
      </Field>

      <Field label="Статус">
        <div className="flex gap-2">
          <SegmentButton active={isValid} onClick={() => setIsValid(true)}>
            Засчитана
          </SegmentButton>
          <SegmentButton active={!isValid} onClick={() => setIsValid(false)}>
            Не засчитана
          </SegmentButton>
        </div>
      </Field>

      <Field label="Флаги анти-чита">
        <div className="flex flex-wrap gap-2">
          {ANTI_CHEAT_FLAGS.map((flag) => (
            <button
              key={flag}
              type="button"
              onClick={() => toggleFlag(flag)}
              className={cn(
                'rounded-pill px-3 py-1.5 text-xs font-medium transition-colors',
                flags.includes(flag)
                  ? 'bg-accent text-accent-foreground'
                  : 'bg-surface-raised text-muted hover:text-foreground',
              )}
            >
              {FLAG_LABEL[flag]}
            </button>
          ))}
        </div>
      </Field>

      <Field label="Причина (обязательно)">
        <textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          maxLength={500}
          rows={2}
          placeholder="Зачем нужна корректировка"
          aria-label="Причина корректировки"
          className={cn(inputClass, 'resize-none py-2.5')}
        />
      </Field>

      {errorText && (
        <p className="text-xs text-danger" role="alert">
          {errorText}
        </p>
      )}

      <div className="flex gap-2">
        <Button variant="secondary" size="sm" onClick={onDone} disabled={correct.isPending}>
          Отмена
        </Button>
        <Button size="sm" block loading={correct.isPending} disabled={!canSave} onClick={handleSave}>
          Сохранить
        </Button>
      </div>
    </div>
  );
}

/** Shared input styling for the correction form. */
const inputClass = cn(
  'w-full rounded-control border border-border bg-surface px-3.5 text-sm text-foreground',
  'h-11 placeholder:text-subtle focus-visible:border-border-strong focus-visible:outline-none',
);

/** A labelled form field. */
function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="mb-1.5 text-xs font-medium text-muted">{label}</p>
      {children}
    </div>
  );
}

/** One button of the validity segmented control. */
function SegmentButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'h-10 flex-1 rounded-control text-xs font-medium transition-colors',
        active
          ? 'bg-accent text-accent-foreground'
          : 'bg-surface-raised text-muted hover:text-foreground',
      )}
    >
      {children}
    </button>
  );
}
