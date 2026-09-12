import { keepPreviousData, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { api, MergeLibraryMangaPayload } from './client';
import { queryKeys } from '../lib/queryKeys';
import {
  sourcesQueryOptions,
  libraryMangasQueryOptions,
  mangaDetailsQueryOptions,
  providerMangaDetailsQueryOptions,
  exploreCatalogQueryOptions,
} from '../lib/queryOptions';
import { useToast } from '../context/ToastContext';
import { Manga, MangaMeta, MangaMetadata, MangaUserState } from '../types/api';

// Per-concern query keys (Stage 3 of manga-metadata-separation).
// These mirror the GET endpoints exposed by the backend under
// /library/manga/:id/{metadata,user_state,bindings}. Invalidation should
// use these granular keys so per-concern mutations don't trigger unrelated
// refetches. Backed by the canonical keys in ../lib/queryKeys to avoid
// duplication.
export const mangaKeys = {
  all: queryKeys.manga.all,
  detail: queryKeys.manga.detail,
  metadata: queryKeys.manga.metadata,
  userState: queryKeys.manga.userState,
  bindings: queryKeys.manga.bindings,
};

// Queries

export const useSources = () => {
  return useQuery(sourcesQueryOptions());
};

export const useLibraryManga = () => {
  return useQuery(libraryMangasQueryOptions());
};

export const useMangaDetails = (id: string, options?: { enabled?: boolean }) => {
  const opts = mangaDetailsQueryOptions(id);
  return useQuery({
    ...opts,
    enabled: options?.enabled ?? opts.enabled,
  });
};

// ---- Per-concern hooks (Stage 3 of manga-metadata-separation) ----
// Each hook targets a single concern (metadata / user_state / bindings) so
// consumers can subscribe to exactly the slice they render. Mutations
// invalidate only the relevant per-concern keys (plus manga.detail for
// user_state, since last_read_* fields are echoed on the joined manga).

export const useMetadata = (id: string, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: mangaKeys.metadata(id),
    queryFn: () => api.getLibraryMetadata(id),
    enabled: options?.enabled ?? Boolean(id),
  });
};

export const useUserState = (id: string, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: mangaKeys.userState(id),
    queryFn: () => api.getLibraryUserState(id),
    enabled: options?.enabled ?? Boolean(id),
  });
};

export const useBindings = (id: string, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: mangaKeys.bindings(id),
    queryFn: () => api.getLibraryBindings(id),
    enabled: options?.enabled ?? Boolean(id),
  });
};

export const useProviderMangaDetails = (
  providerId: string,
  remoteId: string,
  options?: { enabled?: boolean }
) => {
  const opts = providerMangaDetailsQueryOptions(providerId, remoteId);
  return useQuery({
    ...opts,
    enabled: options?.enabled ?? opts.enabled,
  });
};

export const useExploreCatalog = (
  providerId: string,
  mode: string,
  query: string,
  page: number,
  options?: { enabled?: boolean }
) => {
  const opts = exploreCatalogQueryOptions(
    providerId,
    (mode as 'popular' | 'latest') || 'popular',
    query,
    page
  );
  return useQuery({
    ...opts,
    enabled: options?.enabled ?? opts.enabled,
    placeholderData: keepPreviousData,
  });
};

export const useChapterList = (
  mangaId: string,
  options?: { enabled?: boolean; providerId?: string; hasActivePullJobs?: boolean }
) => {
  const providerId = options?.providerId;
  return useQuery({
    queryKey: providerId
      ? queryKeys.chapters.providerList(mangaId, providerId)
      : queryKeys.chapters.list(mangaId),
    queryFn: () =>
      providerId ? api.listChapters(mangaId, providerId) : api.getMangaChapters(mangaId),
    enabled: options?.enabled ?? Boolean(mangaId),
    refetchOnWindowFocus: true,
    refetchInterval: (query) => {
      if (options?.hasActivePullJobs) {
        return 3000;
      }
      const chapters = query.state.data?.chapters;
      if (
        Array.isArray(chapters) &&
        chapters.some((c) => {
          const isDownloaded = Boolean(c.is_downloaded ?? (c as any).isDownloaded ?? c.meta?.is_downloaded);
          const downloadedPages = c.downloaded_pages ?? (c as any).downloadedPages ?? c.meta?.downloaded_pages ?? 0;
          return downloadedPages > 0 && !isDownloaded;
        })
      ) {
        return 3000;
      }
      return false;
    },
  });
};

export const useProviderChapterList = (
  providerId: string,
  remoteId: string,
  options?: { enabled?: boolean }
) => {
  return useQuery({
    queryKey: queryKeys.chapters.remoteList(providerId, remoteId),
    queryFn: () => api.getProviderMangaChapters(providerId, remoteId),
    enabled: options?.enabled ?? Boolean(providerId && remoteId),
  });
};

