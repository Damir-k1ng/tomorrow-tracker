import { createBrowserRouter, Navigate } from 'react-router-dom';
import { RouteError, UserLayout } from '@/widgets';
import { lazyPage } from './lazyPage';
import { userPaths } from './paths';

/**
 * User App route tree. Mounted only after the backend confirms the role is NOT
 * admin. Every page is its own lazily-loaded chunk.
 */
export const userRouter = createBrowserRouter([
  {
    element: <UserLayout />,
    errorElement: <RouteError />,
    children: [
      { index: true, element: lazyPage(() => import('@/pages/user/DashboardPage')) },
      { path: userPaths.sessions, element: lazyPage(() => import('@/pages/user/SessionsPage')) },
      {
        path: userPaths.leaderboard,
        element: lazyPage(() => import('@/pages/user/LeaderboardPage')),
      },
      { path: userPaths.streak, element: lazyPage(() => import('@/pages/user/StreakPage')) },
      { path: userPaths.profile, element: lazyPage(() => import('@/pages/user/ProfilePage')) },
      { path: userPaths.settings, element: lazyPage(() => import('@/pages/user/SettingsPage')) },
      // Unknown paths fall back to the dashboard.
      { path: '*', element: <Navigate to={userPaths.dashboard} replace /> },
    ],
  },
]);
