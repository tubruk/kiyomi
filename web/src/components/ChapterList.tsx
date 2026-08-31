import React, { useMemo, useState, useCallback } from 'react';
import { AlertCircle, BookOpen } from 'lucide-react';
import { Chapter } from '../types/api';
import { Skeleton } from './ui/skeleton';
import { Pagination } from './Pagination';
import { ChapterRow } from './chapter-list/ChapterRow';
import { BatchActionBar } from './chapter-list/BatchActionBar';
import { BatchRemoveDialog } from './chapter-list/BatchRemoveDialog';
import { ChapterListHeader } from './chapter-list/ChapterListHeader';
import { useChapterFilters } from './chapter-list/hooks/useChapterFilters';
import { useChapterSelection } from './chapter-list/hooks/useChapterSelection';

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
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);

  const {
    filterQuery,
    setFilterQuery,
    setPage,
    totalPages,
    safePage,
    filteredChapters,
    paginatedChapters,
    downloadedCount,
  } = useChapterFilters({
    chapters,
    sortBy,
    order,
    pageSize: 50,
  });

  const {
    isSelectionMode,
    setIsSelectionMode,
    selectedChapterIds,
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
  } = useChapterSelection({
    chapters,
    filteredChapters,
    mangaId,
    onBatchRemove,
  });

  const pullingSet = useMemo(() => new Set(pullingChapterIds), [pullingChapterIds]);
  const deletingFilesSet = useMemo(() => new Set(deletingFilesChapterIds), [deletingFilesChapterIds]);
  const removingSet = useMemo(() => new Set(removingChapterIds), [removingChapterIds]);

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
    const displayProvider = providerName || contentProviderName || 'Provider';
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
      <ChapterListHeader
        chapterCount={chapters.length}
        downloadedCount={downloadedCount}
        contentProviderName={contentProviderName}
        isInLibrary={isInLibrary}
        onRefreshChapters={onRefreshChapters}
        isRefreshing={isRefreshing}
        isSelectionMode={isSelectionMode}
        onEnterSelectionMode={() => {
          setIsSelectionMode(true);
        }}
        filterQuery={filterQuery}
        onFilterQueryChange={setFilterQuery}
        sortBy={sortBy}
        order={order}
        onSortByChange={onSortByChange}
        onOrderToggle={onOrderToggle}
      />

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
