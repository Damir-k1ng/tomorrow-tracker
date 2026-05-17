/**
 * Color tokens — JS-accessible mirror of the CSS custom properties defined in
 * styles/globals.css. Each value is a `var(--color-*)` reference, so there is
 * exactly ONE source of truth and theme switching (dark/light) just works.
 *
 * Prefer Tailwind utilities (bg-surface, text-muted, ...) in markup. Use this
 * map only where a raw color string is unavoidable (canvas, inline SVG fills,
 * Framer Motion color animations).
 */
export const colors = {
  background: 'var(--color-background)',
  surface: 'var(--color-surface)',
  surfaceRaised: 'var(--color-surface-raised)',
  border: 'var(--color-border)',
  borderStrong: 'var(--color-border-strong)',

  foreground: 'var(--color-foreground)',
  muted: 'var(--color-muted)',
  subtle: 'var(--color-subtle)',

  accent: 'var(--color-accent)',
  accentHover: 'var(--color-accent-hover)',
  accentForeground: 'var(--color-accent-foreground)',

  success: 'var(--color-success)',
  warning: 'var(--color-warning)',
  danger: 'var(--color-danger)',
} as const;

export type ColorToken = keyof typeof colors;
