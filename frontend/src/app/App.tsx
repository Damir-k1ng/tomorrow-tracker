import { RootRouter } from '@/routes';
import { AppErrorBoundary } from './providers/AppErrorBoundary';
import { QueryProvider } from './providers/QueryProvider';
import { AuthBootstrap } from './bootstrap/AuthBootstrap';

/**
 * App — the composition root.
 *
 *   AppErrorBoundary  — last-resort crash guard
 *     QueryProvider   — server-state layer
 *       AuthBootstrap — splash → verify → role resolution
 *         RootRouter  — the exact role-aware route tree
 *
 * RootRouter is mounted strictly inside AuthBootstrap, so it renders only
 * after the backend has confirmed the user's role.
 */
export function App() {
  return (
    <AppErrorBoundary>
      <QueryProvider>
        <AuthBootstrap>
          <RootRouter />
        </AuthBootstrap>
      </QueryProvider>
    </AppErrorBoundary>
  );
}
