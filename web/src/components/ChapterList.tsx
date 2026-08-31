import React, { useState, useMemo, useEffect, useCallback } from 'react';
import {
  AlertCircle,
  ArrowDown,
  ArrowUp,
  BookOpen,
  HardDrive,
  ListChecks,
  RefreshCw,
  Search,
} from 'lucide-react';
import { Chapter } from '../types/api';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Skeleton } from './ui/skeleton';
import { cn } from '../lib/utils';
import { Pagination } from './Pagination';
import { ChapterRow } from './chapter-list/ChapterRow';
import { BatchActionBar } from './chapter-list/BatchActionBar';
import { BatchRemoveDialog } from './chapter-list/BatchRemoveDialog';

const sortOptions: Record<string, string> = {
  source: 'Source Order',
  number: 'Chapter Number',
  date: 'Upload Date',
};

export interface ChapterListProps {
  chapters: Chapter[];
  mangaId: string;
  sortBy: string;
  order: 'asc' | 'desc';
  onSortByChange: (newSort: string) => void;
  onOrderToggle: () => void;
  isLoading: boolean;
  isError: boolean;
  contentProviderName?: string;
  providerName?: string;
  isUnavailable?: boolean;
  hasNoContentProvider?: boolean;
  isInLibrary?: boolean;
  providerId?: string;
  remoteId?: string;
  onRefreshChapters?: () => void;
  isRefreshing?: boolean;
  onRemoveChapter?: (chapterId: string, providerId?: string) => void;
  onDeleteFiles?: (chapterId: string, providerId: string) => void;
  onPullChapter?: (chapterId: string, providerId: string) => void;
  isDeletingFiles?: boolean;
  isPullingChapter?: boolean;
  isRemovingChapter?: boolean;
  pullingChapterIds?: string[];
  deletingFilesChapterIds?: string[];
  removingChapterIds?: string[];
  // Batch Mutations props
  onBatchUpdateProgress?: (chapterIds: string[], progress: { is_read?: boolean; last_read_page?: number }) => void;
  isBatchUpdatingProgress?: boolean;
  onBatchPull?: (chapterIds: string[]) => void;
  isBatchPulling?: boolean;
  onBatchRefresh?: (chapterIds: string[]) => void;
  isBatchRefreshing?: boolean;
  onBatchDeleteFiles?: (chapterIds: string[]) => void;
  isBatchDeletingFiles?: boolean;
  onBatchRemove?: (chapterIds: string[]) => void;
  isBatchRemoving?: boolean;
}

