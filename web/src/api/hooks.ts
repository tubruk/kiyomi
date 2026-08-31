import { keepPreviousData, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import { queryKeys } from '../lib/queryKeys';
import {
  sourcesQueryOptions,
  libraryMangasQueryOptions,
  mangaDetailsQueryOptions,
  providerMangaDetailsQueryOptions,
  exploreCatalogQueryOptions,
} from '../lib/queryOptions';
import { Manga, MangaMeta } from '../types/api';

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

export const useChapterPages = (
  chapterId: string,
  options?: { mangaId?: string; providerId?: string; enabled?: boolean }
) => {
  return useQuery({
    queryKey: queryKeys.chapters.pages(chapterId, options?.mangaId, options?.providerId),
    queryFn: () => api.getChapterPages(chapterId, options?.mangaId, options?.providerId),
    enabled: options?.enabled ?? Boolean(chapterId),
  });
};

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
      providerId,
      chapterId,
    }: {
      mangaId: string;
      providerId: string;
      chapterId: string;
    }) => api.deleteChapterFiles(mangaId, providerId, chapterId),
    onSuccess: (_, { mangaId, providerId, chapterId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.pages(chapterId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.providerList(mangaId, providerId) });
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
      providerId,
      chapterId,
    }: {
      mangaId: string;
      providerId: string;
      chapterId: string;
    }) => api.pullChapter(mangaId, providerId, chapterId),
    onSuccess: (_, { mangaId, chapterId, providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.pages(chapterId, mangaId, providerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.providerList(mangaId, providerId) });
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
      providerId,
      progress,
    }: {
      mangaId: string;
      chapterId: string;
      providerId: string;
      progress: { is_read?: boolean; last_read_page?: number };
    }) => api.updateChapterProgress(mangaId, chapterId, providerId, progress),
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
      providerId,
      chapterIds,
      progress,
    }: {
      mangaId: string;
      providerId: string;
      chapterIds: string[];
      progress: { is_read?: boolean; last_read_page?: number };
    }) => api.batchUpdateChapterProgress(mangaId, providerId, chapterIds, progress),
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
      providerId,
      chapterIds,
    }: {
      mangaId: string;
      providerId: string;
      chapterIds: string[];
    }) => api.batchPullChapters(mangaId, providerId, chapterIds),
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
      providerId,
      chapterIds,
    }: {
      mangaId: string;
      providerId: string;
      chapterIds: string[];
    }) => api.batchRefreshChapters(mangaId, providerId, chapterIds),
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
      providerId,
      chapterIds,
    }: {
      mangaId: string;
      providerId: string;
      chapterIds: string[];
    }) => api.batchDeleteChapterFiles(mangaId, providerId, chapterIds),
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
      providerId,
      chapterIds,
    }: {
      mangaId: string;
      providerId: string;
      chapterIds: string[];
    }) => api.batchDeleteChapters(mangaId, providerId, chapterIds),
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

