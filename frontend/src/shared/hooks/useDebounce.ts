import { useEffect, useState } from 'react';

/**
 * Debounce a rapidly-changing value (e.g. a search input).
 *
 * Returns the latest value only after it has stayed unchanged for `delayMs`,
 * so a search query fires one request after the user pauses typing rather
 * than one per keystroke.
 */
export function useDebounce<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(value), delayMs);
    return () => window.clearTimeout(id);
  }, [value, delayMs]);

  return debounced;
}
