import { createBrowserRouter, Navigate } from 'react-router-dom';
import { AdminLayout, RouteError } from '@/widgets';
import { RequireAdmin } from './guards';
import { lazyPage } from './lazyPage';
import { adminPaths, adminUserDetailPattern } from './paths';

/**
 * Admin App route tree. Mounted only when the backend confirms `role === admin`
 * (see RootRouter); RequireAdmin is the defence-in-depth second check. Every
 * page is a separate lazily-loaded chunk, so admin code is never shipped to
 * non-admin users.
 */
export const adminRouter = createBrowserRouter([
  {
    element: (
      <RequireAdmin>
        <AdminLayout />
      </RequireAdmin>
    ),
    errorElement: <RouteError />,
    children: [
      { path: adminPaths.dashboard, element: lazyPage(() => import('@/pages/admin/AdminDashboardPage')) },
      { path: adminPaths.users, element: lazyPage(() => import('@/pages/admin/AdminUsersPage')) },
      {
        path: adminUserDetailPattern,
        element: lazyPage(() => import('@/pages/admin/AdminUserDetailPage')),
      },
      { path: adminPaths.sessions, element: lazyPage(() => import('@/pages/admin/AdminSessionsPage')) },
      { path: adminPaths.audit, element: lazyPage(() => import('@/pages/admin/AdminAuditPage')) },
      { path: adminPaths.exports, element: lazyPage(() => import('@/pages/admin/AdminExportsPage')) },
      {
        path: adminPaths.moderation,
        element: lazyPage(() => import('@/pages/admin/AdminModerationPage')),
      },
      // The app launches at "/" — send admins to their dashboard. Any unknown
      // path resolves there too.
      { path: '*', element: <Navigate to={adminPaths.dashboard} replace /> },
    ],
  },
]);
