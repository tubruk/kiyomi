import { useState, useMemo, useEffect, useCallback } from 'react';
import { Chapter } from '../../../types/api';

export interface UseChapterSelectionOptions {
  chapters: Chapter[];
  filteredChapters: Chapter[];
  mangaId?: string;
  onBatchRemove?: (chapterIds: string[]) => void;
}

export interface UseChapterSelectionReturn {
  isSelectionMode: boolean;
  setIsSelectionMode: (val: boolean) => void;
  selectedChapterIds: Set<string>;
  setSelectedChapterIds: React.Dispatch<React.SetStateAction<Set<string>>>;
  lastSelectedId: string | null;
  selectedCount: number;
  hasDownloadedInSelection: boolean;
  isConfirmRemoveOpen: boolean;
  setIsConfirmRemoveOpen: (open: boolean) => void;
  handleSelectAll: () => void;
  handleSelectNone: () => void;
  handleInvertSelection: () => void;
  handleExitSelectionMode: () => void;
  handleToggleRow: (chapterId: string, event?: React.MouseEvent | React.KeyboardEvent) => void;
  handleConfirmRemove: () => void;
}

export function useChapterSelection({
  chapters,
  filteredChapters,
  mangaId,
  onBatchRemove,
}: UseChapterSelectionOptions): UseChapterSelectionReturn {
  const [isSelectionMode, setIsSelectionMode] = useState(false);
  const [selectedChapterIds, setSelectedChapterIds] = useState<Set<string>>(new Set());
  const [lastSelectedId, setLastSelectedId] = useState<string | null>(null);
  const [isConfirmRemoveOpen, setIsConfirmRemoveOpen] = useState(false);

  // Reset selection mode when manga changes
  useEffect(() => {
    setIsSelectionMode(false);
    setSelectedChapterIds(new Set());
    setLastSelectedId(null);
  }, [mangaId]);

  const selectedCount = selectedChapterIds.size;

  const hasDownloadedInSelection = useMemo(() => {
    if (selectedChapterIds.size === 0) return false;
    return chapters.some((c) => {
      if (!selectedChapterIds.has(c.id)) return false;
      const isDownloaded = Boolean(
        c.is_downloaded ??
        (c as any).isDownloaded ??
        c.meta?.is_downloaded ??
        c.isDownloaded
      );
      const downloadedPages =
        c.downloaded_pages ??
        (c as any).downloadedPages ??
        c.meta?.downloaded_pages ??
        c.downloadedPages ??
        0;
      return isDownloaded || downloadedPages > 0 || Boolean(c.meta?.downloaded_at);
    });
  }, [chapters, selectedChapterIds]);

  const handleSelectAll = useCallback(() => {
    const allFilteredIds = filteredChapters.map((c) => c.id);
    setSelectedChapterIds(new Set(allFilteredIds));
  }, [filteredChapters]);

  const handleSelectNone = useCallback(() => {
    setSelectedChapterIds(new Set());
    setLastSelectedId(null);
  }, []);

  const handleInvertSelection = useCallback(() => {
    const next = new Set<string>();
    filteredChapters.forEach((c) => {
      if (!selectedChapterIds.has(c.id)) {
        next.add(c.id);
      }
    });
    setSelectedChapterIds(next);
  }, [filteredChapters, selectedChapterIds]);

  const handleExitSelectionMode = useCallback(() => {
    setIsSelectionMode(false);
    setSelectedChapterIds(new Set());
    setLastSelectedId(null);
  }, []);

  const handleToggleRow = useCallback(
    (chapterId: string, event?: React.MouseEvent | React.KeyboardEvent) => {
      if (event && 'shiftKey' in event && event.shiftKey && lastSelectedId !== null) {
        const idx1 = filteredChapters.findIndex((c) => c.id === lastSelectedId);
        const idx2 = filteredChapters.findIndex((c) => c.id === chapterId);
        if (idx1 !== -1 && idx2 !== -1) {
          const start = Math.min(idx1, idx2);
          const end = Math.max(idx1, idx2);
          const rangeIds = filteredChapters.slice(start, end + 1).map((c) => c.id);
          setSelectedChapterIds((prev) => {
            const next = new Set(prev);
            rangeIds.forEach((id) => next.add(id));
            return next;
          });
          setLastSelectedId(chapterId);
          return;
        }
      }

      setSelectedChapterIds((prev) => {
        const next = new Set(prev);
        if (next.has(chapterId)) {
          next.delete(chapterId);
        } else {
          next.add(chapterId);
        }
        return next;
      });
      setLastSelectedId(chapterId);
    },
    [filteredChapters, lastSelectedId]
  );

  const handleConfirmRemove = useCallback(() => {
    if (!onBatchRemove) return;
    const ids = Array.from(selectedChapterIds);
    onBatchRemove(ids);
    setIsConfirmRemoveOpen(false);
    setSelectedChapterIds(new Set());
    setLastSelectedId(null);
  }, [onBatchRemove, selectedChapterIds]);

  return {
    isSelectionMode,
    setIsSelectionMode,
    selectedChapterIds,
    setSelectedChapterIds,
    lastSelectedId,
    selectedCount,
    hasDownloadedInSelection,
    isConfirmRemoveOpen,
    setIsConfirmRemoveOpen,
    handleSelectAll,
    handleSelectNone,
    handleInvertSelection,
    handleExitSelectionMode,
    handleToggleRow,
    handleConfirmRemove,
  };
}
