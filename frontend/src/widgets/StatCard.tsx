import type { LucideIcon } from 'lucide-react';
import { Card } from '@/shared/ui';
import { cn } from '@/shared/lib/cn';

/**
 * StatCard — a single dashboard metric tile. A solid (matte) card with an
 * iconic accent chip, a prominent value, and an optional hint.
 */
export interface StatCardProps {
  icon: LucideIcon;
  label: string;
  value: React.ReactNode;
  hint?: string;
  className?: string;
}

export function StatCard({ icon: Icon, label, value, hint, className }: StatCardProps) {
  return (
    <Card className={cn('p-4', className)}>
      <div className="flex items-center gap-2">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-[0.6rem] bg-surface-raised">
          <Icon className="size-3.5 text-accent" aria-hidden />
        </span>
        <span className="truncate text-xs font-medium tracking-tight text-muted">{label}</span>
      </div>
      <p className="mt-3 text-[1.6rem] font-semibold leading-none tracking-tight text-foreground">
        {value}
      </p>
      {hint && <p className="mt-1.5 text-xs text-subtle">{hint}</p>}
    </Card>
  );
}
