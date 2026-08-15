import { queryOptions } from '@tanstack/react-query';
import { api } from '../api/client';
import { queryKeys } from './queryKeys';

export const sourcesQueryOptions = () =>
  queryOptions({
    queryKey: queryKeys.sources.all,
    queryFn: api.getSources,
  });

export const libraryMangasQueryOptions = () =>
  queryOptions({
    queryKey: queryKeys.library.mangas(),
    queryFn: api.getLibraryMangas,
  });

export const mangaDetailsQueryOptions = (mangaId: string) =>
  queryOptions({
    queryKey: queryKeys.manga.details(mangaId),
    queryFn: () => api.getMangaDetails(mangaId),
    enabled: Boolean(mangaId),
  });

export const providerMangaDetailsQueryOptions = (providerId: string, remoteId: string) =>
  queryOptions({
    queryKey: queryKeys.manga.providerDetails(providerId, remoteId),
    queryFn: () => api.getProviderMangaDetails(providerId, remoteId),
    enabled: Boolean(providerId && remoteId),
  });

export const exploreCatalogQueryOptions = (
  providerId: string,
  mode: 'popular' | 'latest',
  query?: string,
  page = 1
) => {
  const activeQuery = (query || '').trim();
  return queryOptions({
    queryKey: queryKeys.explore.catalog(providerId, mode, activeQuery, page),
    queryFn: () =>
      activeQuery
        ? api.searchManga(providerId, activeQuery, page)
        : api.getExplore(providerId, mode, page),
    enabled: Boolean(providerId),
  });
};
