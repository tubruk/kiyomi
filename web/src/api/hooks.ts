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
  options?: { enabled?: boolean }
) => {
  return useQuery({
    queryKey: queryKeys.chapters.list(mangaId),
    queryFn: () => api.getMangaChapters(mangaId),
    enabled: options?.enabled ?? Boolean(mangaId),
  });
};

export const useProviderChapterList = (
  providerId: string,
  remoteId: string,
  options?: { enabled?: boolean }
) => {
  return useQuery({
    queryKey: queryKeys.chapters.providerList(providerId, remoteId),
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
      queryClient.invalidateQueries({ queryKey: ['manga', mangaId] });
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

export const useUpdateChapterProgressMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mangaId,
      chapterId,
      progress,
    }: {
      mangaId: string;
      chapterId: string;
      progress: { is_read?: boolean; last_read_page?: number };
    }) => api.updateChapterProgress(mangaId, chapterId, progress),
    onSuccess: (_, { mangaId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.mangas() });
    },
  });
};
