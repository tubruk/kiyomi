import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useReaderProgressTracker } from './useReaderProgressTracker';

describe('useReaderProgressTracker', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('restores initial page to 1 by default', () => {
    const onUpdateProgress = vi.fn();

    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        pagesCount: 20,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 0,
        hasNextChapter: true,
        onUpdateProgress,
      })
    );

    expect(result.current.currentPage).toBe(1);
  });

  it('restores initial page to last page when searchPage is "last"', () => {
    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        pagesCount: 25,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 5,
        searchPage: 'last',
        hasNextChapter: true,
      })
    );

    expect(result.current.currentPage).toBe(25);
  });

  it('restores initial page from numerical searchPage query param', () => {
    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        pagesCount: 25,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 2,
        searchPage: 12,
        hasNextChapter: true,
      })
    );

    expect(result.current.currentPage).toBe(12);
  });

  it('restores initial page to chapterLastReadPage when unread', () => {
    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        pagesCount: 25,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 8,
        hasNextChapter: true,
      })
    );

    expect(result.current.currentPage).toBe(8);
  });

  it('calls onScrollToPage in continuous non-paged mode when restoring page > 1', () => {
    const onScrollToPage = vi.fn();

    renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        pagesCount: 25,
        isPaged: false,
        isChapterRead: false,
        chapterLastReadPage: 14,
        hasNextChapter: true,
        onScrollToPage,
      })
    );

    act(() => {
      vi.runAllTimers();
    });

    expect(onScrollToPage).toHaveBeenCalledWith(14);
  });

  it('debounces progress updates by 1500ms on intermediate page changes', () => {
    const onUpdateProgress = vi.fn();

    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        chapterProviderId: 'provider-1',
        pagesCount: 20,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 1,
        hasNextChapter: true,
        onUpdateProgress,
      })
    );

    act(() => {
      result.current.setCurrentPage(5);
    });

    expect(onUpdateProgress).not.toHaveBeenCalled();

    // Fast-forward 1000ms - still debouncing
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(onUpdateProgress).not.toHaveBeenCalled();

    // Fast-forward another 600ms (1600ms total)
    act(() => {
      vi.advanceTimersByTime(600);
    });

    expect(onUpdateProgress).toHaveBeenCalledWith({
      mangaId: 'm-1',
      chapterId: 'ch-1',
      providerId: 'provider-1',
      progress: { last_read_page: 5 },
    });
  });

  it('auto marks as read when reaching the last page', () => {
    const onUpdateProgress = vi.fn();

    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        pagesCount: 10,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 1,
        hasNextChapter: true,
        onUpdateProgress,
      })
    );

    act(() => {
      result.current.setCurrentPage(10);
    });

    expect(onUpdateProgress).toHaveBeenCalledWith({
      mangaId: 'm-1',
      chapterId: 'ch-1',
      providerId: undefined,
      progress: { is_read: true, last_read_page: 10 },
    });
  });

  it('triggers completion prompt dialog on last page of latest chapter when status is reading', () => {
    const onUpdateProgress = vi.fn();

    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-latest',
        pagesCount: 10,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 1,
        hasNextChapter: false,
        userStatus: 'reading',
        onUpdateProgress,
      })
    );

    expect(result.current.showCompletionDialog).toBe(false);

    act(() => {
      result.current.setCurrentPage(10);
    });

    expect(result.current.showCompletionDialog).toBe(true);
  });

  it('markChapterRead immediately triggers update progress with is_read', () => {
    const onUpdateProgress = vi.fn();

    const { result } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId: 'ch-1',
        pagesCount: 10,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage: 1,
        hasNextChapter: true,
        onUpdateProgress,
      })
    );

    act(() => {
      result.current.markChapterRead(10);
    });

    expect(onUpdateProgress).toHaveBeenCalledWith({
      mangaId: 'm-1',
      chapterId: 'ch-1',
      providerId: undefined,
      progress: { is_read: true, last_read_page: 10 },
    });
  });

  it('resets state when chapterId changes', () => {
    const onUpdateProgress = vi.fn();

    let chapterId = 'ch-1';
    let chapterLastReadPage = 5;

    const { result, rerender } = renderHook(() =>
      useReaderProgressTracker({
        effectiveMangaId: 'm-1',
        chapterId,
        pagesCount: 20,
        isPaged: true,
        isChapterRead: false,
        chapterLastReadPage,
        hasNextChapter: true,
        onUpdateProgress,
      })
    );

    expect(result.current.currentPage).toBe(5);

    // Switch to chapter 2
    chapterId = 'ch-2';
    chapterLastReadPage = 12;
    rerender();

    expect(result.current.currentPage).toBe(12);
  });
});
