import { RotateCw, TriangleAlert } from 'lucide-react';
import { Button, EmptyState } from '@/shared/ui';
import { isApiError } from '@/services/api';

/**
 * QueryError — the inline error surface for a failed page data query.
 *
 * Unlike `RouteError` (which catches errors thrown during routing/render),
 * this renders *inside* a page whose React Query call failed, keeping the
 * page shell intact and offering a retry that re-runs only that query.
 */
export interface QueryErrorProps {
  error: unknown;
  onRetry: () => void;
  className?: string;
}

export function QueryError({ error, onRetry, className }: QueryErrorProps) {
  const message = isApiError(error) ? error.userMessage : 'Не удалось загрузить данные';

  return (
    <EmptyState
      className={className}
      icon={TriangleAlert}
      title="Не удалось загрузить"
      description={message}
      action={
        <Button variant="secondary" size="sm" onClick={onRetry}>
          <RotateCw className="size-4" aria-hidden />
          Повторить
        </Button>
      }
    />
  );
}
