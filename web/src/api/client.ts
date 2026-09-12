import {
  Source,
  Manga,
  MangaMeta,
  MangaMetadata,
  MangaUserState,
  MangaBindings,
  ProviderRef,
  ExploreResponse,
  ChapterListResponse,
  Chapter,
  ChapterMeta,
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

export interface MergeLibraryMangaPayload {
  keep_manga_id: string;
  source_manga_ids: string[];
  content_provider?: {
    provider_id: string;
    provider_manga_id: string;
  };
  metadata: {
    title: 'keep' | 'merge' | `source:${string}`;
    description: 'keep' | `source:${string}`;
    aliases: 'merge';
    tags: 'merge';
    authors: 'merge';
    artists: 'merge';
    publishers: 'merge';
    release_year: 'keep' | 'merge' | `source:${string}`;
    cover_url: 'keep' | `source:${string}`;
    providers: 'merge';
  };
}

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

  getProviderMangaDetails: async (providerId: string, remoteId: string): Promise<Manga> => {
    // Provider details endpoint returns a flat shape (id, title, cover, status,
    // authors, tags, …) without the per-concern envelope. Wrap it into a Manga
    // so consumers can read manga.metadata.* uniformly. If the response already
    // has a metadata envelope (e.g. cached from a previous backend shape), pass
    // it through unchanged.
    const res: any = await fetchAPI<any>(
      `/providers/${encodeURIComponent(providerId)}/manga/${encodeURIComponent(remoteId)}`
    );
    if (res && res.metadata && typeof res.metadata === 'object') {
      return res as Manga;
    }
    return {
      id: res.id ?? remoteId,
      title: res.title,
      cover: res.cover,
      coverUrl: res.cover,
      coverAssetUrl: res.coverAssetUrl,
      banner: res.banner,
      url: res.url,
      description: res.description,
      author: res.author,
      authors: res.authors,
      artist: res.artist,
      artists: res.artists,
      tags: res.tags,
      genres: res.genres,
      aliases: res.aliases,
      status: res.status,
      userStatus: res.userStatus,
      user_status: res.user_status,
      userRating: res.userRating,
      user_rating: res.user_rating,
      userFavorite: res.userFavorite,
      user_favorite: res.user_favorite,
      userNotes: res.userNotes,
      user_notes: res.user_notes,
      lastReadChapterId: res.lastReadChapterId,
      last_read_chapter_id: res.last_read_chapter_id,
      lastReadAt: res.lastReadAt,
      last_read_at: res.last_read_at,
      readingMode: res.reading_mode ?? res.readingMode,
      reading_mode: res.reading_mode,
      contentRating: res.content_rating ?? res.contentRating,
      content_rating: res.content_rating,
      publisher: res.publisher,
      publishers: res.publishers,
      releaseYear: res.release_year ?? res.releaseYear,
      release_year: res.release_year,
      startDate: res.start_date ?? res.startDate,
      start_date: res.start_date,
      endDate: res.end_date ?? res.endDate,
      end_date: res.end_date,
      country: res.country,
      externalLinks: res.externalLinks ?? res.external_links,
      availability: res.availability,
      sourceId: providerId,
      contentProviderId: providerId,
      contentRemoteId: remoteId,
      libraryMangaId: res.libraryMangaId ?? null,
      metadata: {
        title: res.title ?? '',
        aliases: res.aliases ?? [],
        description: res.description ?? '',
        authors: res.authors ?? (res.author ? [res.author] : []),
        artists: res.artists ?? (res.artist ? [res.artist] : []),
        tags: res.tags ?? res.genres ?? [],
        collections: res.collections ?? [],
        publishers: res.publishers ?? (res.publisher ? [res.publisher] : []),
        release_year: res.release_year,
        releaseYear: res.release_year,
        start_date: res.start_date,
        startDate: res.start_date,
        end_date: res.end_date,
        endDate: res.end_date,
        country: res.country,
        content_rating: res.content_rating,
        contentRating: res.content_rating,
        cover_url: res.cover,
        coverUrl: res.cover,
        cover: res.cover,
        externalLinks: res.externalLinks,
      } as MangaMetadata,
      user_state: {
        status: res.user_status ?? res.userStatus ?? '',
        rating: res.user_rating ?? res.userRating ?? 0,
        favorite: res.user_favorite ?? res.userFavorite ?? false,
        notes: res.user_notes ?? res.userNotes ?? '',
        last_read_chapter_id: res.last_read_chapter_id ?? res.lastReadChapterId,
        lastReadChapterId: res.lastReadChapterId,
        last_read_at: res.last_read_at ?? res.lastReadAt,
        lastReadAt: res.lastReadAt,
      } as MangaUserState,
      bindings: {
        providers: [],
        content: {
          provider_id: providerId,
          provider_manga_id: remoteId,
          reading_mode: res.reading_mode,
        },
      } as MangaBindings,
    };
  },

  getProviderMangaChapters: (providerId: string, remoteId: string): Promise<ChapterListResponse> => {
    return fetchAPI<ChapterListResponse>(`/providers/${providerId}/manga/${encodeURIComponent(remoteId)}/chapters`);
  },

  getProviderChapterPages: (
    providerId: string,
    remoteId: string,
    chapterId: string
  ): Promise<PageListResponse> => {
    return fetchAPI<PageListResponse>(
      `/providers/${encodeURIComponent(providerId)}/manga/${encodeURIComponent(remoteId)}/chapters/${encodeURIComponent(chapterId)}/pages`
    );
  },

  // Central Local Library Manga
  getLibraryMangas: async (): Promise<Manga[]> => {
    const res = await fetchAPI<any>('/library/manga');
    return Array.isArray(res) ? res : (res.data ?? []);
  },

  getMangaDetails: (mangaId: string): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}`);
  },

  // ---- Per-concern endpoints (Stage 3 of manga-metadata-separation) ----
  // The backend splits library manga into metadata / user_state / bindings.
  // Each concern has its own GET + PATCH endpoint and dedicated query keys.
  getLibraryMetadata: (mangaId: string): Promise<MangaMetadata> => {
    return fetchAPI<MangaMetadata>(`/library/manga/${mangaId}/metadata`);
  },

  patchLibraryMangaMetadata: (
    mangaId: string,
    partial: Partial<MangaMetadata>
  ): Promise<MangaMetadata> => {
    return fetchAPI<MangaMetadata>(`/library/manga/${mangaId}/metadata`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(partial),
    });
  },

  getLibraryUserState: (mangaId: string): Promise<MangaUserState> => {
    return fetchAPI<MangaUserState>(`/library/manga/${mangaId}/user_state`);
  },

  patchLibraryMangaUserState: (
    mangaId: string,
    partial: Partial<MangaUserState>
  ): Promise<MangaUserState> => {
    return fetchAPI<MangaUserState>(`/library/manga/${mangaId}/user_state`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(partial),
    });
  },

  getLibraryBindings: (mangaId: string): Promise<MangaBindings> => {
    return fetchAPI<MangaBindings>(`/library/manga/${mangaId}/bindings`);
  },

  refreshMetadata: (mangaId: string): Promise<{ job_id?: string; status?: string }> => {
    return fetchAPI(`/library/manga/${mangaId}/metadata/refresh`, {
      method: 'POST',
    });
  },

  // NOTE: switchContentProvider is NOT redefined here: the legacy 3-arg signature
  // (mangaId, providerId, providerMangaId) is still consumed by existing
  // hooks/tests. Stage 4 will rename it to setActiveContentSource(id, body)
  // once the planned PATCH /library/manga/:mangaId/bindings/content route
  // exists.

  /**
   * Add a provider binding to a library manga.
   * Backend endpoint: POST /library/manga/:mangaId/bindings
   */
  addBinding: (
    mangaId: string,
    ref: ProviderRef,
    options?: { setAsContent?: boolean }
  ): Promise<MangaBindings & { added?: number }> => {
    return fetchAPI(`/library/manga/${mangaId}/bindings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...ref, set_as_content: options?.setAsContent }),
    });
  },

  /**
   * Remove a provider binding from a library manga.
   * Backend endpoint: DELETE /library/manga/:mangaId/bindings/:providerId/:providerMangaId
   */
  removeBinding: (
    mangaId: string,
    providerId: string,
    providerMangaId: string
  ): Promise<void> => {
    return fetchAPI<void>(
      `/library/manga/${mangaId}/bindings/${providerId}/${encodeURIComponent(providerMangaId)}`,
      { method: 'DELETE' }
    );
  },

  /**
   * Switch the active content source for a library manga.
   * Backend endpoint: PATCH /library/manga/:mangaId/bindings/content
   * Replaces the deprecated switchContentProvider 3-arg signature.
   */
  setActiveContentSource: (
    mangaId: string,
    body: { provider_id: string; provider_manga_id: string; reading_mode?: string }
  ): Promise<{ id: string; bindings: MangaBindings }> => {
    return fetchAPI<{ id: string; bindings: MangaBindings }>(
      `/library/manga/${mangaId}/bindings/content`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }
    );
  },

  // ---- End per-concern endpoints ----

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

  // Library Manga Merge
  mergeLibraryManga: (payload: MergeLibraryMangaPayload): Promise<Manga> => {
    return fetchAPI<Manga>('/library/manga/merge', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  },

  updateLibraryManga: (mangaId: string, manga: Partial<Manga>): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(manga),
    });
  },

  /**
   * @deprecated Use api.patchLibraryMangaMetadata and/or api.patchLibraryMangaUserState
   * instead. Will be removed in Stage 4. Kept for now so the existing
   * PATCH /library/manga/:mangaId backend route continues to work for any
   * lingering caller until the component migration catches up.
   */
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
  /**
   * @deprecated Use api.addBinding instead. Will be removed in Stage 4.
   * Note: legacy signature returns Manga (the joined shape) for
   * backward-compat with existing callers. New code should use api.addBinding
   * (returns MangaBindings) or read the response from the joined manga
   * endpoint separately.
   */
  addProvider: (mangaId: string, ref: ProviderRef, setAsContent?: boolean): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}/bindings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...ref, set_as_content: setAsContent }),
    });
  },

  /**
   * @deprecated Use api.removeBinding instead. Will be removed in Stage 4.
   */
  removeProvider: (mangaId: string, providerId: string, providerMangaId: string): Promise<void> => {
    return fetchAPI<void>(
      `/library/manga/${mangaId}/bindings/${providerId}/${encodeURIComponent(providerMangaId)}`,
      { method: 'DELETE' }
    );
  },

  /**
   * @deprecated Legacy 3-arg signature retained for existing callers.
   * Stage 4 will replace with setActiveContentSource(id, body).
   */
  switchContentProvider: (mangaId: string, providerId: string, providerMangaId: string): Promise<Manga> => {
    return fetchAPI<Manga>(`/library/manga/${mangaId}/bindings/content`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider_id: providerId, provider_manga_id: providerMangaId }),
    });
  },

  listProviders: (mangaId: string): Promise<ProviderRef[]> => {
    return fetchAPI<{ providers: ProviderRef[] }>(`/library/manga/${mangaId}/bindings`).then(
      (res) => res?.providers || []
    );
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

  getChapter: (mangaId: string, chapterId: string, _legacyProviderId?: string): Promise<Chapter> => {
    return fetchAPI<Chapter>(
      `/library/manga/${encodeURIComponent(mangaId)}/chapters/${encodeURIComponent(chapterId)}`
    );
  },

  saveChapter: (mangaId: string, chapterId: string, meta: ChapterMeta): Promise<Chapter> => {
    return fetchAPI<Chapter>(
      `/library/manga/${encodeURIComponent(mangaId)}/chapters`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: chapterId, meta }),
      }
    );
  },

  getChapterPages: (
    mangaId: string,
    chapterId: string
  ): Promise<PageListResponse> => {
    return fetchAPI<PageListResponse>(
      `/library/manga/${encodeURIComponent(mangaId)}/chapters/${encodeURIComponent(chapterId)}/pages`
    );
  },

  pullChapter: (
    mangaId: string,
    chapterIdOrProviderId: string,
    maybeChapterId?: string
  ): Promise<{ pages: any[]; job_ids: string[] }> => {
    const chapterId = maybeChapterId || chapterIdOrProviderId;
    return fetchAPI<{ pages: any[]; job_ids: string[] }>(
      `/library/manga/${encodeURIComponent(mangaId)}/chapters/${encodeURIComponent(chapterId)}/pull`,
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

  deleteChapter: (mangaId: string, chapterId: string, _legacyProviderId?: string): Promise<void> => {
    return fetchAPI<void>(
      `/library/manga/${encodeURIComponent(mangaId)}/chapters/${encodeURIComponent(chapterId)}`,
      { method: 'DELETE' }
    );
  },

  deleteChapterFiles: (
    mangaId: string,
    chapterIdOrProviderId: string,
    maybeChapterId?: string
  ): Promise<void> => {
    const chapterId = maybeChapterId || chapterIdOrProviderId;
    return fetchAPI<void>(
      `/library/manga/${encodeURIComponent(mangaId)}/chapters/${encodeURIComponent(chapterId)}/files`,
      { method: 'DELETE' }
    );
  },

  updateChapterProgress: (
    mangaId: string,
    chapterId: string,
    progress: { is_read?: boolean; last_read_page?: number }
  ): Promise<{ id: string; manga_id: string; meta: any }> => {
    const path = `/library/manga/${encodeURIComponent(mangaId)}/chapters/${encodeURIComponent(chapterId)}/progress`;
    return fetchAPI<{ id: string; manga_id: string; meta: any }>(path, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(progress),
    });
  },

  batchUpdateChapterProgress: (
    mangaId: string,
    chapterIdsOrProviderId: string | string[],
    chapterIdsOrProgress: string[] | { is_read?: boolean; last_read_page?: number },
    maybeProgress?: { is_read?: boolean; last_read_page?: number }
  ): Promise<{ updated: number; chapter_ids: string[] }> => {
    let chapterIds: string[];
    let progress: { is_read?: boolean; last_read_page?: number };
    if (Array.isArray(chapterIdsOrProviderId)) {
      chapterIds = chapterIdsOrProviderId;
      progress = (chapterIdsOrProgress as { is_read?: boolean; last_read_page?: number }) || {};
    } else {
      chapterIds = (chapterIdsOrProgress as string[]) || [];
      progress = maybeProgress || {};
    }
    return fetchAPI<{ updated: number; chapter_ids: string[] }>(
      `/library/manga/${encodeURIComponent(mangaId)}/batch/chapters/progress`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
          is_read: progress.is_read,
          last_read_page: progress.last_read_page,
        }),
      }
    );
  },

  batchPullChapters: (
    mangaId: string,
    chapterIdsOrProviderId: string | string[],
    maybeChapterIds?: string[]
  ): Promise<{ chapter_count: number; job_ids: string[]; job_count: number }> => {
    const chapterIds = Array.isArray(chapterIdsOrProviderId)
      ? chapterIdsOrProviderId
      : maybeChapterIds || [];
    return fetchAPI<{ chapter_count: number; job_ids: string[]; job_count: number }>(
      `/library/manga/${encodeURIComponent(mangaId)}/batch/chapters/pull`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
        }),
      }
    );
  },

  refreshChaptersBatch: (
    mangaId: string,
    chapterIdsOrProviderId: string | string[],
    maybeChapterIds?: string[]
  ): Promise<{ refreshed: number; chapter_ids: string[] }> => {
    const chapterIds = Array.isArray(chapterIdsOrProviderId)
      ? chapterIdsOrProviderId
      : maybeChapterIds || [];
    return fetchAPI<{ refreshed: number; chapter_ids: string[] }>(
      `/library/manga/${encodeURIComponent(mangaId)}/batch/chapters/refresh`,
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
    chapterIdsOrProviderId: string | string[],
    maybeChapterIds?: string[]
  ): Promise<{ refreshed: number; chapter_ids: string[] }> => {
    return api.refreshChaptersBatch(mangaId, chapterIdsOrProviderId as any, maybeChapterIds);
  },

  batchDeleteChapters: (
    mangaId: string,
    chapterIdsOrProviderId: string | string[],
    maybeChapterIds?: string[]
  ): Promise<{ removed: number }> => {
    const chapterIds = Array.isArray(chapterIdsOrProviderId)
      ? chapterIdsOrProviderId
      : maybeChapterIds || [];
    return fetchAPI<{ removed: number }>(
      `/library/manga/${encodeURIComponent(mangaId)}/batch/chapters/delete`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chapter_ids: chapterIds,
        }),
      }
    );
  },

  deleteChapterFilesBatch: (
    mangaId: string,
    chapterIdsOrProviderId: string | string[],
    maybeChapterIds?: string[]
  ): Promise<{ deleted: number }> => {
    const chapterIds = Array.isArray(chapterIdsOrProviderId)
      ? chapterIdsOrProviderId
      : maybeChapterIds || [];
    return fetchAPI<{ deleted: number }>(
      `/library/manga/${encodeURIComponent(mangaId)}/batch/chapter-files/delete`,
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
    chapterIdsOrProviderId: string | string[],
    maybeChapterIds?: string[]
  ): Promise<{ deleted: number }> => {
    return api.deleteChapterFilesBatch(mangaId, chapterIdsOrProviderId as any, maybeChapterIds);
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