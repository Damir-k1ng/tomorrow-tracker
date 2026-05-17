/**
 * Telegram SDK wrapper — the ONLY module allowed to touch window.Telegram.
 *
 * Everything else (components, stores, services) goes through this façade, so
 * the Telegram dependency is isolated, mockable, and degrades gracefully when
 * the app runs outside Telegram (local browser dev).
 *
 * Responsibilities: lifecycle (ready/expand/close), theme, viewport + safe
 * area → CSS variables, haptics, BackButton, MainButton.
 */
import type {
  HapticImpactStyle,
  HapticNotificationType,
  TelegramColorScheme,
  TelegramUser,
  TelegramWebApp,
} from './types';

function getWebApp(): TelegramWebApp | undefined {
  return typeof window !== 'undefined' ? window.Telegram?.WebApp : undefined;
}

/** True when the app is actually running inside the Telegram client. */
export function isTelegramAvailable(): boolean {
  const wa = getWebApp();
  return !!wa && typeof wa.initData === 'string' && wa.initData.length > 0;
}

/**
 * Raw initData for backend verification. In Telegram this is the signed
 * payload; in local dev it falls back to VITE_DEV_INIT_DATA so the auth flow
 * can be exercised without the Telegram client. It is NEVER persisted.
 */
export function getInitData(): string {
  const wa = getWebApp();
  if (wa?.initData) return wa.initData;
  if (import.meta.env.DEV) return import.meta.env.VITE_DEV_INIT_DATA ?? '';
  return '';
}

/** Unverified Telegram user — for optimistic UI only; the backend is authority. */
export function getUnsafeUser(): TelegramUser | undefined {
  return getWebApp()?.initDataUnsafe.user;
}

/** Current Telegram color scheme; defaults to dark when unavailable. */
export function getColorScheme(): TelegramColorScheme {
  return getWebApp()?.colorScheme ?? 'dark';
}

/** Telegram client platform identifier ('ios', 'android', 'tdesktop', ...). */
export function getPlatform(): string {
  return getWebApp()?.platform ?? 'unknown';
}

// --- CSS variable sync -------------------------------------------------------

function syncViewport(wa: TelegramWebApp): void {
  const root = document.documentElement;
  // viewportStableHeight excludes the keyboard, avoiding layout jumps.
  root.style.setProperty('--tg-viewport-height', `${wa.viewportStableHeight}px`);
}

function syncSafeArea(wa: TelegramWebApp): void {
  const root = document.documentElement;
  const device = wa.safeAreaInset;
  const content = wa.contentSafeAreaInset;
  if (!device && !content) return; // keep the env(safe-area-inset-*) fallback

  const top = (device?.top ?? 0) + (content?.top ?? 0);
  const bottom = (device?.bottom ?? 0) + (content?.bottom ?? 0);
  root.style.setProperty('--safe-top', `${top}px`);
  root.style.setProperty('--safe-bottom', `${bottom}px`);
  root.style.setProperty('--safe-left', `${device?.left ?? 0}px`);
  root.style.setProperty('--safe-right', `${device?.right ?? 0}px`);
}

// applyDarkTheme pins the document to the dark theme. This is a dark-only
// build: the Liquid Glass design commits to one designed dark theme and
// deliberately does not follow the Telegram light scheme.
function applyDarkTheme(): void {
  const root = document.documentElement;
  root.classList.add('dark');
  root.classList.remove('light');
}

// --- Lifecycle ---------------------------------------------------------------

let initialised = false;

/**
 * Initialise the Telegram integration. Idempotent — safe to call once at
 * bootstrap. Outside Telegram it just applies the dark theme and returns.
 */
export function initTelegram(onThemeChange?: (s: TelegramColorScheme) => void): void {
  if (initialised) return;
  initialised = true;

  applyDarkTheme();

  const wa = getWebApp();
  if (!wa) {
    return;
  }

  wa.ready();
  wa.expand();
  // Prevent an accidental swipe-down from closing the app mid-interaction.
  wa.disableVerticalSwipes?.();

  syncViewport(wa);
  syncSafeArea(wa);

  if (wa.isVersionAtLeast('6.1')) {
    wa.setHeaderColor?.('#0a0a0b');
    wa.setBackgroundColor?.('#0a0a0b');
  }

  // The build is dark-only, so a Telegram theme change never re-themes the
  // app; onThemeChange is still notified for any non-visual consumers.
  wa.onEvent('themeChanged', () => {
    onThemeChange?.(wa.colorScheme);
  });
  wa.onEvent('viewportChanged', () => syncViewport(wa));
  wa.onEvent('safeAreaChanged', () => syncSafeArea(wa));
  wa.onEvent('contentSafeAreaChanged', () => syncSafeArea(wa));
}

/** Close the Mini App. */
export function closeApp(): void {
  getWebApp()?.close();
}

// --- Haptics (no-op outside Telegram) ---------------------------------------

export const haptics = {
  impact(style: HapticImpactStyle = 'light'): void {
    getWebApp()?.HapticFeedback?.impactOccurred(style);
  },
  notify(type: HapticNotificationType): void {
    getWebApp()?.HapticFeedback?.notificationOccurred(type);
  },
  select(): void {
    getWebApp()?.HapticFeedback?.selectionChanged();
  },
};

// --- Back button -------------------------------------------------------------

let backHandler: (() => void) | null = null;

/** Show the Telegram BackButton and route taps to `onClick`. */
export function showBackButton(onClick: () => void): void {
  const wa = getWebApp();
  if (!wa) return;
  if (backHandler) wa.BackButton.offClick(backHandler);
  backHandler = onClick;
  wa.BackButton.onClick(onClick);
  wa.BackButton.show();
}

/** Hide the Telegram BackButton and detach its handler. */
export function hideBackButton(): void {
  const wa = getWebApp();
  if (!wa) return;
  if (backHandler) {
    wa.BackButton.offClick(backHandler);
    backHandler = null;
  }
  wa.BackButton.hide();
}

// --- Main button -------------------------------------------------------------

let mainHandler: (() => void) | null = null;

export interface MainButtonConfig {
  text: string;
  onClick: () => void;
  loading?: boolean;
  disabled?: boolean;
}

/** Configure and show the Telegram MainButton. */
export function setMainButton(config: MainButtonConfig): void {
  const wa = getWebApp();
  if (!wa) return;
  const { MainButton } = wa;

  if (mainHandler) MainButton.offClick(mainHandler);
  mainHandler = config.onClick;
  MainButton.onClick(mainHandler);

  MainButton.setText(config.text);
  if (config.disabled) MainButton.disable();
  else MainButton.enable();
  if (config.loading) MainButton.showProgress();
  else MainButton.hideProgress();
  MainButton.show();
}

/** Hide the Telegram MainButton and detach its handler. */
export function hideMainButton(): void {
  const wa = getWebApp();
  if (!wa) return;
  if (mainHandler) {
    wa.MainButton.offClick(mainHandler);
    mainHandler = null;
  }
  wa.MainButton.hideProgress();
  wa.MainButton.hide();
}
