/** User entity — public surface. */
export type { User, RawUser, UserRole, Permission } from './types';
export { permissionsForRole, appModeForRole } from './permissions';
export { mapUser, toRole } from './mappers';
