import { Outlet } from 'react-router-dom';
import { Download, LayoutDashboard, ScrollText, ShieldAlert, Timer, Users } from 'lucide-react';
import { adminPaths } from '@/routes/paths';
import { OfflineNotice } from './OfflineNotice';
import { TabBar, type TabItem } from './TabBar';

/**
 * AdminLayout — the shell for the Admin App. Same structure as UserLayout but
 * with the admin navigation. Only ever mounted for a backend-confirmed admin.
 */
const adminTabs: TabItem[] = [
  { to: adminPaths.dashboard, label: 'Обзор', icon: LayoutDashboard, end: true },
  { to: adminPaths.users, label: 'Юзеры', icon: Users },
  { to: adminPaths.sessions, label: 'Сессии', icon: Timer },
  { to: adminPaths.audit, label: 'Аудит', icon: ScrollText },
  { to: adminPaths.exports, label: 'Экспорт', icon: Download },
  { to: adminPaths.moderation, label: 'Модерация', icon: ShieldAlert },
];

export function AdminLayout() {
  return (
    <div className="flex h-viewport flex-col bg-background">
      <OfflineNotice />
      <main className="no-scrollbar flex-1 overflow-y-auto">
        <Outlet />
      </main>
      <TabBar items={adminTabs} />
    </div>
  );
}
