import { forwardRef } from 'react';
import { motion } from 'framer-motion';
import { cva, type VariantProps } from 'class-variance-authority';
import { Loader2 } from 'lucide-react';
import { cn } from '@/shared/lib/cn';
import { transitionFast } from '@/design/motion';
import { haptics } from '@/telegram/sdk';

/**
 * Button — the primary interactive control. Tokenised variants only; no raw
 * colors. Press feedback is a subtle scale + a light haptic.
 */
const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 font-medium select-none ' +
    'rounded-control transition-colors disabled:opacity-50 ' +
    'disabled:pointer-events-none focus-visible:outline-none ' +
    'focus-visible:ring-2 focus-visible:ring-accent/60',
  {
    variants: {
      variant: {
        primary: 'bg-accent text-accent-foreground hover:bg-accent-hover',
        secondary:
          'bg-surface-raised text-foreground border border-border ' +
          'hover:border-border-strong',
        ghost: 'bg-transparent text-muted hover:text-foreground',
        danger: 'bg-danger text-accent-foreground hover:opacity-90',
      },
      size: {
        sm: 'h-9 px-3.5 text-sm',
        md: 'h-11 px-5 text-[0.95rem]',
        lg: 'h-13 px-6 text-base',
      },
      block: { true: 'w-full', false: '' },
    },
    defaultVariants: { variant: 'primary', size: 'md', block: false },
  },
);

export interface ButtonProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onAnimationStart' | 'onDragStart' | 'onDragEnd' | 'onDrag'>,
    VariantProps<typeof buttonVariants> {
  /** Show a spinner and block interaction. */
  loading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, block, loading, disabled, children, onClick, ...props }, ref) => {
    return (
      <motion.button
        ref={ref}
        whileTap={{ scale: 0.97 }}
        transition={transitionFast}
        className={cn(buttonVariants({ variant, size, block }), className)}
        disabled={disabled || loading}
        onClick={(e) => {
          haptics.impact('light');
          onClick?.(e);
        }}
        {...props}
      >
        {loading && <Loader2 className="size-4 animate-spin" aria-hidden />}
        {children}
      </motion.button>
    );
  },
);

Button.displayName = 'Button';
