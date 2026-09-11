import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useMergeLibraryMangaMutation,
  useUpdateChapterProgressMutation,
  useChapterPages,
  useBatchUpdateChapterProgressMutation,
  useBatchPullChaptersMutation,
  useBatchRefreshChaptersMutation,
  useBatchDeleteChapterFilesMutation,
  useBatchDeleteChaptersMutation,
} from './hooks';
import { api, MergeLibraryMangaPayload } from './client';
import * as toastContext from '../context/ToastContext';
import * as router from '@tanstack/react-router';
import { queryKeys } from '../lib/queryKeys';
import { Manga } from '../types/api';

vi.mock('../context/ToastContext', () => ({
  useToast: vi.fn(),
}));

vi.mock('@tanstack/react-router', () => ({
  useNavigate: vi.fn(),
}));

describe('useMergeLibraryMangaMutation', () => {
  const showToast = vi.fn();
  const navigate = vi.fn();
  let queryClient: QueryClient;

  const buildPayload = (): MergeLibraryMangaPayload => ({
    keep_manga_id: 'm-keep',
    source_manga_ids: ['m-src-1', 'm-src-2'],
    content_provider: {
      provider_id: 'mangafox',
      provider_manga_id: 'mf-1',
    },
    metadata: {
      title: 'keep',
      description: 'source:m-src-1',
      aliases: 'merge',
      tags: 'merge',
      authors: 'merge',
      artists: 'merge',
      publishers: 'merge',
      release_year: 'merge',
      cover_url: 'keep',
      providers: 'merge',
    },
  });

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.mocked(toastContext.useToast).mockReturnValue({ showToast } as any);
    vi.mocked(router.useNavigate).mockReturnValue(navigate);
    vi.spyOn(api, 'mergeLibraryManga').mockResolvedValue({
      id: 'm-keep',
      title: 'Merged Title',
    } as Manga);
  });

  const createWrapper = () => {
    return ({ children }: { children: React.ReactNode }) =>
      React.createElement(QueryClientProvider, { client: queryClient }, children);
  };

  it('calls api.mergeLibraryManga with the provided snake_case payload', async () => {
    const payload = buildPayload();
    const { result } = renderHook(() => useMergeLibraryMangaMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync(payload);
    });

    expect(api.mergeLibraryManga).toHaveBeenCalledTimes(1);
    expect(api.mergeLibraryManga).toHaveBeenCalledWith(payload);
    // Ensure snake_case keys pass through unchanged (no camelCase transformation).
    const callArg = vi.mocked(api.mergeLibraryManga).mock.calls[0][0];
    expect(callArg).toEqual({
      keep_manga_id: 'm-keep',
      source_manga_ids: ['m-src-1', 'm-src-2'],
      content_provider: {
        provider_id: 'mangafox',
        provider_manga_id: 'mf-1',
      },
      metadata: expect.objectContaining({
        title: 'keep',
        cover_url: 'keep',
        release_year: 'merge',
        providers: 'merge',
      }),
    });
  });

  it('on success: invalidates library, manga, and chapters query key groups', async () => {
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useMergeLibraryMangaMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync(buildPayload());
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.library.all });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.manga.all });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.all });
  });

  it('on success: navigates to /manga/$keep_manga_id', async () => {
    const { result } = renderHook(() => useMergeLibraryMangaMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync(buildPayload());
    });

    expect(navigate).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith({
      to: '/manga/$mangaId',
      params: { mangaId: 'm-keep' },
    });
  });

  it('on error: shows an error toast and does not navigate', async () => {
    const errWithStringDetails = Object.assign(new Error('Merge conflict'), {
      details: 'Provider mismatch',
    });
    vi.mocked(api.mergeLibraryManga).mockRejectedValueOnce(errWithStringDetails);

    const { result } = renderHook(() => useMergeLibraryMangaMutation(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate(buildPayload());
    });

    await waitFor(() => {
      expect(showToast).toHaveBeenCalledWith(
        'Merge failed: Merge conflict',
        'error',
        'Provider mismatch'
      );
    });

    expect(navigate).not.toHaveBeenCalled();
  });
});