export const useProviderChapterPages = (
  providerId: string,
  remoteId: string,
  chapterId: string,
  options?: { enabled?: boolean }
) => {
  return useQuery({
    queryKey: queryKeys.chapters.remotePages(providerId, remoteId, chapterId),
    queryFn: () => api.getProviderChapterPages(providerId, remoteId, chapterId),
    enabled: options?.enabled ?? Boolean(providerId && remoteId && chapterId),
  });
};

export function useChapterPages(
  mangaIdOrChapterId: string,
  chapterIdOrOptions?: string | { mangaId?: string; providerId?: string; enabled?: boolean },
  options?: { enabled?: boolean }
) {
  let mangaId = '';
  let chapterId = '';
  let enabled: boolean | undefined;

  if (typeof chapterIdOrOptions === 'string') {
    // Standard signature: useChapterPages(mangaId, chapterId, options)
    mangaId = mangaIdOrChapterId;
    chapterId = chapterIdOrOptions;
    enabled = options?.enabled;
  } else {
    // Legacy signature: useChapterPages(chapterId, options)
    chapterId = mangaIdOrChapterId;
    mangaId = chapterIdOrOptions?.mangaId || '';
    enabled = chapterIdOrOptions?.enabled;
  }

  return useQuery({
    queryKey: queryKeys.chapters.pages(chapterId, mangaId),
    queryFn: () => api.getChapterPages(mangaId, chapterId),
    enabled: enabled ?? Boolean(chapterId && mangaId),
  });
}

// Mutations (Pessimistic updates with standard onSuccess query invalidation)

export const useSaveLibraryMangaMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (manga: Partial<Manga>) => api.postLibraryManga(manga),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

/**
 * @deprecated Use useUpdateMetadataMutation and/or useUpdateUserStateMutation
 * instead. Will be removed in Stage 4 once all components migrate to
 * per-concern mutations.
 */
export const useUpdateLibraryMangaMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ mangaId, fields }: { mangaId: string; fields: Partial<MangaMeta> | Partial<Manga> | Record<string, any> }) =>
      api.patchLibraryManga(mangaId, fields),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

// ---- Per-concern mutation hooks ----
// The mutationFn takes a per-concern partial; onSuccess only invalidates
// the matching concern's cache key. user_state mutations also invalidate
// mangaKeys.detail(id) so consumers reading last_read_* off the joined
// manga view stay fresh.

export const useUpdateMetadataMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ mangaId, partial }: { mangaId: string; partial: Partial<MangaMetadata> }) =>
      api.patchLibraryMangaMetadata(mangaId, partial),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: mangaKeys.metadata(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

export const useUpdateUserStateMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ mangaId, partial }: { mangaId: string; partial: Partial<MangaUserState> }) =>
      api.patchLibraryMangaUserState(mangaId, partial),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: mangaKeys.userState(mangaId) });
      // last_read_chapter_id / last_read_at affect the joined manga view.
      queryClient.invalidateQueries({ queryKey: mangaKeys.detail(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

export const useAddBindingMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      ref,
      setAsContent,
    }: {
      mangaId: string;
      ref: { provider_id: string; provider_manga_id: string; manga_title?: string };
      setAsContent?: boolean;
    }) => api.addBinding(mangaId, ref, { setAsContent }),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: mangaKeys.bindings(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

export const useRemoveBindingMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      providerId,
      providerMangaId,
    }: {
      mangaId: string;
      providerId: string;
      providerMangaId: string;
    }) => api.removeBinding(mangaId, providerId, providerMangaId),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: mangaKeys.bindings(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

/**
 * @deprecated Until the planned PATCH /library/manga/:mangaId/bindings/content
 * route ships, switchContentProvider keeps its legacy 3-arg signature.
 * This mutation hook continues to call api.switchContentProvider directly.
 */
export const useSwitchContentProviderMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      providerId,
      providerMangaId,
    }: {
      mangaId: string;
      providerId: string;
      providerMangaId: string;
    }) => api.switchContentProvider(mangaId, providerId, providerMangaId),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: mangaKeys.bindings(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

export const useSetActiveContentSourceMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      body,
    }: {
      mangaId: string;
      body: { provider_id: string; provider_manga_id: string; reading_mode?: string };
    }) => api.setActiveContentSource(mangaId, body),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: mangaKeys.bindings(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

export const useRefreshMetadataMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mangaId: string) => api.refreshMetadata(mangaId),
    onSuccess: (_, mangaId) => {
      queryClient.invalidateQueries({ queryKey: mangaKeys.metadata(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
  });
};

export const useDeleteLibraryMangaMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mangaId: string) => api.deleteLibraryManga(mangaId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

export const useDeleteChapterPagesCacheMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (chapterId: string) => api.deleteChapterPagesCache(chapterId),
    onSuccess: (_, chapterId) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.pages(chapterId) });
    },
  });
};

