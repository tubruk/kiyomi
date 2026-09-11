import { useState, useDeferredValue, useMemo, useCallback } from 'react';
import { Manga } from '../../../types/api';

export interface ShelfItem {
  id: string;
  label: string;
  isFavorite?: boolean;
}

export const LIBRARY_SORT_OPTIONS: Record<string, string> = {
  title_asc: 'Title (A to Z)',
  title_desc: 'Title (Z to A)',
  rating_desc: 'Rating (Highest)',
  rating_asc: 'Rating (Lowest)',
  added_desc: 'Recently Added',
  updated_desc: 'Recently Updated',
};

export const DEFAULT_SHELVES: ShelfItem[] = [
  { id: 'all', label: 'All' },
  { id: 'reading', label: 'Reading' },
  { id: 'favorites', label: 'Favorites', isFavorite: true },
  { id: 'plan_to_read', label: 'Plan to Read' },
  { id: 'completed', label: 'Completed' },
  { id: 'on_hold', label: 'On Hold' },
  { id: 'dropped', label: 'Dropped' },
  { id: 'unread', label: 'Unread' },
];

export interface UseLibraryFiltersOptions {
  libraryManga: Manga[];
  initialSortBy?: string;
  initialShelf?: string;
  initialTag?: string;
  initialSearch?: string;
}

export interface UseLibraryFiltersReturn {
  activeShelf: string;
  setActiveShelf: (shelf: string) => void;
  selectedTag: string;
  setSelectedTag: (tag: string) => void;
  filterSearch: string;
  setFilterSearch: (search: string) => void;
  sortBy: string;
  setSortBy: (sortBy: string) => void;
  customShelves: string[];
  allTags: string[];
  allShelves: ShelfItem[];
  counts: Record<string, number>;
  filteredManga: Manga[];
  clearFilters: () => void;
}

