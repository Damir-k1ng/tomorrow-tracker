/**
 * Route path constants — the single source of truth for URLs.
 * User routes live at the root; the entire Admin app is namespaced under /admin.
 */

export const userPaths = {
  dashboard: '/',
  sessions: '/sessions',
  leaderboard: '/leaderboard',
  streak: '/streak',
  profile: '/profile',
  settings: '/settings',
} as const;

export const adminPaths = {
  dashboard: '/admin',
  users: '/admin/users',
  sessions: '/admin/sessions',
  audit: '/admin/audit',
  exports: '/admin/exports',
  moderation: '/admin/moderation',
} as const;

export type UserPath = (typeof userPaths)[keyof typeof userPaths];
export type AdminPath = (typeof adminPaths)[keyof typeof adminPaths];
