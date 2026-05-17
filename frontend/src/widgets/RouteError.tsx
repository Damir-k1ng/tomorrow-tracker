import { useRouteError } from 'react-router-dom';
import { RotateCw, TriangleAlert } from 'lucide-react';
import { Button, EmptyState } from '@/shared/ui';
import { isApiError } from '@/services/api';

/**
 * RouteError — the per-route error boundary element. A thrown error in any
 * route (including a failed lazy chunk load) is caught here, so one broken
 * screen never takes down the whole app.
 */
export function RouteError() {
  const error = useRouteError();
  const message = isApiError(error)
    ? error.userMessage
    : 'Не удалось загрузить экран';

  return (
    <div className="flex min-h-viewport items-center justify-center">
      <EmptyState
        icon={TriangleAlert}
        title="Что-то пошло не так"
        description={message}
        action={
          <Button variant="secondary" size="sm" onClick={() => window.location.reload()}>
            <RotateCw className="size-4" aria-hidden />
            Перезагрузить
          </Button>
        }
      />
    </div>
  );
}
