import { AnimatePresence, motion } from 'framer-motion';
import { WifiOff } from 'lucide-react';
import { transition } from '@/design/motion';
import { useOnlineStatus } from '@/shared/hooks/useOnlineStatus';

/**
 * OfflineNotice — a slim banner shown whenever connectivity drops. Telegram
 * WebViews lose the network often; this keeps the failure visible and calm
 * instead of letting requests fail silently.
 */
export function OfflineNotice() {
  const online = useOnlineStatus();
  return (
    <AnimatePresence>
      {!online && (
        <motion.div
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: 'auto', opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={transition}
          className="shrink-0 overflow-hidden bg-warning/15"
        >
          <div className="flex items-center justify-center gap-2 px-4 py-2 text-xs font-medium text-warning">
            <WifiOff className="size-3.5" aria-hidden />
            Нет подключения к интернету
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
