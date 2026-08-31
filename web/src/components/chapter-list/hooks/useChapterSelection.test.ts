import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useChapterSelection } from './useChapterSelection';
import { Chapter } from '../../../types/api';

const mockChapters: Chapter[] = [
  { id: 'ch-1', name: 'Chapter 1', number: 1, title: 'Chapter 1', is_downloaded: true },
  { id: 'ch-2', name: 'Chapter 2', number: 2, title: 'Chapter 2', is_downloaded: false },
  { id: 'ch-3', name: 'Chapter 3', number: 3, title: 'Chapter 3', is_downloaded: false },
  { id: 'ch-4', name: 'Chapter 4', number: 4, title: 'Chapter 4', is_downloaded: true },
];

describe('useChapterSelection', () => {
  it('manages selection mode and single chapter toggle', () => {
    const { result } = renderHook(() =>
      useChapterSelection({
        chapters: mockChapters,
        filteredChapters: mockChapters,
        mangaId: 'manga-1',
      })
    );

    expect(result.current.isSelectionMode).toBe(false);
    expect(result.current.selectedCount).toBe(0);

    act(() => {
      result.current.setIsSelectionMode(true);
      result.current.handleToggleRow('ch-1');
    });

    expect(result.current.isSelectionMode).toBe(true);
    expect(result.current.selectedCount).toBe(1);
    expect(result.current.selectedChapterIds.has('ch-1')).toBe(true);
    expect(result.current.hasDownloadedInSelection).toBe(true);

    // Toggle off
    act(() => {
      result.current.handleToggleRow('ch-1');
    });
    expect(result.current.selectedCount).toBe(0);
  });

  it('supports select all, deselect all, and invert selection', () => {
    const { result } = renderHook(() =>
      useChapterSelection({
        chapters: mockChapters,
        filteredChapters: mockChapters,
        mangaId: 'manga-1',
      })
    );

    act(() => {
      result.current.handleSelectAll();
    });
    expect(result.current.selectedCount).toBe(4);

    act(() => {
      result.current.handleSelectNone();
    });
    expect(result.current.selectedCount).toBe(0);

    act(() => {
      result.current.handleToggleRow('ch-2');
    });
    expect(result.current.selectedCount).toBe(1);

    act(() => {
      result.current.handleInvertSelection();
    });
    expect(result.current.selectedCount).toBe(3);
    expect(result.current.selectedChapterIds.has('ch-2')).toBe(false);
    expect(result.current.selectedChapterIds.has('ch-1')).toBe(true);
    expect(result.current.selectedChapterIds.has('ch-3')).toBe(true);
    expect(result.current.selectedChapterIds.has('ch-4')).toBe(true);
  });

  it('supports Shift+Click range selection', () => {
    const { result } = renderHook(() =>
      useChapterSelection({
        chapters: mockChapters,
        filteredChapters: mockChapters,
        mangaId: 'manga-1',
      })
    );

    // First select ch-1
    act(() => {
      result.current.handleToggleRow('ch-1');
    });

    // Shift click ch-3
    act(() => {
      result.current.handleToggleRow('ch-3', { shiftKey: true } as React.MouseEvent);
    });

    expect(result.current.selectedCount).toBe(3);
    expect(result.current.selectedChapterIds.has('ch-1')).toBe(true);
    expect(result.current.selectedChapterIds.has('ch-2')).toBe(true);
    expect(result.current.selectedChapterIds.has('ch-3')).toBe(true);
    expect(result.current.selectedChapterIds.has('ch-4')).toBe(false);
  });

  it('resets selection state when mangaId changes', () => {
    let mangaId = 'manga-1';
    const { result, rerender } = renderHook(() =>
      useChapterSelection({
        chapters: mockChapters,
        filteredChapters: mockChapters,
        mangaId,
      })
    );

    act(() => {
      result.current.setIsSelectionMode(true);
      result.current.handleToggleRow('ch-1');
    });
    expect(result.current.selectedCount).toBe(1);
    expect(result.current.isSelectionMode).toBe(true);

    mangaId = 'manga-2';
    rerender();

    expect(result.current.selectedCount).toBe(0);
    expect(result.current.isSelectionMode).toBe(false);
  });

  it('handles batch remove confirmation flow', () => {
    const onBatchRemove = vi.fn();
    const { result } = renderHook(() =>
      useChapterSelection({
        chapters: mockChapters,
        filteredChapters: mockChapters,
        mangaId: 'manga-1',
        onBatchRemove,
      })
    );

    act(() => {
      result.current.handleToggleRow('ch-1');
      result.current.handleToggleRow('ch-2');
      result.current.setIsConfirmRemoveOpen(true);
    });

    expect(result.current.isConfirmRemoveOpen).toBe(true);

    act(() => {
      result.current.handleConfirmRemove();
    });

    expect(onBatchRemove).toHaveBeenCalledWith(['ch-1', 'ch-2']);
    expect(result.current.isConfirmRemoveOpen).toBe(false);
    expect(result.current.selectedCount).toBe(0);
  });
});
