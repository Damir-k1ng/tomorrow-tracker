import { NavLink } from 'react-router-dom';
import { motion } from 'framer-motion';
import type { LucideIcon } from 'lucide-react';
import { transition } from '@/design/motion';
import { zIndex } from '@/design/z-index';
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
 * TabBar — the floating Liquid Glass navigation bar.
 *
 * A translucent glass pill detached from the screen edges; content scrolls
 * underneath and is blurred through it. The active tab is marked by an accent
 * capsule that morphs fluidly between tabs via a shared `layoutId`.
 *
 * It overlays the content area (absolute), so the layout reserves bottom
 * padding to keep the last content clear of the bar.
 */
export function TabBar({ items }: { items: TabItem[] }) {
  return (
    <nav
      className="pointer-events-none absolute inset-x-0 bottom-0 flex justify-center px-3"
      style={{
        zIndex: zIndex.navigation,
        paddingBottom: 'max(var(--safe-bottom), 0.75rem)',
      }}
    >
      <ul
        className={cn(
          'glass pointer-events-auto flex w-full max-w-md gap-1 rounded-[1.75rem] p-1.5',
          'shadow-[0_12px_40px_oklch(0%_0_0/0.5)]',
        )}
      >
        {items.map((item) => (
          <li key={item.to} className="min-w-0 flex-1">
            <NavLink
              to={item.to}
              end={item.end}
              onClick={() => haptics.select()}
              className="relative flex flex-col items-center gap-1 rounded-[1.4rem] px-0.5 py-2"
            >
              {({ isActive }) => (
                <>
                  {isActive && (
                    <motion.span
                      layoutId="tab-active-capsule"
                      transition={transition}
                      className="absolute inset-0 rounded-[1.4rem] bg-accent/16 ring-1 ring-inset ring-accent/30"
                    />
                  )}
                  <item.icon
                    className={cn(
                      'relative size-5 transition-colors',
                      isActive ? 'text-accent' : 'text-subtle',
                    )}
                    aria-hidden
                  />
                  <span
                    className={cn(
                      'relative truncate text-[0.6rem] font-medium tracking-tight transition-colors',
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
