import { create } from 'zustand';

/**
 * Telegram store — facts about the runtime environment, captured once at
 * bootstrap. Components read this instead of probing window.Telegram.
 */
interface TelegramState {
  /** True when running inside the real Telegram client (vs. browser dev). */
  isAvailable: boolean;
  /** Telegram client platform: 'ios' | 'android' | 'tdesktop' | 'unknown' | ... */
  platform: string;
  /** Whether the Telegram lifecycle (ready/expand) has been initialised. */
  ready: boolean;

  setEnvironment: (env: { isAvailable: boolean; platform: string }) => void;
  setReady: () => void;
}

export const useTelegramStore = create<TelegramState>((set) => ({
  isAvailable: false,
  platform: 'unknown',
  ready: false,

  setEnvironment: ({ isAvailable, platform }) => set({ isAvailable, platform }),
  setReady: () => set({ ready: true }),
}));
