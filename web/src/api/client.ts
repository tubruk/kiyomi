import {
  Source,
  Manga,
  MangaMeta,
  ProviderRef,
  ExploreResponse,
  ChapterListResponse,
  Chapter,
  PageListResponse,
  PluginItem,
  PluginLogEntry,
  ProviderCollision,
  ReloadPluginsResponse,
  AppInfo,
  CacheStats,
  Job,
} from '../types/api';

export type { Job };

const API_BASE = '/api/v1';

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, options);
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ message: res.statusText, code: 'UNKNOWN_ERROR' }));
    const msg = errorData.message || errorData.error || `HTTP ${res.status}`;
    const err = new Error(msg) as Error & { code?: string; details?: any; requestId?: string };
    err.code = errorData.code;
    err.details = errorData.details;
    err.requestId = errorData.request_id;
    throw err;
  }
  if (res.status === 204) {
    return {} as T;
  }
  return res.json();
}

export const api = {
  // System Info
  getInfo: (): Promise<AppInfo> => {
    return fetchAPI<AppInfo>('/info');
  },

  // Content Providers
  getSources: (): Promise<Source[]> => {
    return fetchAPI<Source[]>('/providers');
  },

  getExplore: (
    providerId: string,
    mode: 'popular' | 'latest',
    page: number = 1
  ): Promise<ExploreResponse> => {
    return fetchAPI<ExploreResponse>(`/providers/${providerId}/manga?mode=${mode}&page=${page}`);
  },

  getCatalog: (
    providerId: string,
    mode: 'popular' | 'latest',
    page: number = 1
  ): Promise<ExploreResponse> => {
    return fetchAPI<ExploreResponse>(`/providers/${providerId}/manga?mode=${mode}&page=${page}`);
  },

  searchManga: (
    providerId: string,
    query: string,
    page: number = 1
  ): Promise<ExploreResponse> => {
    return fetchAPI<ExploreResponse>(
      `/providers/${providerId}/manga?q=${encodeURIComponent(query)}&page=${page}`
    );
  },

  getProviderMangaDetails: (providerId: string, remoteId: string): Promise<Manga> => {
    return fetchAPI<Manga>(`/providers/${providerId}/manga/${remoteId}`);
  },

  getProviderMangaChapters: (providerId: string, remoteId: string): Promise<ChapterListResponse> => {
    return fetchAPI<ChapterListResponse>(`/providers/${providerId}/manga/${encodeURIComponent(remoteId)}/chapters`);
  },

  // Central Local Library Manga
  getLibraryMangas: async (): Promise<Manga[]> => {
    const res = await fetchAPI<any>('/library/manga');
    return Array.isArray(res) ? res : (res.data ?? []);
  },

  getMangaDetails: (mangaId: string): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}`);
  },

  postLibraryManga: (manga: Partial<Manga>): Promise<Manga> => {
    return fetchAPI<Manga>('/library/manga', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(manga),
    });
  },

  createLibraryManga: (manga: Partial<Manga>): Promise<Manga> => {
    return fetchAPI<Manga>('/library/manga', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(manga),
    });
  },

  importProviderManga: (providerId: string, remoteId: string, userStatus: string = 'plan_to_read'): Promise<Manga> => {
    return fetchAPI<Manga>('/library/manga/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider_id: providerId, remote_id: remoteId, user_status: userStatus }),
    });
  },

  updateLibraryManga: (mangaId: string, manga: Partial<Manga>): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(manga),
    });
  },

  patchLibraryManga: (mangaId: string, fields: Partial<MangaMeta> | Partial<Manga> | Record<string, any>): Promise<Manga> => {
    const payload: Record<string, any> = { ...fields };
    if ('userStatus' in payload && payload.user_status === undefined) {
      payload.user_status = payload.userStatus;
    }
    if ('userFavorite' in payload && payload.user_favorite === undefined) {
      payload.user_favorite = payload.userFavorite;
    }
    if ('isFavorite' in payload && payload.user_favorite === undefined) {
      payload.user_favorite = payload.isFavorite;
    }
    if ('userRating' in payload && payload.user_rating === undefined) {
      payload.user_rating = payload.userRating;
    }
    if ('userNotes' in payload && payload.user_notes === undefined) {
      payload.user_notes = payload.userNotes;
    }
    return fetchAPI<Manga>(`/library/manga/${mangaId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  },

  deleteLibraryManga: (mangaId: string): Promise<void> => {
    return fetchAPI<void>(`/library/manga/${mangaId}`, { method: 'DELETE' });
  },

  // Provider Bindings
  addProvider: (mangaId: string, ref: ProviderRef, setAsContent?: boolean): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}/providers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...ref, set_as_content: setAsContent }),
    });
  },

  removeProvider: (mangaId: string, providerId: string, providerMangaId: string): Promise<void> => {
    return fetchAPI<void>(`/library/manga/${mangaId}/providers/${providerId}/${encodeURIComponent(providerMangaId)}`, { method: 'DELETE' });
  },

  switchContentProvider: (mangaId: string, providerId: string, providerMangaId: string): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}/content`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider_id: providerId, provider_manga_id: providerMangaId }),
    });
  },

  listProviders: (mangaId: string): Promise<ProviderRef[]> => {
    return fetchAPI<ProviderRef[]>(`/library/manga/${mangaId}/providers`);
  },

  // Chapters & Pages
  listChapters: (mangaId: string, providerId?: string): Promise<ChapterListResponse> => {
    let path = `/library/manga/${encodeURIComponent(mangaId)}/chapters`;
    if (providerId) {
      const query = new URLSearchParams();
      query.set('provider_id', providerId);
      path += `?${query.toString()}`;
    }
    return fetchAPI<ChapterListResponse>(path);
  },

  getMangaChapters: (mangaId: string): Promise<ChapterListResponse> => {
    return fetchAPI<ChapterListResponse>(
      `/library/manga/${encodeURIComponent(mangaId)}/chapters`
    );
  },

  getChapter: (mangaId: string, chapterId: string, providerId: string): Promise<Chapter> => {
    return fetchAPI<Chapter>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/${encodeURIComponent(chapterId)}`
    );
  },

  getChapterPages: (chapterId: string, mangaId?: string, providerId?: string): Promise<PageListResponse> => {
    let path = `/chapters/${chapterId}/pages`;
    const query = new URLSearchParams();
    if (mangaId) query.set('mangaId', mangaId);
    if (providerId) query.set('providerId', providerId);
    if (query.toString()) path += `?${query.toString()}`;
    return fetchAPI<PageListResponse>(path);
  },

  pullChapter: (
    mangaId: string,
    providerId: string,
    chapterId: string
  ): Promise<{ pages: any[]; job_ids: string[] }> => {
    return fetchAPI<{ pages: any[]; job_ids: string[] }>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/${encodeURIComponent(chapterId)}/pull`,
      { method: 'POST' }
    );
  },

  deleteChapterPagesCache: (chapterId: string): Promise<void> => {
    return fetchAPI<void>(`/chapters/${encodeURIComponent(chapterId)}/pages`, { method: 'DELETE' });
  },

  // Library Refresh & Pull Operations
  refreshLibraryManga: (mangaId: string): Promise<{ added: number; orphaned: number; provider_id: string }> => {
    return fetchAPI(`/library/manga/${encodeURIComponent(mangaId)}/refresh`, {
      method: 'POST',
    });
  },

  pullManga: (mangaId: string, providerId?: string, providerMangaId?: string): Promise<void> => {
    let path = `/library/manga/${encodeURIComponent(mangaId)}/pull`;
    const query = new URLSearchParams();
    if (providerId) query.set('provider_id', providerId);
    if (providerMangaId) query.set('provider_manga_id', providerMangaId);
    const queryString = query.toString();
    if (queryString) path += `?${queryString}`;
    return fetchAPI<void>(path, { method: 'POST' });
  },

  deleteChapter: (mangaId: string, chapterId: string, providerId: string): Promise<void> => {
    return fetchAPI<void>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/${encodeURIComponent(chapterId)}`,
      { method: 'DELETE' }
    );
  },

  deleteChapterFiles: (mangaId: string, providerId: string, chapterId: string): Promise<void> => {
    return fetchAPI<void>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/${encodeURIComponent(chapterId)}/files`,
      { method: 'DELETE' }
    );
  },

  updateChapterProgress: (
    mangaId: string,
    chapterId: string,
    providerId: string,
    progress: { is_read?: boolean; last_read_page?: number }
  ): Promise<{ id: string; manga_id: string; meta: any }> => {
    const path = `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/${encodeURIComponent(chapterId)}/progress`;
    return fetchAPI<{ id: string; manga_id: string; meta: any }>(path, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(progress),
    });
  },

  batchUpdateChapterProgress: (
    mangaId: string,
    providerId: string,
    chapterIds: string[],
    progress: { is_read?: boolean; last_read_page?: number }
  ): Promise<{ updated: number; chapter_ids: string[] }> => {
    return fetchAPI<{ updated: number; chapter_ids: string[] }>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/progress`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
          ...progress,
        }),
      }
    );
  },

  batchPullChapters: (
    mangaId: string,
    providerId: string,
    chapterIds: string[]
  ): Promise<{ chapter_count: number; job_ids: string[]; job_count: number }> => {
    return fetchAPI<{ chapter_count: number; job_ids: string[]; job_count: number }>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/pull`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
        }),
      }
    );
  },

  batchRefreshChapters: (
    mangaId: string,
    providerId: string,
    chapterIds: string[]
  ): Promise<{ refreshed: number; chapter_ids: string[] }> => {
    return fetchAPI<{ refreshed: number; chapter_ids: string[] }>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/refresh`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
        }),
      }
    );
  },

  batchDeleteChapterFiles: (
    mangaId: string,
    providerId: string,
    chapterIds: string[]
  ): Promise<{ deleted: number }> => {
    return fetchAPI<{ deleted: number }>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/files/delete`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
        }),
      }
    );
  },

  batchDeleteChapters: (
    mangaId: string,
    providerId: string,
    chapterIds: string[]
  ): Promise<{ removed: number }> => {
    return fetchAPI<{ removed: number }>(
      `/library/manga/${encodeURIComponent(mangaId)}/providers/${encodeURIComponent(providerId)}/chapters/delete`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
        }),
      }
    );
  },
  // Plugins Management & Diagnostics
  getPlugins: (): Promise<PluginItem[]> => {
    return fetchAPI<PluginItem[]>('/plugins');
  },

  reloadPlugins: (): Promise<ReloadPluginsResponse> => {
    return fetchAPI<ReloadPluginsResponse>('/plugins/reload', {
      method: 'POST',
    });
  },

  getPluginLogs: (pluginId: string): Promise<PluginLogEntry[]> => {
    return fetchAPI<PluginLogEntry[]>(`/plugins/${encodeURIComponent(pluginId)}/logs`);
  },

  updatePluginConfig: (
    pluginId: string,
    config: { globalConfig?: Record<string, string>; providerConfigs?: Record<string, Record<string, string>> }
  ): Promise<{ status: string; message: string; pluginId: string }> => {
    return fetchAPI<{ status: string; message: string; pluginId: string }>(
      `/plugins/${encodeURIComponent(pluginId)}/config`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      }
    );
  },

  getCollisions: (): Promise<ProviderCollision[]> => {
    return fetchAPI<ProviderCollision[]>('/plugins/collisions');
  },

  setPluginPreference: (
    providerId: string,
    preference: string
  ): Promise<{ status: string; message: string; providerId: string; preference: string }> => {
    return fetchAPI<{ status: string; message: string; providerId: string; preference: string }>(
      '/plugins/preference',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ providerId, preference }),
      }
    );
  },

  // Cache Management
  getCacheStats: (): Promise<CacheStats> => {
    return fetchAPI<CacheStats>('/system/cache');
  },

  clearCache: (): Promise<{ status: string }> => {
    return fetchAPI<{ status: string }>('/system/cache/clear', {
      method: 'POST',
    });
  },

  // Job Management
  getJobs: (filter?: { status?: string; type?: string; parent_id?: string; all?: boolean; [key: string]: any }): Promise<Job[]> => {
    let path = '/jobs';
    if (filter && Object.keys(filter).length > 0) {
      const params = new URLSearchParams();
      Object.entries(filter).forEach(([k, v]) => {
        if (v !== undefined && v !== null && v !== '') {
          params.append(k, String(v));
        }
      });
      const qs = params.toString();
      if (qs) path += `?${qs}`;
    }
    return fetchAPI<Job[]>(path);
  },

  cancelJob: (id: string): Promise<Job> => {
    return fetchAPI<Job>(`/jobs/${encodeURIComponent(id)}/cancel`, {
      method: 'POST',
    });
  },

  deleteJob: (id: string): Promise<{ message: string; id: string }> => {
    return fetchAPI<{ message: string; id: string }>(`/jobs/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },

  cleanupJobs: (status?: string): Promise<{ deleted: number; status: string }> => {
    let path = '/jobs';
    if (status) {
      path += `?status=${encodeURIComponent(status)}`;
    }
    return fetchAPI<{ deleted: number; status: string }>(path, {
      method: 'DELETE',
    });
  },
};