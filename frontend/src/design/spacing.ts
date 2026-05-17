/**
 * Spacing scale — a 4px base rhythm. Tailwind's numeric scale (p-4, gap-6, ...)
 * is the primary spacing API; this map exists for JS contexts and to give the
 * recurring layout values a single, named home.
 *
 * Do not invent ad-hoc pixel values in components — pick the nearest step.
 */
export const spacing = {
  /** Inline gap between tightly related items (icon ↔ label). */
  xs: '0.25rem', // 4px
  sm: '0.5rem', // 8px
  md: '0.75rem', // 12px
  lg: '1rem', // 16px
  xl: '1.5rem', // 24px
  '2xl': '2rem', // 32px
  '3xl': '3rem', // 48px

  /** Horizontal page gutter — mobile-first, comfortable at 320px. */
  pageGutter: '1.25rem', // 20px
  /** Vertical rhythm between major page sections. */
  section: '2rem', // 32px
} as const;

export type SpacingToken = keyof typeof spacing;
