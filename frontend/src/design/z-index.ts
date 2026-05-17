/**
 * z-index scale — a single ordered ladder so stacking is never guessed.
 * Every layered surface must pick a name from here, never a raw number.
 */
export const zIndex = {
  base: 0,
  /** Sticky in-page elements (section headers). */
  sticky: 10,
  /** Bottom tab bar / persistent navigation. */
  navigation: 20,
  /** Dimming overlay behind sheets and modals. */
  overlay: 30,
  /** Bottom sheets, dialogs, modals. */
  modal: 40,
  /** Transient toasts / notifications. */
  toast: 50,
  /** Full-screen splash + global error boundary — always on top. */
  splash: 60,
} as const;

export type ZIndexToken = keyof typeof zIndex;
