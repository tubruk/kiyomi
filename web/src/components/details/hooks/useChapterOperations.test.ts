import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useChapterOperations } from './useChapterOperations';
import { api } from '../../../api/client';
import * as toastContext from '../../../context/ToastContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Chapter, Job, Manga, ProviderRef } from '../../../types/api';
import { queryKeys } from '../../../lib/queryKeys';

vi.mock('../../../context/ToastContext', () => ({
  useToast: vi.fn(),
}));

describe('useChapterOperations', () => {
  const showToast = vi.fn();
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.mocked(toastContext.useToast).mockReturnValue({ showToast } as any);

    // Default API mocks
    vi.spyOn(api, 'importProviderManga').mockResolvedValue({ id: 'm-1', title: 'Imported Manga' } as any);
    vi.spyOn(api, 'refreshLibraryManga').mockResolvedValue({ added: 2, orphaned: 0, provider_id: 'mangafox' });
    vi.spyOn(api, 'deleteChapter').mockResolvedValue(undefined);
    vi.spyOn(api, 'deleteChapterFiles').mockResolvedValue(undefined);
    vi.spyOn(api, 'pullChapter').mockResolvedValue({ pages: [], job_ids: ['job-1'] });
    vi.spyOn(api, 'removeProvider').mockResolvedValue(undefined);
    vi.spyOn(api, 'switchContentProvider').mockResolvedValue({ id: 'm-1' } as any);
    vi.spyOn(api, 'batchUpdateChapterProgress').mockResolvedValue({ updated: 2, chapter_ids: ['c1', 'c2'] });
    vi.spyOn(api, 'batchPullChapters').mockResolvedValue({ chapter_count: 2, job_ids: ['j1', 'j2'], job_count: 2 });
    vi.spyOn(api, 'batchRefreshChapters').mockResolvedValue({ refreshed: 2, chapter_ids: ['c1', 'c2'] });
    vi.spyOn(api, 'batchDeleteChapterFiles').mockResolvedValue({ deleted: 2 });
    vi.spyOn(api, 'batchDeleteChapters').mockResolvedValue({ removed: 2 });
    vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({ id: 'm-1' } as any);
  });

  const createWrapper = () => {
    return ({ children }: { children: React.ReactNode }) =>
      React.createElement(QueryClientProvider, { client: queryClient }, children);
  };

  describe('Optimistic pull IDs cleanup', () => {
    it('removes optimistic pull IDs when chapter is downloaded', () => {
      const initialChapters: Chapter[] = [
        { id: 'ch-1', name: 'Chapter 1', number: 1, is_downloaded: false },
        { id: 'ch-2', name: 'Chapter 2', number: 2, is_downloaded: false },
      ];

      const { result, rerender } = renderHook(
        (props: { chapters: Chapter[]; mangaJobs?: Job[] }) =>
          useChapterOperations({
            targetMangaId: 'm-1',
            activeContentProviderId: 'mangafox',
            chapters: props.chapters,
            mangaJobs: props.mangaJobs,
          }),
        {
          wrapper: createWrapper(),
          initialProps: { chapters: initialChapters, mangaJobs: [] },
        }
      );

      act(() => {
        result.current.handlePullChapter('ch-1', 'mangafox');
        result.current.handlePullChapter('ch-2', 'mangafox');
      });

      expect(result.current.optimisticPullIds.has('ch-1')).toBe(true);
      expect(result.current.optimisticPullIds.has('ch-2')).toBe(true);

      // ch-1 becomes downloaded
      const updatedChapters: Chapter[] = [
        { id: 'ch-1', name: 'Chapter 1', number: 1, is_downloaded: true },
        { id: 'ch-2', name: 'Chapter 2', number: 2, is_downloaded: false },
      ];

      rerender({ chapters: updatedChapters, mangaJobs: [] });

      expect(result.current.optimisticPullIds.has('ch-1')).toBe(false);
      expect(result.current.optimisticPullIds.has('ch-2')).toBe(true);
    });

    it('removes optimistic pull IDs when chapter download is indicated via meta.is_downloaded', () => {
      const initialChapters: any[] = [
        { id: 'ch-1', name: 'Chapter 1', number: 1, meta: { is_downloaded: false } },
      ];

      const { result, rerender } = renderHook(
        (props: { chapters: Chapter[] }) =>
          useChapterOperations({
            targetMangaId: 'm-1',
            activeContentProviderId: 'mangafox',
            chapters: props.chapters,
          }),
        {
          wrapper: createWrapper(),
          initialProps: { chapters: initialChapters },
        }
      );

      act(() => {
        result.current.handlePullChapter('ch-1', 'mangafox');
      });
      expect(result.current.optimisticPullIds.has('ch-1')).toBe(true);

      const updatedChapters: any[] = [
        { id: 'ch-1', name: 'Chapter 1', number: 1, meta: { is_downloaded: true } },
      ];

      rerender({ chapters: updatedChapters });
      expect(result.current.optimisticPullIds.has('ch-1')).toBe(false);
    });

    it('removes optimistic pull IDs when pull job is completed or failed', () => {
      const initialChapters: Chapter[] = [
        { id: 'ch-1', name: 'Chapter 1', number: 1, is_downloaded: false },
        { id: 'ch-2', name: 'Chapter 2', number: 2, is_downloaded: false },
      ];

      const { result, rerender } = renderHook(
        (props: { chapters: Chapter[]; mangaJobs: Job[] }) =>
          useChapterOperations({
            targetMangaId: 'm-1',
            activeContentProviderId: 'mangafox',
            chapters: props.chapters,
            mangaJobs: props.mangaJobs,
          }),
        {
          wrapper: createWrapper(),
          initialProps: { chapters: initialChapters, mangaJobs: [] as Job[] },
        }
      );

      act(() => {
        result.current.handlePullChapter('ch-1', 'mangafox');
        result.current.handlePullChapter('ch-2', 'mangafox');
      });

      // ch-1 job completed, ch-2 job failed
      const jobs: Job[] = [
        {
          id: 'j-1',
          type: 'pull_chapter',
          status: 'completed',
          metadata: { chapter_id: 'ch-1' },
          retries: 0,
          max_retries: 3,
          created_at: '',
          updated_at: '',
        },
        {
          id: 'j-2',
          type: 'pull_chapter',
          status: 'failed',
          metadata: { chapter_id: 'ch-2' },
          retries: 0,
          max_retries: 3,
          created_at: '',
          updated_at: '',
        },
      ];

      rerender({ chapters: initialChapters, mangaJobs: jobs });

      expect(result.current.optimisticPullIds.has('ch-1')).toBe(false);
      expect(result.current.optimisticPullIds.has('ch-2')).toBe(false);
    });

    it('retains optimistic pull IDs when job is running or pending', () => {
      const initialChapters: Chapter[] = [
        { id: 'ch-1', name: 'Chapter 1', number: 1, is_downloaded: false },
      ];

      const { result, rerender } = renderHook(
        (props: { chapters: Chapter[]; mangaJobs: Job[] }) =>
          useChapterOperations({
            targetMangaId: 'm-1',
            activeContentProviderId: 'mangafox',
            chapters: props.chapters,
            mangaJobs: props.mangaJobs,
          }),
        {
          wrapper: createWrapper(),
          initialProps: { chapters: initialChapters, mangaJobs: [] as Job[] },
        }
      );

      act(() => {
        result.current.handlePullChapter('ch-1', 'mangafox');
      });

      const runningJobs: Job[] = [
        {
          id: 'j-1',
          type: 'pull_chapter',
          status: 'running',
          metadata: { chapter_id: 'ch-1' },
          retries: 0,
          max_retries: 3,
          created_at: '',
          updated_at: '',
        },
      ];

      rerender({ chapters: initialChapters, mangaJobs: runningJobs });
      expect(result.current.optimisticPullIds.has('ch-1')).toBe(true);
    });
  });

  describe('addToLibraryMutation', () => {
    it('returns null and does not call API when manga is not provided', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      let data: any;
      await act(async () => {
        data = await result.current.addToLibraryMutation.mutateAsync();
      });

      expect(data).toBeNull();
      expect(api.importProviderManga).not.toHaveBeenCalled();
    });

    it('imports manga with provided parameters, invalidates queries, and shows success toast', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
      const mockManga: Manga = { id: 'm-1', title: 'Test Manga', sourceId: 'src-1', contentRemoteId: 'rem-1' };

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'm-1',
            manga: mockManga,
            providerIdParam: 'custom-provider',
            remoteIdParam: 'custom-remote',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.addToLibraryMutation.mutateAsync();
      });

      expect(api.importProviderManga).toHaveBeenCalledWith('custom-provider', 'custom-remote', 'plan_to_read');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.library.all });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.manga.all });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.all });
      expect(showToast).toHaveBeenCalledWith('Series added to library', 'success');
    });

    it('falls back to manga fields when providerIdParam and remoteIdParam are not provided', async () => {
      const mockManga: Manga = { id: 'm-fallback', title: 'Test Manga', sourceId: 'mangadex', contentRemoteId: 'dex-1' };

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'm-fallback',
            manga: mockManga,
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.addToLibraryMutation.mutateAsync();
      });

      expect(api.importProviderManga).toHaveBeenCalledWith('mangadex', 'dex-1', 'plan_to_read');
    });

    it('handles error in addToLibrary with string details and object details formatting', async () => {
      const errWithStringDetails = Object.assign(new Error('Import failed'), {
        details: 'Provider unreachable',
      });
      vi.mocked(api.importProviderManga).mockRejectedValueOnce(errWithStringDetails);

      const mockManga: Manga = { id: 'm-1', title: 'Test Manga' };
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'm-1',
            manga: mockManga,
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.addToLibraryMutation.mutate();
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to add to library: Import failed',
          'error',
          'Provider unreachable'
        );
      });

      // Error with object details
      const errWithObjDetails = Object.assign(new Error('Validation error'), {
        details: { code: 400, reason: 'Invalid ID' },
      });
      vi.mocked(api.importProviderManga).mockRejectedValueOnce(errWithObjDetails);

      await act(async () => {
        result.current.addToLibraryMutation.mutate();
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to add to library: Validation error',
          'error',
          JSON.stringify({ code: 400, reason: 'Invalid ID' }, null, 2)
        );
      });
    });
  });

  describe('refreshChaptersMutation', () => {
    it('throws error when targetMangaId is empty', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      let caughtError: any;
      await act(async () => {
        try {
          await result.current.refreshChaptersMutation.mutateAsync();
        } catch (e) {
          caughtError = e;
        }
      });

      expect(caughtError).toEqual(new Error('No manga ID'));
    });

    it('handles refresh with 0 added chapters by showing "Up to date" toast', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
      vi.mocked(api.refreshLibraryManga).mockResolvedValue({ added: 0, orphaned: 0, provider_id: 'mangafox' });

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.refreshChaptersMutation.mutateAsync();
      });

      expect(api.refreshLibraryManga).toHaveBeenCalledWith('manga-1');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.list('manga-1') });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.manga.details('manga-1') });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.library.refresh('manga-1') });
      expect(showToast).toHaveBeenCalledWith('Up to date', 'success');
    });

    it('handles refresh with >0 added chapters by showing count toast', async () => {
      vi.mocked(api.refreshLibraryManga).mockResolvedValue({ added: 5, orphaned: 0, provider_id: 'mangafox' });

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.refreshChaptersMutation.mutateAsync();
      });

      expect(showToast).toHaveBeenCalledWith('Refresh complete: 5 chapter(s)', 'success');
    });

    it('handles error in refreshChaptersMutation and shows error toast', async () => {
      vi.mocked(api.refreshLibraryManga).mockRejectedValue(new Error('Network failure'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.refreshChaptersMutation.mutate();
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Refresh failed: Network failure',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('removeChapterMutation', () => {
    it('throws error when targetMangaId is empty', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      let caughtError: any;
      await act(async () => {
        try {
          await result.current.removeChapterMutation.mutateAsync({ chapterId: 'ch-1', providerId: 'mangafox' });
        } catch (e) {
          caughtError = e;
        }
      });

      expect(caughtError).toEqual(new Error('No manga ID'));
    });

    it('deletes chapter, invalidates provider-specific and general list queries, and shows toast', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.removeChapterMutation.mutateAsync({ chapterId: 'ch-1', providerId: 'mangafox' });
      });

      expect(api.deleteChapter).toHaveBeenCalledWith('manga-1', 'ch-1', 'mangafox');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.list('manga-1') });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.providerList('manga-1', 'mangafox') });
      expect(showToast).toHaveBeenCalledWith('Chapter removed from library', 'success');
    });

    it('handles removeChapter failure with error toast', async () => {
      vi.mocked(api.deleteChapter).mockRejectedValue(new Error('Chapter not found'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.removeChapterMutation.mutate({ chapterId: 'ch-1', providerId: 'mangafox' });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Remove failed: Chapter not found',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('deleteChapterFilesMutation', () => {
    it('throws error when targetMangaId is empty', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      let caughtError: any;
      await act(async () => {
        try {
          await result.current.deleteChapterFilesMutation.mutateAsync({ chapterId: 'ch-1', providerId: 'mangafox' });
        } catch (e) {
          caughtError = e;
        }
      });

      expect(caughtError).toEqual(new Error('No manga ID'));
    });

    it('deletes files, invalidates queries, and shows success toast', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.deleteChapterFilesMutation.mutateAsync({ chapterId: 'ch-1', providerId: 'mangafox' });
      });

      expect(api.deleteChapterFiles).toHaveBeenCalledWith('manga-1', 'mangafox', 'ch-1');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.list('manga-1') });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.providerList('manga-1', 'mangafox') });
      expect(showToast).toHaveBeenCalledWith('Files deleted (chapter entry preserved)', 'success');
    });

    it('handles deleteChapterFiles failure with error toast', async () => {
      vi.mocked(api.deleteChapterFiles).mockRejectedValue(new Error('Disk locked'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.deleteChapterFilesMutation.mutate({ chapterId: 'ch-1', providerId: 'mangafox' });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Delete files failed: Disk locked',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('pullChapterMutation and handlePullChapter', () => {
    it('throws error when targetMangaId is empty in pullChapterMutation', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      let caughtError: any;
      await act(async () => {
        try {
          await result.current.pullChapterMutation.mutateAsync({ chapterId: 'ch-1', providerId: 'mangafox' });
        } catch (e) {
          caughtError = e;
        }
      });

      expect(caughtError).toEqual(new Error('No manga ID'));
    });

    it('handlePullChapter sets optimistic pull ID, executes mutation, and invalidates queries', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handlePullChapter('ch-1', 'mangafox');
      });

      expect(result.current.optimisticPullIds.has('ch-1')).toBe(true);

      await waitFor(() => {
        expect(api.pullChapter).toHaveBeenCalledWith('manga-1', 'mangafox', 'ch-1');
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.pages('ch-1', 'manga-1', 'mangafox') });
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.list('manga-1') });
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.providerList('manga-1', 'mangafox') });
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.jobs.all });
        expect(showToast).toHaveBeenCalledWith('Chapter pulled from provider', 'success');
      });
    });

    it('handles pullChapter failure with error toast', async () => {
      vi.mocked(api.pullChapter).mockRejectedValue(new Error('Downloader offline'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.pullChapterMutation.mutate({ chapterId: 'ch-1', providerId: 'mangafox' });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Pull failed: Downloader offline',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('switchContentProvider (switchToMutation)', () => {
    it('throws error when targetMangaId is empty', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      let caughtError: any;
      await act(async () => {
        try {
          await result.current.switchToMutation.mutateAsync({
            provider: { provider_id: 'mangafox', provider_manga_id: 'mf-1' },
          });
        } catch (e) {
          caughtError = e;
        }
      });

      expect(caughtError).toEqual(new Error('No manga ID'));
    });

    it('switches content provider, invalidates manga and chapter queries, and shows toast', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
      const provider: ProviderRef = { provider_id: 'mangafox', provider_manga_id: 'mf-1' };

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.switchToMutation.mutateAsync({ provider });
      });

      expect(api.switchContentProvider).toHaveBeenCalledWith('manga-1', 'mangafox', 'mf-1');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.manga.details('manga-1') });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.library.all });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.all });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.list('manga-1') });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.providerList('manga-1', 'mangafox') });
      expect(showToast).toHaveBeenCalledWith('Switched content provider', 'success');
    });

    it('handles switchTo failure with error toast', async () => {
      vi.mocked(api.switchContentProvider).mockRejectedValue(new Error('Provider not bound'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.switchToMutation.mutate({
          provider: { provider_id: 'mangafox', provider_manga_id: 'mf-1' },
        });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Switch failed: Provider not bound',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('removeProvider (removeProviderMutation)', () => {
    it('throws error when targetMangaId is empty', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      let caughtError: any;
      await act(async () => {
        try {
          await result.current.removeProviderMutation.mutateAsync({
            provider_id: 'mangafox',
            provider_manga_id: 'mf-1',
          });
        } catch (e) {
          caughtError = e;
        }
      });

      expect(caughtError).toEqual(new Error('No manga ID'));
    });

    it('removes provider, invalidates manga details and library queries, and shows toast', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
      const provider: ProviderRef = { provider_id: 'mangafox', provider_manga_id: 'mf-1' };

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.removeProviderMutation.mutateAsync(provider);
      });

      expect(api.removeProvider).toHaveBeenCalledWith('manga-1', 'mangafox', 'mf-1');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.manga.details('manga-1') });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.library.all });
      expect(showToast).toHaveBeenCalledWith('Provider removed', 'success');
    });

    it('handles removeProvider failure with error toast', async () => {
      vi.mocked(api.removeProvider).mockRejectedValue(new Error('Cannot remove primary provider'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        result.current.removeProviderMutation.mutate({
          provider_id: 'mangafox',
          provider_manga_id: 'mf-1',
        });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Remove failed: Cannot remove primary provider',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('Batch operations', () => {
    it('handleBatchPull early returns if missing targetMangaId or activeContentProviderId', () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchPull(['c1', 'c2']);
      });
      expect(api.batchPullChapters).not.toHaveBeenCalled();
    });

    it('handleBatchPull executes batch pull, tracks optimistic IDs, invalidates jobs, and shows toast', async () => {
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchPull(['c1', 'c2']);
      });

      expect(result.current.optimisticPullIds.has('c1')).toBe(true);
      expect(result.current.optimisticPullIds.has('c2')).toBe(true);

      await waitFor(() => {
        expect(api.batchPullChapters).toHaveBeenCalledWith('manga-1', 'mangafox', ['c1', 'c2']);
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.jobs.all });
        expect(showToast).toHaveBeenCalledWith('Pull enqueued for 2 chapter(s)', 'success');
      });
    });

    it('handleBatchPull handles error with error toast', async () => {
      vi.mocked(api.batchPullChapters).mockRejectedValue(new Error('Batch pull failed'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchPull(['c1', 'c2']);
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to pull chapters: Batch pull failed',
          'error',
          expect.any(String)
        );
      });
    });

    it('handleBatchRefresh early returns if missing parameters or executes successfully', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchRefresh(['c1', 'c2']);
      });

      await waitFor(() => {
        expect(api.batchRefreshChapters).toHaveBeenCalledWith('manga-1', 'mangafox', ['c1', 'c2']);
        expect(showToast).toHaveBeenCalledWith('Refreshed metadata for 2 chapter(s)', 'success');
      });
    });

    it('handleBatchRefresh handles error with error toast', async () => {
      vi.mocked(api.batchRefreshChapters).mockRejectedValue(new Error('Provider down'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchRefresh(['c1', 'c2']);
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to refresh chapter metadata: Provider down',
          'error',
          expect.any(String)
        );
      });
    });

    it('handleBatchDeleteFiles early returns if missing parameters or executes successfully', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchDeleteFiles(['c1', 'c2']);
      });

      await waitFor(() => {
        expect(api.batchDeleteChapterFiles).toHaveBeenCalledWith('manga-1', 'mangafox', ['c1', 'c2']);
        expect(showToast).toHaveBeenCalledWith('Deleted files for 2 chapter(s)', 'success');
      });
    });

    it('handleBatchDeleteFiles handles error with error toast', async () => {
      vi.mocked(api.batchDeleteChapterFiles).mockRejectedValue(new Error('File locked'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchDeleteFiles(['c1', 'c2']);
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to delete files: File locked',
          'error',
          expect.any(String)
        );
      });
    });

    it('handleBatchRemove early returns if missing parameters or executes successfully', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchRemove(['c1', 'c2']);
      });

      await waitFor(() => {
        expect(api.batchDeleteChapters).toHaveBeenCalledWith('manga-1', 'mangafox', ['c1', 'c2']);
        expect(showToast).toHaveBeenCalledWith('Removed 2 chapter(s) from library', 'success');
      });
    });

    it('handleBatchRemove handles error with error toast', async () => {
      vi.mocked(api.batchDeleteChapters).mockRejectedValue(new Error('DB error'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchRemove(['c1', 'c2']);
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to remove chapters: DB error',
          'error',
          expect.any(String)
        );
      });
    });

    it('handleBatchUpdateProgress updates progress to read and unread', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchUpdateProgress(['c1', 'c2'], { is_read: true });
      });

      await waitFor(() => {
        expect(api.batchUpdateChapterProgress).toHaveBeenCalledWith('manga-1', 'mangafox', ['c1', 'c2'], { is_read: true });
        expect(showToast).toHaveBeenCalledWith('Marked 2 chapter(s) as read', 'success');
      });

      act(() => {
        result.current.handleBatchUpdateProgress(['c1', 'c2'], { is_read: false });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith('Marked 2 chapter(s) as unread', 'success');
      });
    });

    it('handleBatchUpdateProgress handles error with error toast', async () => {
      vi.mocked(api.batchUpdateChapterProgress).mockRejectedValue(new Error('Progress error'));

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleBatchUpdateProgress(['c1', 'c2'], { is_read: true });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to update chapter progress: Progress error',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('handleUserMetadataChange', () => {
    it('early returns if targetMangaId is empty', () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: '',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleUserMetadataChange({ user_status: 'reading' });
      });

      expect(api.patchLibraryManga).not.toHaveBeenCalled();
    });

    it('patches metadata and handles error with toast', async () => {
      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleUserMetadataChange({ user_status: 'reading' });
      });

      await waitFor(() => {
        expect(api.patchLibraryManga).toHaveBeenCalledWith('manga-1', { user_status: 'reading' });
      });

      vi.mocked(api.patchLibraryManga).mockRejectedValueOnce(new Error('Patch failed'));

      act(() => {
        result.current.handleUserMetadataChange({ user_status: 'completed' });
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith(
          'Failed to update metadata: Patch failed',
          'error',
          expect.any(String)
        );
      });
    });
  });

  describe('Derived computed properties', () => {
    it('computes pullingChapterIds correctly including activeJobChapterIds and excluding downloaded chapters from optimistic set', async () => {
      const chapters: Chapter[] = [
        { id: 'ch-opt-1', name: 'Chapter 1', number: 1, is_downloaded: false },
        { id: 'ch-opt-2', name: 'Chapter 2', number: 2, is_downloaded: true },
      ];

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeJobChapterIds: ['ch-active-1'],
            chapters,
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handlePullChapter('ch-opt-1', 'mangafox');
        result.current.handlePullChapter('ch-opt-2', 'mangafox');
      });

      await waitFor(() => {
        expect(api.pullChapter).toHaveBeenCalledWith('manga-1', 'mangafox', 'ch-opt-1');
        expect(api.pullChapter).toHaveBeenCalledWith('manga-1', 'mangafox', 'ch-opt-2');
      });

      expect(result.current.pullingChapterIds).toContain('ch-active-1');
      expect(result.current.pullingChapterIds).toContain('ch-opt-1');
      expect(result.current.pullingChapterIds).not.toContain('ch-opt-2');
    });

    it('computes deletingFilesChapterIds and removingChapterIds from single and batch operations', async () => {
      let resolveDeleteFiles: () => void;
      const deleteFilesPromise = new Promise<void>((resolve) => {
        resolveDeleteFiles = resolve;
      });
      vi.mocked(api.deleteChapterFiles).mockReturnValue(deleteFilesPromise as any);

      let resolveBatchRemove: (value: any) => void;
      const batchRemovePromise = new Promise<any>((resolve) => {
        resolveBatchRemove = resolve;
      });
      vi.mocked(api.batchDeleteChapters).mockReturnValue(batchRemovePromise as any);

      const { result } = renderHook(
        () =>
          useChapterOperations({
            targetMangaId: 'manga-1',
            activeContentProviderId: 'mangafox',
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.deleteChapterFilesMutation.mutate({ chapterId: 'ch-del-1', providerId: 'mangafox' });
        result.current.handleBatchRemove(['ch-rem-1', 'ch-rem-2']);
      });

      await waitFor(() => {
        expect(result.current.deletingFilesChapterIds).toContain('ch-del-1');
        expect(result.current.removingChapterIds).toContain('ch-rem-1');
        expect(result.current.removingChapterIds).toContain('ch-rem-2');
      });

      act(() => {
        resolveDeleteFiles!();
        resolveBatchRemove!({ removed: 2 });
      });

      await waitFor(() => {
        expect(result.current.deletingFilesChapterIds).toHaveLength(0);
        expect(result.current.removingChapterIds).toHaveLength(0);
      });
    });
  });
});

