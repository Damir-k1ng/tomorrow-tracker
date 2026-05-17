/**
 * App-wide configuration constants.
 *
 * The API is same-origin (the Go binary embeds and serves this SPA), so the
 * base URL is a relative path — no cross-origin, no CORS.
 */

/** Base path for every backend call. */
export const API_BASE_URL = '/api/v1';

/** Per-request network timeout (ms). Telegram WebViews can stall — fail fast. */
export const API_TIMEOUT_MS = 12_000;

/** Retry budget for idempotent (GET) requests on transient failures. */
export const API_MAX_RETRIES = 2;

/** Base backoff between retries (ms); grows linearly with the attempt. */
export const API_RETRY_BACKOFF_MS = 400;

/** TanStack Query: how long fetched data stays fresh before a background refetch. */
export const QUERY_STALE_TIME_MS = 30_000;
