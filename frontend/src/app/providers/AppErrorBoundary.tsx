import { Component, type ErrorInfo, type ReactNode } from 'react';
import { RotateCw, TriangleAlert } from 'lucide-react';
import { Button, EmptyState } from '@/shared/ui';

interface Props {
  children: ReactNode;
}
interface State {
  hasError: boolean;
}

/**
 * AppErrorBoundary — the global, last-resort error boundary.
 *
 * Catches render-time crashes anywhere below it (route boundaries handle
 * route-scoped errors first). Telegram WebViews are unstable, so an
 * unrecoverable crash must still show a calm, recoverable screen rather than a
 * blank WebView.
 *
 * SECURITY: only the error message/stack is logged — never request payloads,
 * never Telegram initData or any auth material.
 */
export class AppErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Minimal, non-sensitive diagnostics.
    console.error('[AppErrorBoundary]', error.message, info.componentStack);
  }

  render(): ReactNode {
    if (!this.state.hasError) return this.props.children;
    return (
      <div className="flex min-h-viewport items-center justify-center bg-background">
        <EmptyState
          icon={TriangleAlert}
          title="Приложение остановилось"
          description="Произошла непредвиденная ошибка. Перезагрузите приложение."
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
}
