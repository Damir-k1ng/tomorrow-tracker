import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { userQueryKeys } from '@/services/api';

/**
 * Restores authoritative session state after the Mini App is reopened.
 *
 * Telegram WebViews keep the page alive across a close/reopen, so React state —
 * including any running timer — survives but can be stale. On every return to
 * visibility this invalidates the profile query, forcing a fresh GET /me. The
 * backend is the single source of truth for active-session state; a local
 * timer is never trusted for recovery.
 */
export function useReopenRefetch(): void {
  const queryClient = useQueryClient();

  useEffect(() => {
    function onVisibilityChange() {
      if (document.visibilityState === 'visible') {
        void queryClient.invalidateQueries({ queryKey: userQueryKeys.me });
      }
    }
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => document.removeEventListener('visibilitychange', onVisibilityChange);
  }, [queryClient]);
}