export function useLibraryFilters({
  libraryManga,
  initialSortBy = 'title_asc',
  initialShelf = 'all',
  initialTag = '',
  initialSearch = '',
}: UseLibraryFiltersOptions): UseLibraryFiltersReturn {
  const [activeShelf, setActiveShelf] = useState<string>(initialShelf);
  const [selectedTag, setSelectedTag] = useState<string>(initialTag);
  const [filterSearch, setFilterSearch] = useState<string>(initialSearch);
  const [sortBy, setSortBy] = useState<string>(initialSortBy);

  const deferredFilterSearch = useDeferredValue(filterSearch);

  // Shelves & Tags Extraction — memoized to avoid recompute on every filter change
  const { customShelves, allTags } = useMemo(() => {
    const shelves = Array.from(
      new Set(libraryManga.flatMap((m) => m.shelves || []))
    ).filter(Boolean);
    const tags = Array.from(
      new Set(
        libraryManga.flatMap((m) => m.metadata?.tags ?? m.tags ?? m.genres ?? [])
      )
    ).sort();
    return { customShelves: shelves, allTags: tags };
  }, [libraryManga]);

  const allShelves = useMemo(
    () => [
      ...DEFAULT_SHELVES,
      ...customShelves.map((s) => ({ id: `custom:${s}`, label: s })),
    ],
    [customShelves]
  );

  // Counts per status, favorites, and custom shelves
  const counts = useMemo(() => {
    const res: Record<string, number> = {
      all: libraryManga.length,
      reading: 0,
      favorites: 0,
      plan_to_read: 0,
      completed: 0,
      on_hold: 0,
      dropped: 0,
      unread: 0,
    };
    for (const m of libraryManga) {
      const st = m.user_state?.status ?? m.userStatus ?? m.user_status ?? m.meta?.user_status ?? 'unread';
      if (res[st] !== undefined) {
        res[st]++;
      }
      if (m.user_state?.favorite ?? m.userFavorite ?? m.user_favorite ?? m.meta?.user_favorite) {
        res.favorites++;
      }
      if (m.shelves) {
        for (const s of m.shelves) {
          const key = 'custom:' + s;
          res[key] = (res[key] || 0) + 1;
        }
      }
    }
    return res;
  }, [libraryManga]);

  // Filtered and Sorted Manga List
  const filteredManga = useMemo(() => {
    const filtered = libraryManga.filter((manga) => {
      const uStatus =
        manga.user_state?.status ?? manga.userStatus ?? manga.user_status ?? manga.meta?.user_status ?? 'unread';
      const isFav =
        manga.user_state?.favorite ?? manga.userFavorite ?? manga.user_favorite ?? manga.meta?.user_favorite;

      if (activeShelf === 'reading' && uStatus !== 'reading') return false;
      if (activeShelf === 'favorites' && !isFav) return false;
      if (activeShelf === 'plan_to_read' && uStatus !== 'plan_to_read') return false;
      if (activeShelf === 'completed' && uStatus !== 'completed') return false;
      if (activeShelf === 'on_hold' && uStatus !== 'on_hold') return false;
      if (activeShelf === 'dropped' && uStatus !== 'dropped') return false;
      if (activeShelf === 'unread' && uStatus !== 'unread') return false;
      if (activeShelf.startsWith('custom:')) {
        const shelfName = activeShelf.replace('custom:', '');
        if (!manga.shelves?.includes(shelfName)) return false;
      }

      if (selectedTag) {
        const mangaTags = manga.metadata?.tags ?? manga.tags ?? manga.genres ?? [];
        if (!mangaTags.includes(selectedTag)) return false;
      }

      if (deferredFilterSearch.trim()) {
        const q = deferredFilterSearch.toLowerCase();
        const titleMatch = (manga.metadata?.title ?? manga.title ?? '').toLowerCase().includes(q);
        const authorList =
          manga.metadata?.authors ??
          manga.authors ??
          (manga.author ? [manga.author] : []);
        const authorMatch = authorList.some((a) => a.toLowerCase().includes(q));
        if (!titleMatch && !authorMatch) return false;
      }

      return true;
    });

    return [...filtered].sort((a, b) => {
      const aTitle = a.metadata?.title ?? a.title ?? '';
      const bTitle = b.metadata?.title ?? b.title ?? '';
      if (sortBy === 'title_asc') {
        return aTitle.localeCompare(bTitle);
      }
      if (sortBy === 'title_desc') {
        return bTitle.localeCompare(aTitle);
      }
      if (sortBy === 'rating_desc') {
        const rA = a.user_state?.rating ?? a.userRating ?? a.user_rating ?? a.meta?.user_rating ?? 0;
        const rB = b.user_state?.rating ?? b.userRating ?? b.user_rating ?? b.meta?.user_rating ?? 0;
        if (rB !== rA) return rB - rA;
        return aTitle.localeCompare(bTitle);
      }
      if (sortBy === 'rating_asc') {
        const rA = a.user_state?.rating ?? a.userRating ?? a.user_rating ?? a.meta?.user_rating ?? 0;
        const rB = b.user_state?.rating ?? b.userRating ?? b.user_rating ?? b.meta?.user_rating ?? 0;
        if (rA !== rB) return rA - rB;
        return aTitle.localeCompare(bTitle);
      }
      if (sortBy === 'added_desc') {
        const tA = a.user_state?.added_at
          ? new Date(a.user_state.added_at).getTime()
          : a.meta?.added_at
          ? new Date(a.meta.added_at).getTime()
          : 0;
        const tB = b.user_state?.added_at
          ? new Date(b.user_state.added_at).getTime()
          : b.meta?.added_at
          ? new Date(b.meta.added_at).getTime()
          : 0;
        return tB - tA;
      }
      if (sortBy === 'updated_desc') {
        const tA = a.user_state?.updated_at
          ? new Date(a.user_state.updated_at).getTime()
          : a.meta?.updated_at
          ? new Date(a.meta.updated_at).getTime()
          : 0;
        const tB = b.user_state?.updated_at
          ? new Date(b.user_state.updated_at).getTime()
          : b.meta?.updated_at
          ? new Date(b.meta.updated_at).getTime()
          : 0;
        return tB - tA;
      }
      return 0;
    });
  }, [libraryManga, activeShelf, selectedTag, deferredFilterSearch, sortBy]);

  const clearFilters = useCallback(() => {
    setActiveShelf('all');
    setSelectedTag('');
    setFilterSearch('');
  }, []);

  return {
    activeShelf,
    setActiveShelf,
    selectedTag,
    setSelectedTag,
    filterSearch,
    setFilterSearch,
    sortBy,
    setSortBy,
    customShelves,
    allTags,
    allShelves,
    counts,
    filteredManga,
    clearFilters,
  };
}