describe('useUpdateChapterProgressMutation', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.spyOn(api, 'updateChapterProgress').mockResolvedValue({ id: 'c1', manga_id: 'm1', meta: {} });
  });

  const createWrapper = () => ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);

  it('calls api.updateChapterProgress without providerId', async () => {
    const { result } = renderHook(() => useUpdateChapterProgressMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        mangaId: 'm1',
        chapterId: 'c1',
        isRead: true,
        lastReadPage: 12,
      });
    });

    expect(api.updateChapterProgress).toHaveBeenCalledTimes(1);
    expect(api.updateChapterProgress).toHaveBeenCalledWith('m1', 'c1', {
      is_read: true,
      last_read_page: 12,
    });
  });

  it('invalidates relevant queries on progress update success', async () => {
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useUpdateChapterProgressMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        mangaId: 'm1',
        chapterId: 'c1',
        progress: { is_read: false },
      });
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.list('m1') });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.manga.details('m1') });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.library.mangas() });
  });
});

describe('useChapterPages', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.spyOn(api, 'getChapterPages').mockResolvedValue({
      pages: [
        { index: 1, url: 'https://example.com/p1.jpg' },
        { index: 2, url: 'https://example.com/p2.jpg' },
      ],
    });
  });

  const createWrapper = () => ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);

  it('fetches chapter pages using (mangaId, chapterId) signature', async () => {
    const { result } = renderHook(() => useChapterPages('m1', 'c1', { enabled: true }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(api.getChapterPages).toHaveBeenCalledTimes(1);
    expect(api.getChapterPages).toHaveBeenCalledWith('m1', 'c1');
    expect(result.current.data?.pages).toHaveLength(2);
  });
});

describe('Batch chapter mutations', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.spyOn(api, 'batchUpdateChapterProgress').mockResolvedValue({ updated: 3, chapter_ids: ['c1', 'c2', 'c3'] });
    vi.spyOn(api, 'batchPullChapters').mockResolvedValue({ chapter_count: 2, job_ids: ['j1'], job_count: 1 });
    vi.spyOn(api, 'batchRefreshChapters').mockResolvedValue({ refreshed: 2, chapter_ids: ['c1', 'c2'] });
    vi.spyOn(api, 'batchDeleteChapterFiles').mockResolvedValue({ deleted: 2 });
    vi.spyOn(api, 'batchDeleteChapters').mockResolvedValue({ removed: 2 });
  });

  const createWrapper = () => ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);

  it('useBatchUpdateChapterProgressMutation calls api with mangaId and chapterIds', async () => {
    const { result } = renderHook(() => useBatchUpdateChapterProgressMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        mangaId: 'm1',
        chapterIds: ['c1', 'c2', 'c3'],
        isRead: true,
      });
    });

    expect(api.batchUpdateChapterProgress).toHaveBeenCalledWith('m1', ['c1', 'c2', 'c3'], {
      is_read: true,
    });
  });

  it('useBatchPullChaptersMutation calls api with mangaId and chapterIds', async () => {
    const { result } = renderHook(() => useBatchPullChaptersMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        mangaId: 'm1',
        chapterIds: ['c1', 'c2'],
      });
    });

    expect(api.batchPullChapters).toHaveBeenCalledWith('m1', ['c1', 'c2']);
  });

  it('useBatchRefreshChaptersMutation calls api with mangaId and chapterIds', async () => {
    const { result } = renderHook(() => useBatchRefreshChaptersMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        mangaId: 'm1',
        chapterIds: ['c1', 'c2'],
      });
    });

    expect(api.batchRefreshChapters).toHaveBeenCalledWith('m1', ['c1', 'c2']);
  });

  it('useBatchDeleteChapterFilesMutation calls api with mangaId and chapterIds', async () => {
    const { result } = renderHook(() => useBatchDeleteChapterFilesMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        mangaId: 'm1',
        chapterIds: ['c1', 'c2'],
      });
    });

    expect(api.batchDeleteChapterFiles).toHaveBeenCalledWith('m1', ['c1', 'c2']);
  });

  it('useBatchDeleteChaptersMutation calls api with mangaId and chapterIds', async () => {
    const { result } = renderHook(() => useBatchDeleteChaptersMutation(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        mangaId: 'm1',
        chapterIds: ['c1', 'c2'],
      });
    });

    expect(api.batchDeleteChapters).toHaveBeenCalledWith('m1', ['c1', 'c2']);
  });
});