export const useDeleteChapterFilesMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterId,
    }: {
      mangaId: string;
      chapterId: string;
      providerId?: string;
    }) => api.deleteChapterFiles(mangaId, chapterId),
    onSuccess: (_, { mangaId, providerId, chapterId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.pages(chapterId, mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({ queryKey: queryKeys.chapters.providerList(mangaId, providerId) });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.system.all });
    },
  });
};

export const usePullChapterMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterId,
    }: {
      mangaId: string;
      chapterId: string;
      providerId?: string;
    }) => api.pullChapter(mangaId, chapterId),
    onSuccess: (_, { mangaId, chapterId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.pages(chapterId, mangaId, providerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({ queryKey: queryKeys.chapters.providerList(mangaId, providerId) });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
  });
};

export const usePullMangaMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      providerId,
      providerMangaId,
    }: {
      mangaId: string;
      providerId?: string;
      providerMangaId?: string;
    }) => api.pullManga(mangaId, providerId, providerMangaId),
    onSuccess: (_, { mangaId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.pull(mangaId, providerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(mangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
  });
};

export const useRefreshLibraryMangaMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mangaId: string) => api.refreshLibraryManga(mangaId),
    onSuccess: (_, mangaId) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.refresh(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
    },
  });
};

export const useUpdateChapterProgressMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterId,
      isRead,
      lastReadPage,
      is_read,
      last_read_page,
      progress,
    }: {
      mangaId: string;
      chapterId: string;
      isRead?: boolean;
      lastReadPage?: number;
      is_read?: boolean;
      last_read_page?: number;
      providerId?: string;
      progress?: { is_read?: boolean; last_read_page?: number; isRead?: boolean; lastReadPage?: number };
    }) => {
      const readVal = isRead ?? is_read ?? progress?.is_read ?? progress?.isRead;
      const pageVal = lastReadPage ?? last_read_page ?? progress?.last_read_page ?? progress?.lastReadPage;
      const payload: { is_read?: boolean; last_read_page?: number } = {};
      if (readVal !== undefined) payload.is_read = readVal;
      if (pageVal !== undefined) payload.last_read_page = pageVal;
      return api.updateChapterProgress(mangaId, chapterId, payload);
    },
    onSuccess: (_, { mangaId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(mangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};

export const useBatchUpdateChapterProgressMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterIds,
      progress,
      isRead,
      lastReadPage,
      is_read,
      last_read_page,
    }: {
      mangaId: string;
      chapterIds: string[];
      providerId?: string;
      progress?: { is_read?: boolean; last_read_page?: number; isRead?: boolean; lastReadPage?: number };
      isRead?: boolean;
      lastReadPage?: number;
      is_read?: boolean;
      last_read_page?: number;
    }) => {
      const readVal = isRead ?? is_read ?? progress?.is_read ?? progress?.isRead;
      const pageVal = lastReadPage ?? last_read_page ?? progress?.last_read_page ?? progress?.lastReadPage;
      const payload: { is_read?: boolean; last_read_page?: number } = {};
      if (readVal !== undefined) payload.is_read = readVal;
      if (pageVal !== undefined) payload.last_read_page = pageVal;
      return api.batchUpdateChapterProgress(mangaId, chapterIds, payload);
    },
    onSuccess: (_, { mangaId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(mangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
  });
};

export const useBatchPullChaptersMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterIds,
    }: {
      mangaId: string;
      chapterIds: string[];
      providerId?: string;
    }) => api.batchPullChapters(mangaId, chapterIds),
    onSuccess: (_, { mangaId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(mangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
  });
};

export const useBatchRefreshChaptersMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterIds,
    }: {
      mangaId: string;
      chapterIds: string[];
      providerId?: string;
    }) => api.batchRefreshChapters(mangaId, chapterIds),
    onSuccess: (_, { mangaId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(mangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
    },
  });
};

export const useBatchDeleteChapterFilesMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterIds,
    }: {
      mangaId: string;
      chapterIds: string[];
      providerId?: string;
    }) => api.batchDeleteChapterFiles(mangaId, chapterIds),
    onSuccess: (_, { mangaId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(mangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.system.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
  });
};

export const useBatchDeleteChaptersMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterIds,
    }: {
      mangaId: string;
      chapterIds: string[];
      providerId?: string;
    }) => api.batchDeleteChapters(mangaId, chapterIds),
    onSuccess: (_, { mangaId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(mangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
  });
};

export const useMergeLibraryMangaMutation = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { showToast } = useToast();
  return useMutation({
    mutationFn: (payload: MergeLibraryMangaPayload) => api.mergeLibraryManga(payload),
    onSuccess: (_, payload) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.all });
      navigate({ to: '/manga/$mangaId', params: { mangaId: payload.keep_manga_id } });
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Merge failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });
};

