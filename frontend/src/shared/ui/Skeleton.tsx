import { cn } from '@/shared/lib/cn';

/**
 * Skeleton — a calm loading placeholder. Used in place of spinners for content
 * regions so the layout never collapses or jumps while data loads.
 */
export function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('animate-pulse rounded-control bg-surface-raised', className)}
      {...props}
    />
  );
}
