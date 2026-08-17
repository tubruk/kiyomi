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

export const pluginsQueryOptions = () =>
  queryOptions({
    queryKey: queryKeys.plugins.all,
    queryFn: api.getPlugins,
  });

export const pluginLogsQueryOptions = (pluginId: string) =>
  queryOptions({
    queryKey: queryKeys.plugins.logs(pluginId),
    queryFn: () => api.getPluginLogs(pluginId),
    enabled: Boolean(pluginId),
  });

export const collisionsQueryOptions = () =>
  queryOptions({
    queryKey: queryKeys.collisions.all,
    queryFn: api.getCollisions,
  });

export const infoQueryOptions = () =>
  queryOptions({
    queryKey: queryKeys.info,
    queryFn: () => api.getInfo(),
    staleTime: Infinity, // build info never changes at runtime
  });

export const cacheStatsQueryOptions = () =>
  queryOptions({
    queryKey: queryKeys.system.cache,
    queryFn: api.getCacheStats,
  });
