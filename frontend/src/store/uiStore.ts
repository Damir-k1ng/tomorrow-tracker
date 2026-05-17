import { create } from 'zustand';
import type { TelegramColorScheme } from '@/telegram/types';

/**
 * UI store — app-level presentation state.
 *
 * `theme` follows the Telegram color scheme at bootstrap (the app is
 * dark-first) and is the single source of truth for the `.dark` / `.light`
 * class on <html>. A future user override would live here too.
 */
interface UiState {
  theme: TelegramColorScheme;
  setTheme: (theme: TelegramColorScheme) => void;
}

export const useUiStore = create<UiState>((set) => ({
  theme: 'dark',
  setTheme: (theme) => set({ theme }),
}));
