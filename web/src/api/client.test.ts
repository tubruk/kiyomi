import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { api } from './client';
import { ChapterMeta, ProviderRef } from '../types/api';

describe('API Client - Zero-Collision Endpoints', () => {
  const originalFetch = global.fetch;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  const mockJsonResponse = (data: any, status = 200) => {
    global.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(data), {
        status,
        headers: { 'Content-Type': 'application/json' },
      })
    );
  };

  const mockEmptyResponse = (status = 204) => {
    global.fetch = vi.fn().mockResolvedValue(
      new Response(null, { status })
    );
  };

  describe('Single chapter zero-collision endpoints', () => {
    const mangaId = 'manga-123';
    const chapterId = 'ch-456';

    it('getChapter calls GET /library/manga/:mangaId/chapters/:chapterId', async () => {
      mockJsonResponse({ id: chapterId, manga_id: mangaId, number: 1 });

      const result = await api.getChapter(mangaId, chapterId);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/chapters/${chapterId}`);
      expect(init?.method ?? 'GET').toBe('GET');
      expect(result).toEqual({ id: chapterId, manga_id: mangaId, number: 1 });
    });

    it('saveChapter calls POST /library/manga/:mangaId/chapters', async () => {
      const meta: ChapterMeta = {
        title: 'Prologue',
        number: 1,
      };
      mockJsonResponse({ id: chapterId, manga_id: mangaId, meta });

      const result = await api.saveChapter(mangaId, chapterId, meta);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/chapters`);
      expect(init?.method).toBe('POST');
      expect(JSON.parse(init?.body as string)).toEqual({
        id: chapterId,
        meta,
      });
      expect(result).toEqual({ id: chapterId, manga_id: mangaId, meta });
    });

    it('deleteChapter calls DELETE /library/manga/:mangaId/chapters/:chapterId', async () => {
      mockEmptyResponse();

      await api.deleteChapter(mangaId, chapterId);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/chapters/${chapterId}`);
      expect(init?.method).toBe('DELETE');
    });

    it('deleteChapterFiles calls DELETE /library/manga/:mangaId/chapters/:chapterId/files', async () => {
      mockEmptyResponse();

      await api.deleteChapterFiles(mangaId, chapterId);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/chapters/${chapterId}/files`);
      expect(init?.method).toBe('DELETE');
    });

    it('updateChapterProgress calls PATCH /library/manga/:mangaId/chapters/:chapterId/progress without providerId', async () => {
      const progress = { is_read: true, last_read_page: 25 };
      mockJsonResponse({ id: chapterId, manga_id: mangaId, meta: { is_read: true, last_read_page: 25 } });

      const result = await api.updateChapterProgress(mangaId, chapterId, progress);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/chapters/${chapterId}/progress`);
      expect(init?.method).toBe('PATCH');
      expect(JSON.parse(init?.body as string)).toEqual(progress);
      expect(result).toEqual({ id: chapterId, manga_id: mangaId, meta: { is_read: true, last_read_page: 25 } });
    });

    it('pullChapter calls POST /library/manga/:mangaId/chapters/:chapterId/pull', async () => {
      mockJsonResponse({ pages: [], job_ids: ['job-1'] });

      const result = await api.pullChapter(mangaId, chapterId);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/chapters/${chapterId}/pull`);
      expect(init?.method).toBe('POST');
      expect(result).toEqual({ pages: [], job_ids: ['job-1'] });
    });

    it('getChapterPages calls GET /library/manga/:mangaId/chapters/:chapterId/pages', async () => {
      mockJsonResponse({ pages: [{ index: 1, url: 'https://example.com/1.jpg' }] });

      const result = await api.getChapterPages(mangaId, chapterId);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/chapters/${chapterId}/pages`);
      expect(init?.method ?? 'GET').toBe('GET');
      expect(result).toEqual({ pages: [{ index: 1, url: 'https://example.com/1.jpg' }] });
    });
  });

  describe('Batch chapter zero-collision endpoints', () => {
    const mangaId = 'manga-abc';
    const chapterIds = ['c-1', 'c-2', 'c-3'];

    it('batchUpdateChapterProgress calls POST /library/manga/:mangaId/batch/chapters/progress', async () => {
      mockJsonResponse({ updated: 3, chapter_ids: chapterIds });

      const result = await api.batchUpdateChapterProgress(mangaId, chapterIds, { is_read: true, last_read_page: 10 });

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/batch/chapters/progress`);
      expect(init?.method).toBe('POST');
      expect(JSON.parse(init?.body as string)).toEqual({
        chapter_ids: chapterIds,
        is_read: true,
        last_read_page: 10,
      });
      expect(result).toEqual({ updated: 3, chapter_ids: chapterIds });
    });

    it('batchPullChapters calls POST /library/manga/:mangaId/batch/chapters/pull', async () => {
      mockJsonResponse({ chapter_count: 3, job_ids: ['j1', 'j2'], job_count: 2 });

      const result = await api.batchPullChapters(mangaId, chapterIds);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/batch/chapters/pull`);
      expect(init?.method).toBe('POST');
      expect(JSON.parse(init?.body as string)).toEqual({
        chapter_ids: chapterIds,
      });
      expect(result).toEqual({ chapter_count: 3, job_ids: ['j1', 'j2'], job_count: 2 });
    });

    it('refreshChaptersBatch calls POST /library/manga/:mangaId/batch/chapters/refresh', async () => {
      mockJsonResponse({ refreshed: 3, chapter_ids: chapterIds });

      const result = await api.refreshChaptersBatch(mangaId, chapterIds);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/batch/chapters/refresh`);
      expect(init?.method).toBe('POST');
      expect(JSON.parse(init?.body as string)).toEqual({
        chapter_ids: chapterIds,
      });
      expect(result).toEqual({ refreshed: 3, chapter_ids: chapterIds });
    });

    it('batchDeleteChapters calls POST /library/manga/:mangaId/batch/chapters/delete', async () => {
      mockJsonResponse({ removed: 3 });

      const result = await api.batchDeleteChapters(mangaId, chapterIds);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/batch/chapters/delete`);
      expect(init?.method).toBe('POST');
      expect(JSON.parse(init?.body as string)).toEqual({
        chapter_ids: chapterIds,
      });
      expect(result).toEqual({ removed: 3 });
    });

    it('deleteChapterFilesBatch calls POST /library/manga/:mangaId/batch/chapter-files/delete', async () => {
      mockJsonResponse({ deleted: 3 });

      const result = await api.deleteChapterFilesBatch(mangaId, chapterIds);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/batch/chapter-files/delete`);
      expect(init?.method).toBe('POST');
      expect(JSON.parse(init?.body as string)).toEqual({
        chapter_ids: chapterIds,
      });
      expect(result).toEqual({ deleted: 3 });
    });
  });

  describe('Provider bindings endpoints', () => {
    const mangaId = 'manga-bind-1';

    it('addProvider calls POST /library/manga/:mangaId/bindings', async () => {
      const binding: ProviderRef = {
        provider_id: 'mangadex',
        provider_manga_id: 'md-100',
        manga_title: 'Test Manga',
      };
      mockJsonResponse({ success: true, binding });

      const result = await api.addProvider(mangaId, binding);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/bindings`);
      expect(init?.method).toBe('POST');
      expect(JSON.parse(init?.body as string)).toEqual(binding);
      expect(result).toEqual({ success: true, binding });
    });

    it('removeProvider calls DELETE /library/manga/:mangaId/bindings/:providerId/:providerMangaId', async () => {
      mockEmptyResponse();

      await api.removeProvider(mangaId, 'mangadex', 'md-100');

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/bindings/mangadex/md-100`);
      expect(init?.method).toBe('DELETE');
    });

    it('switchContentProvider calls PATCH /library/manga/:mangaId/bindings/content', async () => {
      mockJsonResponse({ success: true, active_provider: 'mangadex' });

      const result = await api.switchContentProvider(mangaId, 'mangadex', 'md-100');

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/bindings/content`);
      expect(init?.method).toBe('PATCH');
      expect(JSON.parse(init?.body as string)).toEqual({
        provider_id: 'mangadex',
        provider_manga_id: 'md-100',
      });
      expect(result).toEqual({ success: true, active_provider: 'mangadex' });
    });

    it('listProviders calls GET /library/manga/:mangaId/bindings', async () => {
      const providers = [{ provider_id: 'mangadex', provider_manga_id: 'md-100' }];
      mockJsonResponse({ providers });

      const result = await api.listProviders(mangaId);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = vi.mocked(global.fetch).mock.calls[0];
      expect(url).toBe(`/api/v1/library/manga/${mangaId}/bindings`);
      expect(init?.method ?? 'GET').toBe('GET');
      expect(result).toEqual(providers);
    });
  });
});
