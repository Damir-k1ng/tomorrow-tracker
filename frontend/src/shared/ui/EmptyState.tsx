import type { LucideIcon } from 'lucide-react';
import { cn } from '@/shared/lib/cn';

/**
 * EmptyState — the canonical "nothing here yet" / "no results" surface.
 * Also used as a lightweight placeholder for not-yet-built foundation screens.
 */
export interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 px-6 py-14 text-center',
        className,
      )}
    >
      <div className="flex size-14 items-center justify-center rounded-pill bg-surface-raised">
        <Icon className="size-6 text-subtle" aria-hidden />
      </div>
      <div className="space-y-1">
        <h3 className="text-base font-medium text-foreground">{title}</h3>
        {description && (
          <p className="mx-auto max-w-xs text-sm text-muted">{description}</p>
        )}
      </div>
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
