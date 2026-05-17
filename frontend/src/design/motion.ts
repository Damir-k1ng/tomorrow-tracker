/**
 * Motion tokens for Framer Motion.
 *
 * Philosophy: motion clarifies flow, it never decorates. Transitions are short
 * and subtle — no bounce, no spring overshoot, no looping ambient animation.
 *
 * `easeStandard` mirrors the `--ease-standard` CSS variable. Components import
 * these presets instead of writing inline duration/easing literals.
 */

/** Durations in seconds (Framer Motion's unit). */
export const duration = {
  fast: 0.15,
  normal: 0.22,
  slow: 0.32,
} as const;

/** A calm, decelerating cubic-bezier — the single house easing curve. */
export const easeStandard = [0.32, 0.72, 0, 1] as const;

/** Standard transition for most enter/exit and layout changes. */
export const transition = {
  duration: duration.normal,
  ease: easeStandard,
} as const;

/** A faster transition for press feedback and micro-interactions. */
export const transitionFast = {
  duration: duration.fast,
  ease: easeStandard,
} as const;

/** Fade + slight rise — the default page / panel enter animation. */
export const fadeInUp = {
  initial: { opacity: 0, y: 8 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: 8 },
  transition,
} as const;

/** Plain cross-fade — for content that should not move. */
export const fade = {
  initial: { opacity: 0 },
  animate: { opacity: 1 },
  exit: { opacity: 0 },
  transition,
} as const;
