import { useState } from 'react';
import { QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { QUERY_STALE_TIME_MS } from '@/shared/config/app';
import { isApiError } from '@/services/api';
import { bootstrapAuth } from '@/features/auth';
import { useAuthStore } from '@/store';

/**
 * TanStack Query provider. All server data flows through Query — caching,
 * refetching, pagination — never through Zustand.
 *
 * Retry policy: the apiClient already retries transient failures, so Query
 * adds only one extra attempt and never retries client errors (4xx) or auth
 * failures. Window-focus refetch is off — Telegram WebView focus events are
 * noisy and unreliable.
 *
 * Auth-failure recovery: a query that 401s means the session is gone (initData
 * expired while the app stayed open, or was never valid). Re-running the
 * bootstrap either silently recovers the session or, when initData is truly
 * stale, flips the whole app to the full-screen AuthError — one clear "reopen
 * the app" prompt instead of a stuck screen behind a failed query.
 */
function createQueryClient(): QueryClient {
  return new QueryClient({
    queryCache: new QueryCache({
      onError: (error) => {
        if (!isApiError(error) || !error.isAuthFailure) return;
        // Act only from a healthy session: if status is already 'loading' or
        // 'error' the bootstrap is (re)running, so re-triggering would just
        // cause a retry storm.
        if (useAuthStore.getState().status !== 'authenticated') return;
        void bootstrapAuth();
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: QUERY_STALE_TIME_MS,
        refetchOnWindowFocus: false,
        retry: (failureCount, error) => {
          if (isApiError(error)) return error.isRetryable && failureCount < 1;
          return false;
        },
      },
      mutations: { retry: false },
    },
  });
}

export function QueryProvider({ children }: { children: React.ReactNode }) {
  // One client per app instance, kept stable across renders.
  const [client] = useState(createQueryClient);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
