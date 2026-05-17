import { motion } from 'framer-motion';
import { fadeInUp } from '@/design/motion';
import { cn } from '@/shared/lib/cn';

/**
 * Screen — the standard page container. Applies the page gutter, safe-area
 * top padding, bottom breathing room above the tab bar, and the subtle
 * fade-in-up entrance shared by every route.
 */
export function Screen({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <motion.div
      initial={fadeInUp.initial}
      animate={fadeInUp.animate}
      transition={fadeInUp.transition}
      className={cn('px-5 pb-10 pt-[max(var(--safe-top),1.5rem)]', className)}
    >
      {children}
    </motion.div>
  );
}
