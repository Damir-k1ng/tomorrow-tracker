import { motion } from 'framer-motion';
import { GraduationCap } from 'lucide-react';
import { transition } from '@/design/motion';
import { Spinner } from '@/shared/ui';

/**
 * SplashScreen — shown immediately on launch and held for the whole auth
 * bootstrap. Nothing role-specific renders until bootstrap resolves, so this
 * is the only thing on screen while the role is still unknown.
 */
export function SplashScreen() {
  return (
    <div className="flex min-h-viewport flex-col items-center justify-center gap-7 bg-background px-8">
      <motion.div
        initial={{ opacity: 0, scale: 0.92 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={transition}
        className="flex size-16 items-center justify-center rounded-sheet bg-accent/15"
      >
        <GraduationCap className="size-8 text-accent" aria-hidden />
      </motion.div>

      <div className="text-center">
        <h1 className="text-lg font-semibold tracking-tight text-foreground">
          Tomorrow Tracker
        </h1>
        <p className="mt-1 text-sm text-muted">Загружаем приложение…</p>
      </div>

      <Spinner />
    </div>
  );
}
