/**
 * Russian-first display formatters for study durations and timestamps.
 *
 * Pure functions, no locale config beyond `ru-RU` — the product is
 * Russian-first. Used by the user Mini App pages to render API data.
 */

/** Format a minute count as a compact Russian duration: "1 ч 30 мин", "45 мин". */
export function formatDuration(minutes: number): string {
  if (minutes <= 0) return '0 мин';
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  if (hours === 0) return `${mins} мин`;
  if (mins === 0) return `${hours} ч`;
  return `${hours} ч ${mins} мин`;
}

/** Total hours rounded to one decimal, for compact stat tiles: "12,5". */
export function formatHours(minutes: number): string {
  return (minutes / 60).toLocaleString('ru-RU', { maximumFractionDigits: 1 });
}

/** The correct Russian plural of "день" for a count: день / дня / дней. */
export function dayWord(days: number): string {
  const n = Math.abs(Math.trunc(days));
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return 'день';
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return 'дня';
  return 'дней';
}

/** A day count with its Russian plural: "1 день", "2 дня", "5 дней". */
export function formatDayCount(days: number): string {
  const n = Math.abs(Math.trunc(days));
  return `${n} ${dayWord(n)}`;
}

/**
 * Format an elapsed-seconds count as a running clock: "MM:SS", or "H:MM:SS"
 * once it passes an hour. For the presentation-only live session timer — the
 * backend remains the authority on the recorded duration.
 */
export function formatElapsed(totalSeconds: number): string {
  const safe = Math.max(0, Math.floor(totalSeconds));
  const hours = Math.floor(safe / 3600);
  const minutes = Math.floor((safe % 3600) / 60);
  const seconds = safe % 60;
  const pad = (n: number) => String(n).padStart(2, '0');
  if (hours > 0) return `${hours}:${pad(minutes)}:${pad(seconds)}`;
  return `${pad(minutes)}:${pad(seconds)}`;
}

/** Format an ISO timestamp as a short Russian date: "16 мая". */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
}

/** Format an ISO timestamp as a short Russian date + time: "16 мая, 21:30". */
export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU', {
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  });
}
