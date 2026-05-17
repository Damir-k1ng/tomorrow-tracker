import { Button } from '@/shared/ui';

/**
 * Pager — the shared prev/next pagination control for admin and user list
 * screens. Buttons are disabled at the bounds and while a fetch is in flight.
 */
export interface PagerProps {
  page: number;
  totalPages: number;
  busy?: boolean;
  onChange: (page: number) => void;
}

export function Pager({ page, totalPages, busy = false, onChange }: PagerProps) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <Button
        variant="secondary"
        size="sm"
        disabled={busy || page <= 1}
        onClick={() => onChange(page - 1)}
      >
        Назад
      </Button>
      <span className="text-xs text-muted">
        Стр. {page} из {totalPages}
      </span>
      <Button
        variant="secondary"
        size="sm"
        disabled={busy || page >= totalPages}
        onClick={() => onChange(page + 1)}
      >
        Вперёд
      </Button>
    </div>
  );
}
