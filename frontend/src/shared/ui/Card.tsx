import { forwardRef } from 'react';
import { cn } from '@/shared/lib/cn';

/**
 * Card — the standard content surface. Soft radius, hairline border, gentle
 * elevation. `interactive` adds press affordance for tappable cards.
 */
export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  interactive?: boolean;
}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, interactive, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'rounded-card border border-border bg-surface p-5 shadow-card',
        interactive &&
          'cursor-pointer transition-colors hover:border-border-strong active:bg-surface-raised',
        className,
      )}
      {...props}
    />
  ),
);
Card.displayName = 'Card';

export function CardHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('mb-3 flex items-center justify-between gap-3', className)} {...props} />;
}

export function CardTitle({ className, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3
      className={cn('text-sm font-medium tracking-tight text-muted', className)}
      {...props}
    />
  );
}

export function CardValue({ className, ...props }: React.HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p
      className={cn('text-3xl font-semibold tracking-tight text-foreground', className)}
      {...props}
    />
  );
}
