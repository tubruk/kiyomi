import React from 'react';
import { Link } from '@tanstack/react-router';
import {
  Check,
  Loader2,
  MoreVertical,
  RefreshCw,
  Trash2,
} from 'lucide-react';
import { Chapter } from '../../types/api';
import { Checkbox } from '../ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import { cn } from '../../lib/utils';
import { ChapterDownloadBadge } from './ChapterDownloadBadge';

export interface ChapterRowProps {
  chapter: Chapter;
  mangaId: string;
  providerId?: string;
  remoteId?: string;
  isInLibrary?: boolean;
  isUnavailable?: boolean;
  hasNoContentProvider?: boolean;
  isSelectionMode: boolean;
  isSelected: boolean;
  isProcessing: boolean;
  isPulling: boolean;
  isDeletingFiles: boolean;
  isRemoving: boolean;
  openMenuId: string | null;
  onOpenMenuChange: (id: string | null) => void;
  onToggleSelect: (chapterId: string, event?: React.MouseEvent | React.KeyboardEvent) => void;
  onPullChapter?: (chapterId: string, providerId: string) => void;
  onDeleteFiles?: (chapterId: string, providerId: string) => void;
  onRemoveChapter?: (chapterId: string, providerId?: string) => void;
}

const ChapterRowComponent: React.FC<ChapterRowProps> = ({
  chapter,
  mangaId,
  providerId,
  remoteId,
  isInLibrary = false,
  isSelectionMode,
  isSelected,
  isProcessing,
  isPulling,
  isDeletingFiles,
  isRemoving,
  openMenuId,
  onOpenMenuChange,
  onToggleSelect,
  onPullChapter,
  onDeleteFiles,
  onRemoveChapter,
}) => {
  const dateVal = chapter.uploadDate || chapter.uploadedAt;
  const dateStr = dateVal ? new Date(dateVal).toLocaleDateString() : '';
  const chapterTitleDisplay = chapter.title || chapter.name || `Chapter ${chapter.number}`;
  const isRead = Boolean(
    chapter.meta?.is_read ??
    (chapter as any).is_read ??
    chapter.is_read ??
    chapter.isRead
  );
  const lastReadPage =
    chapter.meta?.last_read_page ??
    (chapter as any).last_read_page ??
    chapter.last_read_page ??
    chapter.lastReadPage ??
    0;
  const pageCount =
    chapter.page_count ??
    (chapter as any).pageCount ??
    chapter.meta?.page_count ??
    0;
  const isInProgress = !isRead && lastReadPage > 1;
  const isOrphaned = Boolean(chapter.meta?.orphaned ?? chapter.orphaned);
  const chapterProviderId =
    chapter.providerId ?? chapter.provider_id ?? chapter.meta?.provider_id ?? '';

  const isDownloaded = Boolean(
    chapter.is_downloaded ??
    (chapter as any).isDownloaded ??
    chapter.meta?.is_downloaded ??
    chapter.isDownloaded
  );
  const downloadedPages =
    chapter.downloaded_pages ??
    (chapter as any).downloadedPages ??
    chapter.meta?.downloaded_pages ??
    chapter.downloadedPages ??
    0;
  const isPartial = downloadedPages > 0 && !isDownloaded;
  const percent =
    pageCount > 0 ? Math.round((downloadedPages / pageCount) * 100) : null;

  const showChapterMenu =
    isInLibrary &&
    (Boolean(onRemoveChapter) ||
      Boolean(onDeleteFiles && chapterProviderId) ||
      Boolean(onPullChapter));

  return (
    <div
      role={isSelectionMode ? 'checkbox' : undefined}
      aria-checked={isSelectionMode ? isSelected : undefined}
      tabIndex={isSelectionMode ? 0 : undefined}
      onClick={
        isSelectionMode && !isProcessing
          ? (e) => onToggleSelect(chapter.id, e)
          : undefined
      }
      onKeyDown={
        isSelectionMode && !isProcessing
          ? (e) => {
              if (e.key === ' ' || e.key === 'Enter') {
                e.preventDefault();
                onToggleSelect(chapter.id, e);
              }
            }
          : undefined
      }
      className={cn(
        'group flex items-center justify-between rounded-lg border px-4 py-3 text-sm transition-colors',
        isSelectionMode && 'cursor-pointer select-none',
        isSelected
          ? 'border-primary/50 bg-primary/10 hover:bg-primary/15'
          : isRead
            ? 'border-border opacity-60 bg-card/60 hover:border-primary/50 hover:bg-accent/50 hover:opacity-100'
            : 'border-border bg-card hover:border-primary/50 hover:bg-accent/50',
        isOrphaned && !isSelected && 'border-amber-500/30 bg-amber-500/5'
      )}
    >
      {/* Checkbox in selection mode */}
      {isSelectionMode && (
        <div
          className={cn(
            'flex items-center mr-3 shrink-0 pointer-events-none',
            isProcessing && 'opacity-50'
          )}
        >
          <Checkbox
            checked={isSelected}
            tabIndex={-1}
            aria-label={`Select chapter ${chapter.number}`}
          />
        </div>
      )}

      {isSelectionMode ? (
        <div className="flex items-center gap-2.5 flex-1 min-w-0">
          <span
            className={cn(
              'inline-flex items-center justify-center font-mono font-semibold text-xs px-2 py-0.5 rounded shrink-0 border border-border/50 gap-1',
              isRead
                ? 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20'
                : 'bg-secondary text-secondary-foreground'
            )}
          >
            {chapter.number}
            {isRead && (
              <Check className="size-3 text-emerald-500 stroke-[3]" aria-label="Read" />
            )}
          </span>

          <span
            className={cn(
              'font-medium truncate',
              isRead ? 'text-muted-foreground' : 'text-foreground group-hover:text-primary'
            )}
          >
            {chapterTitleDisplay}
          </span>

          {isInProgress && (
            <span className="inline-flex items-center rounded bg-primary/10 border border-primary/25 px-1.5 py-0.5 text-[10px] font-semibold text-primary shrink-0">
              Page {lastReadPage}
              {pageCount > 0 ? `/${pageCount}` : ''}
            </span>
          )}
        </div>
      ) : !isInLibrary && providerId && remoteId ? (
        <Link
          to="/providers/$providerId/manga/$remoteId/chapter/$chapterId"
          params={{ providerId, remoteId, chapterId: chapter.id }}
          search={isInProgress ? { page: lastReadPage } : undefined}
          className="flex items-center gap-2.5 flex-1 min-w-0"
        >
          <span
            className={cn(
              'inline-flex items-center justify-center font-mono font-semibold text-xs px-2 py-0.5 rounded shrink-0 border border-border/50 gap-1',
              isRead
                ? 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20'
                : 'bg-secondary text-secondary-foreground'
            )}
          >
            {chapter.number}
            {isRead && (
              <Check className="size-3 text-emerald-500 stroke-[3]" aria-label="Read" />
            )}
          </span>

          <span
            className={cn(
              'font-medium truncate',
              isRead ? 'text-muted-foreground' : 'text-foreground group-hover:text-primary'
            )}
          >
            {chapterTitleDisplay}
          </span>

          {isInProgress && (
            <span className="inline-flex items-center rounded bg-primary/10 border border-primary/25 px-1.5 py-0.5 text-[10px] font-semibold text-primary shrink-0">
              Page {lastReadPage}
              {pageCount > 0 ? `/${pageCount}` : ''}
            </span>
          )}
        </Link>
      ) : (
        <Link
          to="/manga/$mangaId/chapter/$chapterId"
          params={{ mangaId, chapterId: chapter.id }}
          search={isInProgress ? { page: lastReadPage } : undefined}
          className="flex items-center gap-2.5 flex-1 min-w-0"
        >
          <span
            className={cn(
              'inline-flex items-center justify-center font-mono font-semibold text-xs px-2 py-0.5 rounded shrink-0 border border-border/50 gap-1',
              isRead
                ? 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20'
                : 'bg-secondary text-secondary-foreground'
            )}
          >
            {chapter.number}
            {isRead && (
              <Check className="size-3 text-emerald-500 stroke-[3]" aria-label="Read" />
            )}
          </span>

          <span
            className={cn(
              'font-medium truncate',
              isRead ? 'text-muted-foreground' : 'text-foreground group-hover:text-primary'
            )}
          >
            {chapterTitleDisplay}
          </span>

          {isInProgress && (
            <span className="inline-flex items-center rounded bg-primary/10 border border-primary/25 px-1.5 py-0.5 text-[10px] font-semibold text-primary shrink-0">
              Page {lastReadPage}
              {pageCount > 0 ? `/${pageCount}` : ''}
            </span>
          )}
        </Link>
      )}

      <div className="flex items-center gap-3 shrink-0 ml-4">
        <ChapterDownloadBadge
          isPulling={isPulling}
          isDeletingFiles={isDeletingFiles}
          isRemoving={isRemoving}
          isDownloaded={isDownloaded}
          isPartiallyDownloaded={isPartial}
          downloadedPages={downloadedPages}
          pageCount={pageCount}
          percent={percent}
        />

        {isOrphaned && (
          <span
            className="inline-flex items-center rounded bg-amber-500/10 border border-amber-500/30 px-1.5 py-0.5 text-[10px] font-semibold text-amber-600 dark:text-amber-400 shrink-0"
            title="Chapter is no longer returned by the content provider but is retained locally."
          >
            Orphaned
          </span>
        )}

        {dateStr && (
          <span className="text-xs text-muted-foreground font-mono">
            {dateStr}
          </span>
        )}

        {!isSelectionMode && showChapterMenu && (
          isRemoving ? (
            <div
              title="Removing chapter from library..."
              className="inline-flex items-center justify-center rounded-md text-destructive shrink-0 size-8"
            >
              <Loader2 className="size-4 animate-spin" aria-label="Removing chapter" />
            </div>
          ) : (
            <DropdownMenu
              open={openMenuId === chapter.id}
              onOpenChange={(open) => onOpenMenuChange(open ? chapter.id : null)}
            >
              <DropdownMenuTrigger
                disabled={isProcessing}
                className="inline-flex items-center justify-center rounded-md hover:bg-muted text-muted-foreground shrink-0 size-8 transition-colors cursor-pointer disabled:opacity-50"
                aria-label="Chapter Actions"
              >
                <MoreVertical className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52 p-1">
                {onPullChapter && chapterProviderId && (
                  <DropdownMenuItem
                    onClick={() => {
                      onPullChapter(chapter.id, chapterProviderId);
                      onOpenMenuChange(null);
                    }}
                    disabled={isProcessing}
                    className="w-full text-xs cursor-pointer font-medium h-8"
                  >
                    {isPulling ? (
                      <Loader2 className="size-3.5 mr-1.5 animate-spin" />
                    ) : (
                      <RefreshCw className="size-3.5 mr-1.5" />
                    )}
                    Pull chapter
                  </DropdownMenuItem>
                )}
                {onDeleteFiles && chapterProviderId && (
                  <DropdownMenuItem
                    onClick={() => {
                      onDeleteFiles(chapter.id, chapterProviderId);
                      onOpenMenuChange(null);
                    }}
                    disabled={isProcessing || (!isDownloaded && downloadedPages === 0)}
                    className="w-full text-xs cursor-pointer font-medium h-8"
                    title={
                      isDownloaded || downloadedPages > 0
                        ? `Remove on-disk page artifacts${downloadedPages > 0 ? ` (${downloadedPages} pages)` : ''} but keep the chapter entry`
                        : 'No files to delete'
                    }
                  >
                    {isDeletingFiles ? (
                      <Loader2 className="size-3.5 mr-1.5 animate-spin" />
                    ) : (
                      <Trash2 className="size-3.5 mr-1.5" />
                    )}
                    Delete files
                  </DropdownMenuItem>
                )}
                {onRemoveChapter && (
                  <DropdownMenuItem
                    onClick={() => {
                      onRemoveChapter(chapter.id, chapterProviderId || undefined);
                      onOpenMenuChange(null);
                    }}
                    disabled={isProcessing}
                    className="w-full text-xs cursor-pointer justify-center font-medium h-8 text-destructive focus:text-destructive"
                  >
                    {isRemoving ? (
                      <Loader2 className="size-3.5 mr-1.5 animate-spin" />
                    ) : (
                      <Trash2 className="size-3.5 mr-1.5" />
                    )}
                    Remove Chapter
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )
        )}
      </div>
    </div>
  );
};

export const ChapterRow = React.memo(ChapterRowComponent);
