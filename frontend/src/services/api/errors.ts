/**
 * Typed API errors. Every failure from the API layer is normalised into an
 * `ApiError` with a discriminating `kind`, so callers and error boundaries can
 * branch on a stable enum instead of inspecting raw fetch/HTTP details.
 */

export type ApiErrorKind =
  | 'offline' // no network connectivity
  | 'timeout' // request exceeded the deadline
  | 'network' // fetch failed (DNS, connection, CORS, WebView drop)
  | 'unauthorized' // 401 — initData invalid or expired
  | 'forbidden' // 403 — authenticated but not allowed
  | 'not_found' // 404
  | 'rate_limited' // 429
  | 'server' // 5xx
  | 'http' // other non-2xx / envelope success:false
  | 'parse'; // malformed response body

/** Russian, user-facing fallback messages keyed by error kind. */
const USER_MESSAGES: Record<ApiErrorKind, string> = {
  offline: 'Нет подключения к интернету',
  timeout: 'Превышено время ожидания. Попробуйте ещё раз',
  network: 'Ошибка сети. Проверьте подключение',
  unauthorized: 'Сессия устарела. Откройте приложение заново',
  forbidden: 'Недостаточно прав для этого действия',
  not_found: 'Данные не найдены',
  rate_limited: 'Слишком много запросов. Подождите немного',
  server: 'Ошибка сервера. Попробуйте позже',
  http: 'Что-то пошло не так',
  parse: 'Не удалось обработать ответ сервера',
};

export class ApiError extends Error {
  readonly kind: ApiErrorKind;
  /** HTTP status, or 0 when the request never produced a response. */
  readonly status: number;
  /** Message safe to show to the user (Russian). */
  readonly userMessage: string;

  constructor(kind: ApiErrorKind, opts: { status?: number; serverMessage?: string } = {}) {
    const userMessage = opts.serverMessage?.trim() || USER_MESSAGES[kind];
    super(`[${kind}] ${userMessage}`);
    this.name = 'ApiError';
    this.kind = kind;
    this.status = opts.status ?? 0;
    this.userMessage = userMessage;
  }

  /** Transient failures that are safe to retry for idempotent requests. */
  get isRetryable(): boolean {
    return this.kind === 'network' || this.kind === 'timeout' || this.kind === 'server';
  }

  /** The session is no longer valid — the app should re-bootstrap. */
  get isAuthFailure(): boolean {
    return this.kind === 'unauthorized';
  }
}

/** Narrowing helper. */
export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError;
}

/** Map an HTTP status to the corresponding error kind. */
export function kindFromStatus(status: number): ApiErrorKind {
  if (status === 401) return 'unauthorized';
  if (status === 403) return 'forbidden';
  if (status === 404) return 'not_found';
  if (status === 429) return 'rate_limited';
  if (status >= 500) return 'server';
  return 'http';
}
