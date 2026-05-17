import { useEffect } from 'react';
import { hideBackButton, showBackButton } from '@/telegram/sdk';

/**
 * Bind the Telegram BackButton to a handler for the lifetime of a screen.
 * The button is shown on mount and hidden on unmount, so each screen owns its
 * own back affordance without leaking handlers.
 */
export function useTelegramBackButton(onBack: () => void): void {
  useEffect(() => {
    showBackButton(onBack);
    return () => hideBackButton();
  }, [onBack]);
}
