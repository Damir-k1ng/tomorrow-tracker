import { useState } from 'react';
import { Database, FileSpreadsheet, TriangleAlert } from 'lucide-react';
import { PageHeader, Screen } from '@/widgets';
import { Button, Card } from '@/shared/ui';
import { adminApi, isApiError, type ExportRange } from '@/services/api';
import { downloadBlob } from '@/shared/lib/download';
import { haptics } from '@/telegram/sdk';

type ExportKind = 'users' | 'sessions';

/** Build a YYYY-MM-DD range covering the last `days` days. */
function lastDaysRange(days: number): ExportRange {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - days);
  const fmt = (d: Date) => d.toISOString().slice(0, 10);
  return { from: fmt(from), to: fmt(to) };
}

/**
 * Admin Exports — Phase 3A.
 *
 * Functional: the export is fully backend-generated CSV; the frontend only
 * triggers the download (no client-side CSV generation). The date range is a
 * fixed last-30-days for now; a range picker arrives with the full Admin app.
 */
export default function AdminExportsPage() {
  const [busy, setBusy] = useState<ExportKind | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function runExport(kind: ExportKind) {
    setBusy(kind);
    setError(null);
    const range = lastDaysRange(30);
    try {
      const blob =
        kind === 'users'
          ? await adminApi.exportUsers(range)
          : await adminApi.exportSessions(range);
      downloadBlob(blob, `${kind}-${range.from}_${range.to}.csv`);
      haptics.notify('success');
    } catch (err) {
      haptics.notify('error');
      setError(isApiError(err) ? err.userMessage : 'Не удалось выполнить экспорт');
    } finally {
      setBusy(null);
    }
  }

  return (
    <Screen>
      <PageHeader title="Экспорт" subtitle="Выгрузка данных за 30 дней" />

      <div className="space-y-3">
        <Card>
          <div className="flex items-center gap-3">
            <FileSpreadsheet className="size-5 text-accent" aria-hidden />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground">Пользователи</p>
              <p className="text-xs text-muted">CSV со списком пользователей</p>
            </div>
          </div>
          <Button
            variant="secondary"
            size="sm"
            block
            className="mt-4"
            loading={busy === 'users'}
            disabled={busy !== null}
            onClick={() => void runExport('users')}
          >
            Скачать CSV
          </Button>
        </Card>

        <Card>
          <div className="flex items-center gap-3">
            <Database className="size-5 text-accent" aria-hidden />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground">Сессии</p>
              <p className="text-xs text-muted">CSV со списком учебных сессий</p>
            </div>
          </div>
          <Button
            variant="secondary"
            size="sm"
            block
            className="mt-4"
            loading={busy === 'sessions'}
            disabled={busy !== null}
            onClick={() => void runExport('sessions')}
          >
            Скачать CSV
          </Button>
        </Card>
      </div>

      {error && (
        <div className="mt-4 flex items-center gap-2 rounded-control bg-danger/12 px-4 py-3 text-sm text-danger">
          <TriangleAlert className="size-4 shrink-0" aria-hidden />
          {error}
        </div>
      )}
    </Screen>
  );
}
