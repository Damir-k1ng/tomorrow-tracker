import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { QUERY_STALE_TIME_MS } from '@/shared/config/app';
import { isApiError } from '@/services/api';

/**
 * TanStack Query provider. All server data flows through Query — caching,
 * refetching, pagination — never through Zustand.
 *
 * Retry policy: the apiClient already retries transient failures, so Query
 * adds only one extra attempt and never retries client errors (4xx) or auth
 * failures. Window-focus refetch is off — Telegram WebView focus events are
 * noisy and unreliable.
 */
function createQueryClient(): QueryClient {
  return new QueryClient({
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
