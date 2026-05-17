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
