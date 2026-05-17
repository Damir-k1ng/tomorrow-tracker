import { Loader2 } from 'lucide-react';
import { cn } from '@/shared/lib/cn';

/** Spinner — minimal indeterminate indicator for inline / transient loads. */
export function Spinner({ className }: { className?: string }) {
  return (
    <Loader2
      className={cn('size-5 animate-spin text-muted', className)}
      aria-label="Загрузка"
    />
  );
}
