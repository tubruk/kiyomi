import { useState, useMemo, useEffect } from 'react';
import { Chapter } from '../../../types/api';

export const SORT_OPTIONS: Record<string, string> = {
  source: 'Source Order',
  number: 'Chapter Number',
  date: 'Upload Date',
};

export interface UseChapterFiltersOptions {
  chapters: Chapter[];
  sortBy?: string;
  order?: 'asc' | 'desc';
  pageSize?: number;
}

export interface UseChapterFiltersReturn {
  filterQuery: string;
  setFilterQuery: (query: string) => void;
  page: number;
  setPage: (page: number) => void;
  totalPages: number;
  safePage: number;
  filteredChapters: Chapter[];
  paginatedChapters: Chapter[];
  downloadedCount: number;
}

export function useChapterFilters({
  chapters,
  sortBy = 'source',
  order = 'desc',
  pageSize = 50,
}: UseChapterFiltersOptions): UseChapterFiltersReturn {
  const [filterQuery, setFilterQuery] = useState('');
  const [page, setPage] = useState(1);

  // Reset page when filter query, sortBy, or order changes
  const handleSetFilterQuery = (query: string) => {
    setFilterQuery(query);
    setPage(1);
  };

  useEffect(() => {
    setPage(1);
  }, [sortBy, order]);

  const filteredChapters = useMemo(() => {
    let result = chapters.map((ch, idx) => ({
      ...ch,
      sourceOrder: ch.sourceOrder ?? idx,
    }));

    if (filterQuery.trim()) {
      const q = filterQuery.toLowerCase().trim();
      result = result.filter(
        (c) =>
          (c.title || c.name || '').toLowerCase().includes(q) ||
          String(c.number).includes(q)
      );
    }

    return [...result].sort((a, b) => {
      let comparison = 0;
      if (sortBy === 'number') {
        comparison = (a.number ?? 0) - (b.number ?? 0);
      } else if (sortBy === 'date') {
        const dateA = a.uploadDate || a.uploadedAt || a.meta?.upload_date || '';
        const dateB = b.uploadDate || b.uploadedAt || b.meta?.upload_date || '';
        comparison = dateA.localeCompare(dateB);
      } else {
        // sortBy === 'source'
        comparison = (a.sourceOrder ?? 0) - (b.sourceOrder ?? 0);
      }

      return order === 'asc' ? comparison : -comparison;
    });
  }, [chapters, filterQuery, sortBy, order]);

  const totalPages = Math.ceil(filteredChapters.length / pageSize);
  const safePage = Math.min(Math.max(1, page), totalPages || 1);
  const paginatedChapters = useMemo(() => {
    return filteredChapters.slice((safePage - 1) * pageSize, safePage * pageSize);
  }, [filteredChapters, safePage, pageSize]);

  const downloadedCount = useMemo(() => {
    return chapters.reduce((count, c) => {
      const isDownloaded = Boolean(
        c.is_downloaded ??
        (c as any).isDownloaded ??
        c.meta?.is_downloaded ??
        c.isDownloaded
      );
      return isDownloaded ? count + 1 : count;
    }, 0);
  }, [chapters]);

  return {
    filterQuery,
    setFilterQuery: handleSetFilterQuery,
    page,
    setPage,
    totalPages,
    safePage,
    filteredChapters,
    paginatedChapters,
    downloadedCount,
  };
}
