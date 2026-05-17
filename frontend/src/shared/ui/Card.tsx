import { forwardRef } from 'react';
import { cn } from '@/shared/lib/cn';

/**
 * Card — the standard content surface. Soft radius, hairline border, gentle
 * elevation. `interactive` adds press affordance for tappable cards.
 *
 * `glass` renders the card as a Liquid Glass surface — reserved for hero
 * surfaces (the session control, weekly progress). Plain content cards stay
 * solid so the UI keeps strong contrast and stays fast on low-end devices.
 */
export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  interactive?: boolean;
  glass?: boolean;
}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, interactive, glass, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'rounded-card p-5',
        glass
          ? 'glass shadow-[0_10px_32px_oklch(0%_0_0/0.42)]'
          : 'border border-border bg-surface shadow-card',
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
