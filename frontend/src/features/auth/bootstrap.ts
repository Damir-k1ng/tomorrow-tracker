import { ApiError, authApi } from '@/services/api';
import { useAuthStore, useTelegramStore, useUiStore } from '@/store';
import {
  getColorScheme,
  getPlatform,
  initTelegram,
  isTelegramAvailable,
} from '@/telegram/sdk';

/**
 * Auth bootstrap — the single startup sequence.
 *
 *   1. Initialise the Telegram integration (lifecycle, theme, viewport,
 *      safe-area → CSS variables).
 *   2. Record the runtime environment in the telegram + ui stores.
 *   3. Verify the Telegram initData with the backend and resolve the user.
 *
 * The role is known only when this resolves, so the route tree is mounted
 * strictly after — no flicker, no premature admin rendering. Safe to call
 * again (used by the auth-error retry); each run resets to a clean 'loading'.
 */
export async function bootstrapAuth(): Promise<void> {
  const auth = useAuthStore.getState();
  auth.startLoading();

  // 1–2. Telegram environment. initTelegram is idempotent.
  initTelegram((scheme) => useUiStore.getState().setTheme(scheme));
  useTelegramStore.getState().setEnvironment({
    isAvailable: isTelegramAvailable(),
    platform: getPlatform(),
  });
  useTelegramStore.getState().setReady();
  useUiStore.getState().setTheme(getColorScheme());

  // 3. Backend verification — the backend is the sole identity authority.
  try {
    const user = await authApi.verify();
    auth.authenticate(user);
  } catch (err) {
    auth.fail(err instanceof ApiError ? err : new ApiError('network'));
  }
}
