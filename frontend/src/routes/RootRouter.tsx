import { RouterProvider } from 'react-router-dom';
import { appModeForRole } from '@/entities/user';
import { useAuthStore } from '@/store';
import { adminRouter } from './adminRoutes';
import { userRouter } from './userRoutes';

/**
 * RootRouter — selects the exact route tree for the resolved role.
 *
 * Rendered ONLY after auth bootstrap reaches 'authenticated', so the role is
 * always known here. There is no user-UI-then-admin-UI swap: exactly one
 * router is ever mounted, eliminating flicker and unsafe admin rendering.
 */
export function RootRouter() {
  const role = useAuthStore((s) => s.user?.role ?? 'user');
  const router = appModeForRole(role) === 'admin' ? adminRouter : userRouter;
  return <RouterProvider router={router} />;
}
