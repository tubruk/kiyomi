import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useChapterFilters } from './useChapterFilters';
import { Chapter } from '../../../types/api';

const mockChapters: Chapter[] = [
  {
    id: 'ch-1',
    name: 'Chapter 1: The Beginning',
    number: 1,
    title: 'Chapter 1: The Beginning',
    uploadDate: '2023-01-01T00:00:00Z',
    is_downloaded: true,
  },
  {
    id: 'ch-2',
    name: 'Chapter 2: The Journey',
    number: 2,
    title: 'Chapter 2: The Journey',
    uploadDate: '2023-01-02T00:00:00Z',
    is_downloaded: false,
  },
  {
    id: 'ch-3',
    name: 'Chapter 3: The Climax',
    number: 3,
    title: 'Chapter 3: The Climax',
    uploadDate: '2023-01-03T00:00:00Z',
    is_downloaded: true,
  },
];

describe('useChapterFilters', () => {
  it('initializes with default source sort and pagination', () => {
    const { result } = renderHook(() =>
      useChapterFilters({
        chapters: mockChapters,
        sortBy: 'source',
        order: 'asc',
        pageSize: 2,
      })
    );

    expect(result.current.filterQuery).toBe('');
    expect(result.current.page).toBe(1);
    expect(result.current.totalPages).toBe(2);
    expect(result.current.safePage).toBe(1);
    expect(result.current.filteredChapters.length).toBe(3);
    expect(result.current.paginatedChapters.length).toBe(2);
    expect(result.current.downloadedCount).toBe(2);
  });

  it('filters chapters by title or chapter number', () => {
    const { result } = renderHook(() =>
      useChapterFilters({
        chapters: mockChapters,
        sortBy: 'number',
        order: 'asc',
      })
    );

    act(() => {
      result.current.setFilterQuery('Journey');
    });

    expect(result.current.filteredChapters.length).toBe(1);
    expect(result.current.filteredChapters[0].id).toBe('ch-2');

    act(() => {
      result.current.setFilterQuery('3');
    });

    expect(result.current.filteredChapters.length).toBe(1);
    expect(result.current.filteredChapters[0].id).toBe('ch-3');
  });

  it('sorts chapters by chapter number asc and desc', () => {
    const { result, rerender } = renderHook(
      ({ order }: { order: 'asc' | 'desc' }) =>
        useChapterFilters({
          chapters: mockChapters,
          sortBy: 'number',
          order,
        }),
      { initialProps: { order: 'asc' } }
    );

    expect(result.current.filteredChapters[0].number).toBe(1);
    expect(result.current.filteredChapters[2].number).toBe(3);

    rerender({ order: 'desc' });

    expect(result.current.filteredChapters[0].number).toBe(3);
    expect(result.current.filteredChapters[2].number).toBe(1);
  });

  it('sorts chapters by upload date', () => {
    const { result, rerender } = renderHook(
      ({ order }: { order: 'asc' | 'desc' }) =>
        useChapterFilters({
          chapters: mockChapters,
          sortBy: 'date',
          order,
        }),
      { initialProps: { order: 'asc' } }
    );

    expect(result.current.filteredChapters[0].id).toBe('ch-1');
    expect(result.current.filteredChapters[2].id).toBe('ch-3');

    rerender({ order: 'desc' });

    expect(result.current.filteredChapters[0].id).toBe('ch-3');
    expect(result.current.filteredChapters[2].id).toBe('ch-1');
  });

  it('handles pagination page changes properly', () => {
    const { result } = renderHook(() =>
      useChapterFilters({
        chapters: mockChapters,
        sortBy: 'number',
        order: 'asc',
        pageSize: 2,
      })
    );

    expect(result.current.safePage).toBe(1);
    expect(result.current.paginatedChapters.map((c) => c.id)).toEqual(['ch-1', 'ch-2']);

    act(() => {
      result.current.setPage(2);
    });

    expect(result.current.safePage).toBe(2);
    expect(result.current.paginatedChapters.map((c) => c.id)).toEqual(['ch-3']);
  });
});
