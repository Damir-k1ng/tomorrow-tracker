import type { Permission, UserRole } from './types';

/**
 * Role → permission mapping.
 *
 * NOTE: the backend currently returns only `role`, not an explicit permission
 * list. Permissions are derived here from the *backend-confirmed* role, so
 * they are still trustworthy as a UI gate. If the backend later returns
 * permissions directly, this map becomes the fallback. The backend always
 * remains the real enforcement authority — this never grants access, it only
 * decides which affordances to render.
 */
const ROLE_PERMISSIONS: Record<UserRole, readonly Permission[]> = {
  user: ['view_user_app'],
  // Phase 3A is binary (User App / Admin App). The moderator role exists in
  // the backend but is treated as a regular user here until its own UI lands.
  moderator: ['view_user_app'],
  admin: [
    'view_user_app',
    'view_admin_app',
    'manage_users',
    'correct_sessions',
    'view_audit',
    'export_data',
  ],
};

/** Resolve the permission set for a role (unknown roles → minimal user set). */
export function permissionsForRole(role: UserRole): Permission[] {
  return [...(ROLE_PERMISSIONS[role] ?? ROLE_PERMISSIONS.user)];
}

/** Which app mode a role mounts. Only `admin` enters the Admin App. */
export function appModeForRole(role: UserRole): 'admin' | 'user' {
  return role === 'admin' ? 'admin' : 'user';
}
