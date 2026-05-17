/**
 * apiClient — the single fetch wrapper for the whole app.
 *
 * No component or feature calls `fetch` directly. This module centralises:
 *   - the Telegram initData auth header,
 *   - request timeouts (AbortController),
 *   - retry-with-backoff for idempotent GETs,
 *   - offline short-circuiting,
 *   - the {success,data,error} envelope unwrap,
 *   - normalisation of every failure into a typed ApiError.
 */
import {
  API_BASE_URL,
  API_MAX_RETRIES,
  API_RETRY_BACKOFF_MS,
  API_TIMEOUT_MS,
} from '@/shared/config/app';
import { getInitData } from '@/telegram/sdk';
import { ApiError, kindFromStatus } from './errors';

type HttpMethod = 'GET' | 'POST' | 'PATCH';

export interface RequestOptions {
  method?: HttpMethod;
  body?: unknown;
  signal?: AbortSignal;
  timeoutMs?: number;
  /** Retry budget; defaults to API_MAX_RETRIES for GET, 0 otherwise. */
  retries?: number;
}

/** Backend response envelope — exactly one of data/error is meaningful. */
interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: string;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** Build the auth header. Telegram initData is sent on every request. */
function authHeaders(): Record<string, string> {
  const initData = getInitData();
  return initData ? { Authorization: `tma ${initData}` } : {};
}

/** One network attempt — no retry logic here. */
async function attempt<T>(path: string, method: HttpMethod, body: unknown, opts: RequestOptions): Promise<T> {
  if (typeof navigator !== 'undefined' && !navigator.onLine) {
    throw new ApiError('offline');
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), opts.timeoutMs ?? API_TIMEOUT_MS);
  // Honour a caller-supplied signal alongside the timeout.
  const onExternalAbort = () => controller.abort();
  opts.signal?.addEventListener('abort', onExternalAbort);

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers: {
        Accept: 'application/json',
        ...authHeaders(),
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    });
  } catch (err) {
    // AbortError → distinguish caller cancel from our timeout.
    if (err instanceof DOMException && err.name === 'AbortError') {
      if (opts.signal?.aborted) throw err; // genuine caller cancellation
      throw new ApiError('timeout');
    }
    throw new ApiError('network');
  } finally {
    clearTimeout(timeout);
    opts.signal?.removeEventListener('abort', onExternalAbort);
  }

  let envelope: Envelope<T> | null = null;
  try {
    envelope = (await response.json()) as Envelope<T>;
  } catch {
    envelope = null; // non-JSON body (proxy error page, empty 5xx, ...)
  }

  if (!response.ok) {
    throw new ApiError(kindFromStatus(response.status), {
      status: response.status,
      serverMessage: envelope?.error,
    });
  }
  if (!envelope || envelope.success !== true) {
    throw new ApiError(envelope ? 'http' : 'parse', {
      status: response.status,
      serverMessage: envelope?.error,
    });
  }
  return envelope.data as T;
}

/** Core request with retry-with-backoff for transient failures. */
async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const method = opts.method ?? 'GET';
  const maxRetries = opts.retries ?? (method === 'GET' ? API_MAX_RETRIES : 0);

  let lastError: unknown;
  for (let i = 0; i <= maxRetries; i++) {
    try {
      return await attempt<T>(path, method, opts.body, opts);
    } catch (err) {
      lastError = err;
      const retryable = err instanceof ApiError && err.isRetryable;
      if (!retryable || i === maxRetries) break;
      await sleep(API_RETRY_BACKOFF_MS * (i + 1));
    }
  }
  throw lastError;
}

export const apiClient = {
  get: <T>(path: string, opts?: RequestOptions) => request<T>(path, { ...opts, method: 'GET' }),
  post: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { ...opts, method: 'POST', body }),
  patch: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { ...opts, method: 'PATCH', body }),

  /**
   * Fetch a binary payload (CSV exports). The backend generates the file;
   * the frontend only streams the bytes — no client-side CSV generation.
   */
  async download(path: string, opts?: RequestOptions): Promise<Blob> {
    if (typeof navigator !== 'undefined' && !navigator.onLine) {
      throw new ApiError('offline');
    }
    const controller = new AbortController();
    const timeout = setTimeout(
      () => controller.abort(),
      opts?.timeoutMs ?? API_TIMEOUT_MS * 3, // exports are slower
    );
    try {
      const res = await fetch(`${API_BASE_URL}${path}`, {
        method: 'GET',
        headers: { ...authHeaders() },
        signal: controller.signal,
      });
      if (!res.ok) {
        throw new ApiError(kindFromStatus(res.status), { status: res.status });
      }
      return await res.blob();
    } catch (err) {
      if (err instanceof ApiError) throw err;
      if (err instanceof DOMException && err.name === 'AbortError') {
        throw new ApiError('timeout');
      }
      throw new ApiError('network');
    } finally {
      clearTimeout(timeout);
    }
  },
};
