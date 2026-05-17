import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { userQueryKeys } from '@/services/api';

/**
 * Restores authoritative session state after the Mini App is reopened.
 *
 * Telegram WebViews keep the page alive across a close/reopen, so React state —
 * including any running timer — survives but can be stale. The backend is the
 * single source of truth for active-session state; a local timer is never
 * trusted for recovery.
 *
 * Two signals are covered so recovery is reliable across clients:
 *   - `visibilitychange` → fired when Telegram (mobile/desktop) is reopened or
 *     the device wakes from sleep.
 *   - `pageshow` → fired when the page is restored from the back/forward
 *     cache; `event.persisted` distinguishes a bfcache restore from a fresh
 *     load (a fresh load already fetches /me on mount).
 *
 * A hard browser refresh needs no handling here — it remounts the app, and the
 * profile query fetches /me fresh on mount.
 */
export function useReopenRefetch(): void {
  const queryClient = useQueryClient();

  useEffect(() => {
    function refreshProfile() {
      void queryClient.invalidateQueries({ queryKey: userQueryKeys.me });
    }

    function onVisibilityChange() {
      if (document.visibilityState === 'visible') refreshProfile();
    }

    function onPageShow(event: PageTransitionEvent) {
      if (event.persisted) refreshProfile();
    }

    document.addEventListener('visibilitychange', onVisibilityChange);
    window.addEventListener('pageshow', onPageShow);
    return () => {
      document.removeEventListener('visibilitychange', onVisibilityChange);
      window.removeEventListener('pageshow', onPageShow);
    };
  }, [queryClient]);
}
