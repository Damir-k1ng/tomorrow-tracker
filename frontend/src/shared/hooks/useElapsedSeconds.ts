import { useEffect, useState } from 'react';

/**
 * Live elapsed-seconds counter since an ISO start timestamp.
 *
 * Presentation-only: it drives the running session timer in the UI and is
 * never the authority on a session's duration — the backend computes that on
 * finish. The counter is re-derived from wall-clock time on every tick, so it
 * stays correct even if a tick is dropped while the Mini App is backgrounded.
 *
 * Passing `null` (no active session) yields 0 and runs no interval.
 */
export function useElapsedSeconds(startedAt: string | null): number {
  const [seconds, setSeconds] = useState(() => computeElapsed(startedAt));

  useEffect(() => {
    if (!startedAt) {
      setSeconds(0);
      return;
    }
    setSeconds(computeElapsed(startedAt));
    const id = window.setInterval(() => setSeconds(computeElapsed(startedAt)), 1000);
    return () => window.clearInterval(id);
  }, [startedAt]);

  return seconds;
}

/** Whole seconds between `startedAt` and now; never negative. */
function computeElapsed(startedAt: string | null): number {
  if (!startedAt) return 0;
  const startMs = new Date(startedAt).getTime();
  if (Number.isNaN(startMs)) return 0;
  return Math.max(0, Math.floor((Date.now() - startMs) / 1000));
}