export const ChapterList: React.FC<ChapterListProps> = ({
  chapters,
  mangaId,
  sortBy,
  order,
  onSortByChange,
  onOrderToggle,
  isLoading,
  isError,
  contentProviderName,
  providerName,
  isUnavailable = false,
  hasNoContentProvider = false,
  isInLibrary = false,
  providerId,
  remoteId,
  onRefreshChapters,
  isRefreshing = false,
  onRemoveChapter,
  onDeleteFiles,
  onPullChapter,
  pullingChapterIds = [],
  deletingFilesChapterIds = [],
  removingChapterIds = [],
  onBatchUpdateProgress,
  isBatchUpdatingProgress = false,
  onBatchPull,
  isBatchPulling = false,
  onBatchRefresh,
  isBatchRefreshing = false,
  onBatchDeleteFiles,
  isBatchDeletingFiles = false,
  onBatchRemove,
  isBatchRemoving = false,
}) => {
  const [filterQuery, setFilterQuery] = useState('');
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 50;

  // Selection Mode State
  const [isSelectionMode, setIsSelectionMode] = useState(false);
  const [selectedChapterIds, setSelectedChapterIds] = useState<Set<string>>(new Set());
  const [lastSelectedId, setLastSelectedId] = useState<string | null>(null);
  const [isConfirmRemoveOpen, setIsConfirmRemoveOpen] = useState(false);

  const pullingSet = useMemo(() => new Set(pullingChapterIds), [pullingChapterIds]);
  const deletingFilesSet = useMemo(() => new Set(deletingFilesChapterIds), [deletingFilesChapterIds]);
  const removingSet = useMemo(() => new Set(removingChapterIds), [removingChapterIds]);

  // Reset selection mode when manga changes
  useEffect(() => {
    setIsSelectionMode(false);
    setSelectedChapterIds(new Set());
    setLastSelectedId(null);
  }, [mangaId]);

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

  const totalPages = Math.ceil(filteredChapters.length / PAGE_SIZE);
  const safePage = Math.min(Math.max(1, page), totalPages || 1);
  const paginatedChapters = filteredChapters.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  const resolvedProviderName = contentProviderName;

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

  // Selection Helpers
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

  const handleBatchUpdateProgress = useCallback(
    (progress: { is_read?: boolean; last_read_page?: number }) => {
      if (selectedChapterIds.size === 0 || !onBatchUpdateProgress) return;
      onBatchUpdateProgress(Array.from(selectedChapterIds), progress);
    },
    [onBatchUpdateProgress, selectedChapterIds]
  );

  const handleBatchRefresh = useCallback(() => {
    if (selectedChapterIds.size === 0 || !onBatchRefresh) return;
    onBatchRefresh(Array.from(selectedChapterIds));
  }, [onBatchRefresh, selectedChapterIds]);

  const handleBatchPull = useCallback(() => {
    if (selectedChapterIds.size === 0 || !onBatchPull) return;
    onBatchPull(Array.from(selectedChapterIds));
  }, [onBatchPull, selectedChapterIds]);

  const handleBatchDeleteFiles = useCallback(() => {
    if (
      selectedChapterIds.size === 0 ||
      !onBatchDeleteFiles ||
      !hasDownloadedInSelection
    )
      return;
    onBatchDeleteFiles(Array.from(selectedChapterIds));
  }, [onBatchDeleteFiles, hasDownloadedInSelection, selectedChapterIds]);

  const handleConfirmRemove = useCallback(() => {
    if (!onBatchRemove) return;
    const ids = Array.from(selectedChapterIds);
    onBatchRemove(ids);
    setIsConfirmRemoveOpen(false);
    setSelectedChapterIds(new Set());
    setLastSelectedId(null);
  }, [onBatchRemove, selectedChapterIds]);

  if (hasNoContentProvider) {
    return (
      <div className="flex flex-col gap-4 mt-8">
        <div className="flex items-center justify-between border-b border-border/50 pb-3">
          <h2 className="text-xl font-bold tracking-tight text-foreground">Chapters (0)</h2>
        </div>
        <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-card p-10 text-center">
          <BookOpen className="size-10 text-muted-foreground/40 mb-3" aria-hidden />
          <h3 className="text-base font-semibold text-foreground">No content provider linked</h3>
          <p className="mt-1 max-w-md text-xs text-muted-foreground mb-4">
            This series does not have a content provider linked yet to fetch chapters and read content.
          </p>
        </div>
      </div>
    );
  }

  if (isUnavailable) {
    const displayProvider = providerName || resolvedProviderName || 'Provider';
    return (
      <div className="flex flex-col gap-4 mt-8">
        <div className="flex items-center justify-between border-b border-border/50 pb-3">
          <h2 className="text-xl font-bold tracking-tight text-foreground">Chapters (0)</h2>
        </div>
        <div className="flex flex-col items-center justify-center rounded-xl border border-destructive/20 bg-destructive/5 p-10 text-center">
          <AlertCircle className="size-10 text-destructive/70 mb-3" aria-hidden />
          <h3 className="text-base font-semibold text-foreground">
            Content Unavailable in {displayProvider}
          </h3>
          <p className="mt-1 max-w-md text-xs text-muted-foreground">
            This series is marked as unavailable on {displayProvider}. Chapters cannot be retrieved or read.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 mt-8">
      {/* Selection Mode Active Toolbar */}
      {isSelectionMode && (
        <BatchActionBar
          selectedCount={selectedCount}
          totalCount={filteredChapters.length}
          hasDownloadedInSelection={hasDownloadedInSelection}
          onSelectAll={handleSelectAll}
          onSelectNone={handleSelectNone}
          onInvertSelection={handleInvertSelection}
          onExitSelectionMode={handleExitSelectionMode}
          onBatchUpdateProgress={onBatchUpdateProgress ? handleBatchUpdateProgress : undefined}
          isBatchUpdatingProgress={isBatchUpdatingProgress}
          onBatchPull={onBatchPull ? handleBatchPull : undefined}
          isBatchPulling={isBatchPulling}
          onBatchRefresh={onBatchRefresh ? handleBatchRefresh : undefined}
          isBatchRefreshing={isBatchRefreshing}
          onBatchDeleteFiles={onBatchDeleteFiles ? handleBatchDeleteFiles : undefined}
          isBatchDeletingFiles={isBatchDeletingFiles}
          onOpenRemoveDialog={() => setIsConfirmRemoveOpen(true)}
          isBatchRemoving={isBatchRemoving}
          canPull={Boolean(onBatchPull)}
          canDeleteFiles={Boolean(onBatchDeleteFiles)}
          canRemove={Boolean(onBatchRemove)}
        />
      )}

      {/* Header & Controls Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-border/50 pb-3">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="text-xl font-bold tracking-tight text-foreground">
            Chapters ({chapters.length})
          </h2>
          {isInLibrary && chapters.length > 0 && downloadedCount > 0 && (
            <span
              className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
              title={`${downloadedCount} of ${chapters.length} chapters downloaded`}
            >
              <HardDrive className="size-3 text-emerald-500" />
              {downloadedCount}/{chapters.length}
            </span>
          )}
          {resolvedProviderName && (
            <span className="text-xs text-muted-foreground">
              Provided by <span className="font-medium text-foreground">{resolvedProviderName}</span>
            </span>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-3">
          {onRefreshChapters && (
            <Button
              variant="outline"
              size="sm"
              onClick={onRefreshChapters}
              disabled={isRefreshing}
              className="h-9 px-3 text-xs bg-card border-border gap-1.5 cursor-pointer"
              title="Refresh chapter list from content provider"
              aria-label="Refresh chapter list"
            >
              <RefreshCw className={cn('size-3.5', isRefreshing && 'animate-spin')} aria-hidden />
              {isRefreshing ? 'Refreshing…' : 'Refresh'}
            </Button>
          )}

          {/* Select Mode Button (when in library and has chapters) */}
          {isInLibrary && chapters.length > 0 && !isSelectionMode && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setIsSelectionMode(true);
                setSelectedChapterIds(new Set());
                setLastSelectedId(null);
              }}
              className="h-9 px-2.5 text-xs bg-card border-border gap-1.5 cursor-pointer"
              title="Enter selection mode for batch actions"
              aria-label="Select chapters"
            >
              <ListChecks className="size-4" />
              Select
            </Button>
          )}

          {/* Filter Input */}
          <div className="relative min-w-[180px] flex-1 sm:flex-none">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" aria-hidden />
            <Input
              type="text"
              placeholder="Filter chapters..."
              className="h-9 pl-9 text-xs bg-card border-border"
              value={filterQuery}
              onChange={(e) => {
                setFilterQuery(e.target.value);
                setPage(1);
              }}
            />
          </div>

          {/* Sort Selector */}
          <Select
            value={sortBy}
            onValueChange={(val) => {
              if (val) {
                onSortByChange(val);
                setPage(1);
              }
            }}
          >
            <SelectTrigger className="h-9 w-[130px] text-xs bg-card border-border">
              <SelectValue placeholder="Sort order">
                {sortOptions[sortBy] || sortBy}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="source" className="text-xs">Source Order</SelectItem>
              <SelectItem value="number" className="text-xs">Chapter Number</SelectItem>
              <SelectItem value="date" className="text-xs">Upload Date</SelectItem>
            </SelectContent>
          </Select>

          {/* Order Toggle */}
          <Button
            variant="outline"
            size="icon"
            className="h-9 w-9 bg-card border-border cursor-pointer"
            onClick={() => {
              onOrderToggle();
              setPage(1);
            }}
            title={order === 'desc' ? 'Sort descending' : 'Sort ascending'}
            aria-label={order === 'desc' ? 'Sort descending' : 'Sort ascending'}
          >
            {order === 'desc' ? <ArrowDown className="size-4" aria-hidden /> : <ArrowUp className="size-4" aria-hidden />}
          </Button>
        </div>
      </div>

      {/* Chapter List */}
      <div className="flex flex-col gap-2">
        {isLoading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full rounded-md" />
            ))}
          </div>
        ) : isError ? (
          <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-6 text-center text-sm text-destructive">
            Failed to load chapter list.
          </div>
        ) : filteredChapters.length === 0 ? (
          <div className="rounded-lg border border-border bg-card p-8 text-center text-sm text-muted-foreground">
            {filterQuery ? 'No matching chapters found.' : 'No chapters found.'}
          </div>
        ) : (
          paginatedChapters.map((c) => {
            const isPulling = pullingSet.has(c.id);
            const isDeletingFiles = deletingFilesSet.has(c.id);
            const isRemoving = removingSet.has(c.id);
            const isProcessing = isPulling || isDeletingFiles || isRemoving;
            const isSelected = selectedChapterIds.has(c.id);

            return (
              <ChapterRow
                key={c.id}
                chapter={c}
                mangaId={mangaId}
                providerId={providerId}
                remoteId={remoteId}
                isInLibrary={isInLibrary}
                isUnavailable={isUnavailable}
                hasNoContentProvider={hasNoContentProvider}
                isSelectionMode={isSelectionMode}
                isSelected={isSelected}
                isProcessing={isProcessing}
                isPulling={isPulling}
                isDeletingFiles={isDeletingFiles}
                isRemoving={isRemoving}
                openMenuId={openMenuId}
                onOpenMenuChange={setOpenMenuId}
                onToggleSelect={handleToggleRow}
                onPullChapter={onPullChapter}
                onDeleteFiles={onDeleteFiles}
                onRemoveChapter={onRemoveChapter}
              />
            );
          })
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <Pagination
            currentPage={safePage}
            totalPages={totalPages}
            onPageChange={setPage}
          />
        )}
      </div>

      {/* Batch Remove Confirmation Dialog */}
      <BatchRemoveDialog
        open={isConfirmRemoveOpen}
        onOpenChange={setIsConfirmRemoveOpen}
        selectedCount={selectedCount}
        onConfirm={handleConfirmRemove}
        isRemoving={isBatchRemoving}
      />
    </div>
  );
};
