import { mapLeaderboard, type Leaderboard, type RawLeaderboard } from '@/entities/leaderboard';
import { mapProfile, type RawUserProfile, type UserProfile } from '@/entities/profile';
import { mapSession, type RawSession, type StudySession } from '@/entities/session';
import { apiClient } from './apiClient';

/**
 * User data API — typed wrapper for the read-only user endpoints
 * (GET /api/v1/user/*). Every method maps the snake_case backend DTO into the
 * app's camelCase domain model, so call sites never see raw API shapes.
 */

/** Default page size for the sessions listing — mirrors the backend default. */
export const SESSIONS_PAGE_SIZE = 20;

/** Pagination metadata, app-internal shape. */
export interface PageMeta {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

/** One page of the caller's study sessions. */
export interface SessionsPage {
  items: StudySession[];
  pagination: PageMeta;
}

interface RawPageMeta {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

interface RawSessionsPage {
  items: RawSession[];
  pagination: RawPageMeta;
}

export const userApi = {
  /** GET /api/v1/user/me — the caller's profile, streak and lifetime totals. */
  async getMe(signal?: AbortSignal): Promise<UserProfile> {
    return mapProfile(await apiClient.get<RawUserProfile>('/user/me', { signal }));
  },

  /** GET /api/v1/user/sessions — one page of the caller's sessions, newest first. */
  async listSessions(page: number, signal?: AbortSignal): Promise<SessionsPage> {
    const raw = await apiClient.get<RawSessionsPage>(
      `/user/sessions?page=${page}&limit=${SESSIONS_PAGE_SIZE}`,
      { signal },
    );
    return {
      items: raw.items.map(mapSession),
      pagination: {
        page: raw.pagination.page,
        limit: raw.pagination.limit,
        total: raw.pagination.total,
        totalPages: raw.pagination.total_pages,
      },
    };
  },

  /** GET /api/v1/user/leaderboard — the weekly Top-N plus the caller's rank. */
  async getLeaderboard(signal?: AbortSignal): Promise<Leaderboard> {
    return mapLeaderboard(await apiClient.get<RawLeaderboard>('/user/leaderboard', { signal }));
  },
};
