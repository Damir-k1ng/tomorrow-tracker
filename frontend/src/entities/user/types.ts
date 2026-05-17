/** User domain model — the identity resolved by the backend at bootstrap. */

export type UserRole = 'user' | 'moderator' | 'admin';

/**
 * Capabilities the UI may surface. Derived from the backend-confirmed role —
 * see permissions.ts. The backend remains the enforcement authority; these
 * gate affordances only.
 */
export type Permission =
  | 'view_user_app'
  | 'view_admin_app'
  | 'manage_users'
  | 'correct_sessions'
  | 'view_audit'
  | 'export_data';

/** The authenticated user, in app-internal (camelCase) shape. */
export interface User {
  id: number;
  telegramId: number;
  firstName: string;
  username: string;
  role: UserRole;
}

/**
 * Raw user DTO as returned by the Go backend (POST /api/v1/auth/verify).
 * snake_case — mapped to `User` by entities/user/mappers.
 */
export interface RawUser {
  id: number;
  telegram_id: number;
  first_name: string;
  username: string;
  role: string;
}
