import { RotateCw, ShieldX, WifiOff } from 'lucide-react';
import { Button, EmptyState } from '@/shared/ui';
import { useAuthStore } from '@/store';

/**
 * AuthError — full-screen failure state for the auth bootstrap. Shown instead
 * of the app when verification fails, so no app tree mounts without a
 * confirmed identity. Offers a retry that re-runs the bootstrap.
 */
export function AuthError({ onRetry }: { onRetry: () => void }) {
  const error = useAuthStore((s) => s.error);
  const isAuthFailure = error?.isAuthFailure ?? false;

  return (
    <div className="flex min-h-viewport items-center justify-center bg-background">
      <EmptyState
        icon={isAuthFailure ? ShieldX : WifiOff}
        title={isAuthFailure ? 'Сессия недоступна' : 'Не удалось подключиться'}
        description={
          error?.userMessage ??
          'Не удалось запустить приложение. Попробуйте ещё раз.'
        }
        action={
          <Button variant="secondary" size="sm" onClick={onRetry}>
            <RotateCw className="size-4" aria-hidden />
            Повторить
          </Button>
        }
      />
    </div>
  );
}
