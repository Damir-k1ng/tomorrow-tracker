/**
 * Border-radius tokens — JS mirror of the `--radius-*` custom properties in
 * styles/globals.css. Tailwind utilities `rounded-control`, `rounded-card`,
 * `rounded-sheet`, and `rounded-pill` are generated from the same @theme block.
 */
export const radius = {
  /** Buttons, inputs, small interactive controls. */
  control: 'var(--radius-control)',
  /** Cards and content surfaces. */
  card: 'var(--radius-card)',
  /** Bottom sheets / modals. */
  sheet: 'var(--radius-sheet)',
  /** Fully rounded — chips, avatars, pills. */
  pill: 'var(--radius-pill)',
} as const;

export type RadiusToken = keyof typeof radius;
