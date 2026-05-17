import { Navigate } from 'react-router-dom';
import { useAuthStore } from '@/store';

/**
 * RequireAdmin — defence-in-depth route guard.
 *
 * The Admin router is only ever mounted for a backend-confirmed admin (see
 * RootRouter), so in practice this never redirects. It exists so that no admin
 * route element can render without `role === 'admin'`, even if the route tree
 * is ever wired differently. The backend remains the real authority.
 */
export function RequireAdmin({ children }: { children: React.ReactNode }) {
  const role = useAuthStore((s) => s.user?.role);
  if (role !== 'admin') {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
