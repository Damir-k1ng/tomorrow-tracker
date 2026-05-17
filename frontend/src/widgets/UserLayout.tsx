import { Outlet } from 'react-router-dom';
import { Flame, House, Timer, Trophy, User } from 'lucide-react';
import { userPaths } from '@/routes/paths';
import { OfflineNotice } from './OfflineNotice';
import { TabBar, type TabItem } from './TabBar';

/**
 * UserLayout — the shell for the User App: a scrollable content area above a
 * persistent bottom tab bar. Settings is reachable from the Profile screen, so
 * it is intentionally not a tab.
 */
const userTabs: TabItem[] = [
  { to: userPaths.dashboard, label: 'Главная', icon: House, end: true },
  { to: userPaths.sessions, label: 'Сессии', icon: Timer },
  { to: userPaths.leaderboard, label: 'Рейтинг', icon: Trophy },
  { to: userPaths.streak, label: 'Серия', icon: Flame },
  { to: userPaths.profile, label: 'Профиль', icon: User },
];

export function UserLayout() {
  return (
    <div className="flex h-viewport flex-col bg-background">
      <OfflineNotice />
      <main className="no-scrollbar flex-1 overflow-y-auto">
        <Outlet />
      </main>
      <TabBar items={userTabs} />
    </div>
  );
}
