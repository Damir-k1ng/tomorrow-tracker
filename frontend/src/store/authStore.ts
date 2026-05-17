import { create } from 'zustand';
import { permissionsForRole, type Permission, type User } from '@/entities/user';
import type { ApiError } from '@/services/api';

/**
 * Auth store — the resolved identity for the session.
 *
 * Populated exactly once during bootstrap from the backend's verify response.
 * The route tree (User vs Admin) is chosen from `user.role` only AFTER status
 * becomes 'authenticated', so no UI ever renders against an unknown role.
 *
 * initData is NEVER stored here or anywhere persistent — only the resolved,
 * non-sensitive user profile lives in memory.
 */
export type AuthStatus = 'idle' | 'loading' | 'authenticated' | 'error';

interface AuthState {
  status: AuthStatus;
  user: User | null;
  permissions: Permission[];
  error: ApiError | null;

  startLoading: () => void;
  authenticate: (user: User) => void;
  fail: (error: ApiError) => void;
  reset: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  status: 'idle',
  user: null,
  permissions: [],
  error: null,

  startLoading: () => set({ status: 'loading', error: null }),
  authenticate: (user) =>
    set({
      status: 'authenticated',
      user,
      permissions: permissionsForRole(user.role),
      error: null,
    }),
  fail: (error) => set({ status: 'error', error }),
  reset: () => set({ status: 'idle', user: null, permissions: [], error: null }),
}));

/** Reactive permission check — for conditionally rendering affordances. */
export function useHasPermission(permission: Permission): boolean {
  return useAuthStore((s) => s.permissions.includes(permission));
}
