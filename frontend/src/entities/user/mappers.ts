import type { RawUser, User, UserRole } from './types';

const KNOWN_ROLES: readonly UserRole[] = ['user', 'moderator', 'admin'];

/** Coerce an arbitrary backend role string into a known UserRole. */
export function toRole(raw: string): UserRole {
  return (KNOWN_ROLES as readonly string[]).includes(raw) ? (raw as UserRole) : 'user';
}

/** Map the backend user DTO (snake_case) into the app's `User` model. */
export function mapUser(raw: RawUser): User {
  return {
    id: raw.id,
    telegramId: raw.telegram_id,
    firstName: raw.first_name,
    username: raw.username,
    role: toRole(raw.role),
  };
}
