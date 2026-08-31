import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useLibraryFilters } from './useLibraryFilters';
import { Manga } from '../../../types/api';

const mockMangaList: Manga[] = [
  {
    id: 'm-1',
    title: 'Attack on Titan',
    authors: ['Hajime Isayama'],
    userStatus: 'reading',
    userFavorite: true,
    userRating: 9,
    shelves: ['Shonen', 'Favorites2'],
    tags: ['Action', 'Fantasy'],
    meta: {
      added_at: '2023-01-01T00:00:00Z',
      updated_at: '2023-01-05T00:00:00Z',
    },
  },
  {
    id: 'm-2',
    title: 'Berserk',
    authors: ['Kentaro Miura'],
    userStatus: 'completed',
    userFavorite: false,
    userRating: 10,
    shelves: ['Seinen'],
    tags: ['Dark Fantasy', 'Action'],
    meta: {
      added_at: '2023-01-02T00:00:00Z',
      updated_at: '2023-01-04T00:00:00Z',
    },
  },
  {
    id: 'm-3',
    title: 'Chainsaw Man',
    authors: ['Tatsuki Fujimoto'],
    userStatus: 'plan_to_read',
    userFavorite: true,
    userRating: 7,
    shelves: ['Shonen'],
    tags: ['Comedy', 'Action'],
    meta: {
      added_at: '2023-01-03T00:00:00Z',
      updated_at: '2023-01-03T00:00:00Z',
    },
  },
];

describe('useLibraryFilters', () => {
  it('extracts custom shelves, tags, and status counts', () => {
    const { result } = renderHook(() =>
      useLibraryFilters({
        libraryManga: mockMangaList,
      })
    );

    expect(result.current.customShelves).toEqual(['Shonen', 'Favorites2', 'Seinen']);
    expect(result.current.allTags).toEqual(['Action', 'Comedy', 'Dark Fantasy', 'Fantasy']);
    expect(result.current.counts.all).toBe(3);
    expect(result.current.counts.reading).toBe(1);
    expect(result.current.counts.completed).toBe(1);
    expect(result.current.counts.favorites).toBe(2);
    expect(result.current.counts['custom:Shonen']).toBe(2);
  });

  it('filters by status shelf and custom shelf', () => {
    const { result } = renderHook(() =>
      useLibraryFilters({
        libraryManga: mockMangaList,
      })
    );

    act(() => {
      result.current.setActiveShelf('reading');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-1']);

    act(() => {
      result.current.setActiveShelf('custom:Seinen');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-2']);
  });

  it('filters by search query matching title or author', () => {
    const { result } = renderHook(() =>
      useLibraryFilters({
        libraryManga: mockMangaList,
      })
    );

    act(() => {
      result.current.setFilterSearch('Miura');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-2']);

    act(() => {
      result.current.setFilterSearch('Titan');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-1']);
  });

  it('filters by tag', () => {
    const { result } = renderHook(() =>
      useLibraryFilters({
        libraryManga: mockMangaList,
      })
    );

    act(() => {
      result.current.setSelectedTag('Comedy');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-3']);
  });

  it('sorts by title, rating, added, and updated', () => {
    const { result } = renderHook(() =>
      useLibraryFilters({
        libraryManga: mockMangaList,
        initialSortBy: 'title_desc',
      })
    );
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-3', 'm-2', 'm-1']);

    act(() => {
      result.current.setSortBy('rating_desc');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-2', 'm-1', 'm-3']);

    act(() => {
      result.current.setSortBy('added_desc');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-3', 'm-2', 'm-1']);

    act(() => {
      result.current.setSortBy('updated_desc');
    });
    expect(result.current.filteredManga.map((m) => m.id)).toEqual(['m-1', 'm-2', 'm-3']);
  });

  it('clears all filters properly', () => {
    const { result } = renderHook(() =>
      useLibraryFilters({
        libraryManga: mockMangaList,
      })
    );

    act(() => {
      result.current.setActiveShelf('reading');
      result.current.setSelectedTag('Fantasy');
      result.current.setFilterSearch('Titan');
    });

    expect(result.current.activeShelf).toBe('reading');
    expect(result.current.selectedTag).toBe('Fantasy');
    expect(result.current.filterSearch).toBe('Titan');

    act(() => {
      result.current.clearFilters();
    });

    expect(result.current.activeShelf).toBe('all');
    expect(result.current.selectedTag).toBe('');
    expect(result.current.filterSearch).toBe('');
  });
});
