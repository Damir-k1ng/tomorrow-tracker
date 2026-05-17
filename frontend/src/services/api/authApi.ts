import { mapUser, type RawUser, type User } from '@/entities/user';
import { apiClient } from './apiClient';

/**
 * Auth API — the single call made during bootstrap.
 *
 * `verify` sends the Telegram initData (attached by apiClient) to the backend,
 * which verifies the signature + freshness and returns the resolved user and
 * role. There is no token exchange: Telegram is the identity provider.
 */
export const authApi = {
  async verify(signal?: AbortSignal): Promise<User> {
    const raw = await apiClient.post<RawUser>('/auth/verify', undefined, { signal });
    return mapUser(raw);
  },
};
