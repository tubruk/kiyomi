import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api } from '../api/client';
import { queryKeys } from './queryKeys';
import {
  sourcesQueryOptions,
  libraryMangasQueryOptions,
  mangaDetailsQueryOptions,
  providerMangaDetailsQueryOptions,
  exploreCatalogQueryOptions,
  pluginsQueryOptions,
  pluginLogsQueryOptions,
  collisionsQueryOptions,
  infoQueryOptions,
  cacheStatsQueryOptions,
  jobsQueryOptions,
} from './queryOptions';

vi.mock('../api/client', () => ({
  api: {
    getSources: vi.fn(),
    getLibraryMangas: vi.fn(),
    getMangaDetails: vi.fn(),
    getProviderMangaDetails: vi.fn(),
    getExplore: vi.fn(),
    searchManga: vi.fn(),
    getPlugins: vi.fn(),
    getPluginLogs: vi.fn(),
    getCollisions: vi.fn(),
    getInfo: vi.fn(),
    getCacheStats: vi.fn(),
    getJobs: vi.fn(),
  },
}));

describe('queryOptions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('sourcesQueryOptions', () => {
    it('returns correct queryKey and executes queryFn', async () => {
      const mockData = [{ id: 'src-1', name: 'Source 1' }];
      vi.mocked(api.getSources).mockResolvedValue(mockData as any);

      const opts = sourcesQueryOptions();
      expect(opts.queryKey).toEqual(queryKeys.sources.all);

      const result = await (opts.queryFn as Function)();
      expect(api.getSources).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockData);
    });
  });

  describe('libraryMangasQueryOptions', () => {
    it('returns correct queryKey and executes queryFn', async () => {
      const mockData = [{ id: 'm-1', title: 'Manga 1' }];
      vi.mocked(api.getLibraryMangas).mockResolvedValue(mockData as any);

      const opts = libraryMangasQueryOptions();
      expect(opts.queryKey).toEqual(queryKeys.library.mangas());

      const result = await (opts.queryFn as Function)();
      expect(api.getLibraryMangas).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockData);
    });
  });

  describe('mangaDetailsQueryOptions', () => {
    it('returns correct options when mangaId is provided', async () => {
      const mockData = { id: 'm-1', title: 'Manga Details' };
      vi.mocked(api.getMangaDetails).mockResolvedValue(mockData as any);

      const opts = mangaDetailsQueryOptions('m-1');
      expect(opts.queryKey).toEqual(queryKeys.manga.details('m-1'));
      expect(opts.enabled).toBe(true);

      const result = await (opts.queryFn as Function)();
      expect(api.getMangaDetails).toHaveBeenCalledWith('m-1');
      expect(result).toEqual(mockData);
    });

    it('sets enabled to false when mangaId is empty', () => {
      const opts = mangaDetailsQueryOptions('');
      expect(opts.queryKey).toEqual(queryKeys.manga.details(''));
      expect(opts.enabled).toBe(false);
    });
  });

  describe('providerMangaDetailsQueryOptions', () => {
    it('returns correct options when providerId and remoteId are provided', async () => {
      const mockData = { id: 'rem-1', title: 'Provider Manga' };
      vi.mocked(api.getProviderMangaDetails).mockResolvedValue(mockData as any);

      const opts = providerMangaDetailsQueryOptions('mangadex', 'rem-1');
      expect(opts.queryKey).toEqual(queryKeys.manga.providerDetails('mangadex', 'rem-1'));
      expect(opts.enabled).toBe(true);

      const result = await (opts.queryFn as Function)();
      expect(api.getProviderMangaDetails).toHaveBeenCalledWith('mangadex', 'rem-1');
      expect(result).toEqual(mockData);
    });

    it('sets enabled to false when providerId or remoteId is missing', () => {
      expect(providerMangaDetailsQueryOptions('', 'rem-1').enabled).toBe(false);
      expect(providerMangaDetailsQueryOptions('mangadex', '').enabled).toBe(false);
      expect(providerMangaDetailsQueryOptions('', '').enabled).toBe(false);
    });
  });

  describe('exploreCatalogQueryOptions', () => {
    it('calls getExplore when query is empty string or undefined', async () => {
      const mockData = { mangas: [{ id: 'p-1' }], hasNext: false, page: 1 };
      vi.mocked(api.getExplore).mockResolvedValue(mockData as any);

      const opts = exploreCatalogQueryOptions('mangadex', 'popular', '', 2);
      expect(opts.queryKey).toEqual(queryKeys.explore.catalog('mangadex', 'popular', '', 2));
      expect(opts.enabled).toBe(true);

      const result = await (opts.queryFn as Function)();
      expect(api.getExplore).toHaveBeenCalledWith('mangadex', 'popular', 2);
      expect(api.searchManga).not.toHaveBeenCalled();
      expect(result).toEqual(mockData);
    });

    it('calls getExplore with default page 1', async () => {
      const mockData = { mangas: [], hasNext: false, page: 1 };
      vi.mocked(api.getExplore).mockResolvedValue(mockData as any);

      const opts = exploreCatalogQueryOptions('mangadex', 'latest');
      expect(opts.queryKey).toEqual(queryKeys.explore.catalog('mangadex', 'latest', '', 1));

      await (opts.queryFn as Function)();
      expect(api.getExplore).toHaveBeenCalledWith('mangadex', 'latest', 1);
    });

    it('calls searchManga when query is provided and trims whitespace', async () => {
      const mockData = { mangas: [{ id: 'search-1' }], hasNext: true, page: 3 };
      vi.mocked(api.searchManga).mockResolvedValue(mockData as any);

      const opts = exploreCatalogQueryOptions('mangafox', 'popular', '  naruto  ', 3);
      expect(opts.queryKey).toEqual(queryKeys.explore.catalog('mangafox', 'popular', 'naruto', 3));
      expect(opts.enabled).toBe(true);

      const result = await (opts.queryFn as Function)();
      expect(api.searchManga).toHaveBeenCalledWith('mangafox', 'naruto', 3);
      expect(api.getExplore).not.toHaveBeenCalled();
      expect(result).toEqual(mockData);
    });

    it('sets enabled to false when providerId is empty', () => {
      const opts = exploreCatalogQueryOptions('', 'popular');
      expect(opts.enabled).toBe(false);
    });
  });

  describe('pluginsQueryOptions', () => {
    it('returns correct queryKey and executes queryFn', async () => {
      const mockData = [{ id: 'pl-1', name: 'Plugin 1' }];
      vi.mocked(api.getPlugins).mockResolvedValue(mockData as any);

      const opts = pluginsQueryOptions();
      expect(opts.queryKey).toEqual(queryKeys.plugins.all);

      const result = await (opts.queryFn as Function)();
      expect(api.getPlugins).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockData);
    });
  });

  describe('pluginLogsQueryOptions', () => {
    it('returns correct options when pluginId is provided', async () => {
      const mockData = [{ timestamp: '2026-08-30', message: 'log' }];
      vi.mocked(api.getPluginLogs).mockResolvedValue(mockData as any);

      const opts = pluginLogsQueryOptions('pl-1');
      expect(opts.queryKey).toEqual(queryKeys.plugins.logs('pl-1'));
      expect(opts.enabled).toBe(true);

      const result = await (opts.queryFn as Function)();
      expect(api.getPluginLogs).toHaveBeenCalledWith('pl-1');
      expect(result).toEqual(mockData);
    });

    it('sets enabled to false when pluginId is empty', () => {
      const opts = pluginLogsQueryOptions('');
      expect(opts.queryKey).toEqual(queryKeys.plugins.logs(''));
      expect(opts.enabled).toBe(false);
    });
  });

  describe('collisionsQueryOptions', () => {
    it('returns correct queryKey and executes queryFn', async () => {
      const mockData = [{ id: 'col-1' }];
      vi.mocked(api.getCollisions).mockResolvedValue(mockData as any);

      const opts = collisionsQueryOptions();
      expect(opts.queryKey).toEqual(queryKeys.collisions.all);

      const result = await (opts.queryFn as Function)();
      expect(api.getCollisions).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockData);
    });
  });

  describe('infoQueryOptions', () => {
    it('returns correct queryKey, staleTime Infinity, and executes queryFn', async () => {
      const mockData = { version: '1.0.0', commit: 'abc' };
      vi.mocked(api.getInfo).mockResolvedValue(mockData as any);

      const opts = infoQueryOptions();
      expect(opts.queryKey).toEqual(queryKeys.info.all);
      expect(opts.staleTime).toBe(Infinity);

      const result = await (opts.queryFn as Function)();
      expect(api.getInfo).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockData);
    });
  });

  describe('cacheStatsQueryOptions', () => {
    it('returns correct queryKey and executes queryFn', async () => {
      const mockData = { size: 1024, count: 10 };
      vi.mocked(api.getCacheStats).mockResolvedValue(mockData as any);

      const opts = cacheStatsQueryOptions();
      expect(opts.queryKey).toEqual(queryKeys.system.cache);

      const result = await (opts.queryFn as Function)();
      expect(api.getCacheStats).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockData);
    });
  });

  describe('jobsQueryOptions', () => {
    it('returns correct queryKey, staleTime, gcTime, and executes queryFn with filter', async () => {
      const filter = { status: 'running' };
      const mockData = [{ id: 'job-1', status: 'running' }];
      vi.mocked(api.getJobs).mockResolvedValue(mockData as any);

      const opts = jobsQueryOptions(filter);
      expect(opts.queryKey).toEqual(queryKeys.jobs.list(filter));
      expect(opts.staleTime).toBe(0);
      expect(opts.gcTime).toBe(120000);

      const result = await (opts.queryFn as Function)();
      expect(api.getJobs).toHaveBeenCalledWith(filter);
      expect(result).toEqual(mockData);
    });

    describe('refetchInterval', () => {
      it('returns 3000 when jobs include running status', () => {
        const opts = jobsQueryOptions();
        const refetchFn = opts.refetchInterval as Function;

        const interval = refetchFn({
          state: {
            data: [
              { id: 'job-1', status: 'completed' },
              { id: 'job-2', status: 'running' },
            ],
          },
        });
        expect(interval).toBe(3000);
      });

      it('returns 3000 when jobs include pending status', () => {
        const opts = jobsQueryOptions();
        const refetchFn = opts.refetchInterval as Function;

        const interval = refetchFn({
          state: {
            data: [
              { id: 'job-1', status: 'pending' },
              { id: 'job-2', status: 'failed' },
            ],
          },
        });
        expect(interval).toBe(3000);
      });

      it('returns false when jobs only have completed or failed statuses', () => {
        const opts = jobsQueryOptions();
        const refetchFn = opts.refetchInterval as Function;

        const interval = refetchFn({
          state: {
            data: [
              { id: 'job-1', status: 'completed' },
              { id: 'job-2', status: 'failed' },
            ],
          },
        });
        expect(interval).toBe(false);
      });

      it('returns false when data is not an array or undefined', () => {
        const opts = jobsQueryOptions();
        const refetchFn = opts.refetchInterval as Function;

        expect(refetchFn({ state: { data: null } })).toBe(false);
        expect(refetchFn({ state: { data: undefined } })).toBe(false);
        expect(refetchFn({ state: { data: {} } })).toBe(false);
        expect(refetchFn({ state: { data: [] } })).toBe(false);
      });
    });
  });
});
