import { NavLink } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import { cn } from '@/shared/lib/cn';
import { haptics } from '@/telegram/sdk';

export interface TabItem {
  to: string;
  label: string;
  icon: LucideIcon;
  /** `end` for the index route so it isn't active on every nested path. */
  end?: boolean;
}

/**
 * TabBar — the persistent bottom navigation. In-flow (not fixed) so it never
 * overlaps content; carries its own safe-area bottom padding for the home bar.
 */
export function TabBar({ items }: { items: TabItem[] }) {
  return (
    <nav className="shrink-0 border-t border-border bg-surface/95 pb-safe backdrop-blur">
      <ul className="flex">
        {items.map((item) => (
          <li key={item.to} className="min-w-0 flex-1">
            <NavLink
              to={item.to}
              end={item.end}
              onClick={() => haptics.select()}
              className="flex flex-col items-center gap-1 py-2.5"
            >
              {({ isActive }) => (
                <>
                  <item.icon
                    className={cn(
                      'size-5 transition-colors',
                      isActive ? 'text-accent' : 'text-subtle',
                    )}
                    aria-hidden
                  />
                  <span
                    className={cn(
                      'text-[0.625rem] font-medium tracking-tight transition-colors',
                      isActive ? 'text-accent' : 'text-subtle',
                    )}
                  >
                    {item.label}
                  </span>
                </>
              )}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}
