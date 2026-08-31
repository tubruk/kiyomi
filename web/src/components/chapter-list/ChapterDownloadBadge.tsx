import React from 'react';
import { HardDrive, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface ChapterDownloadBadgeProps {
  isPulling?: boolean;
  isDeletingFiles?: boolean;
  isRemoving?: boolean;
  isDownloaded?: boolean;
  isPartiallyDownloaded?: boolean;
  downloadedPages?: number;
  pageCount?: number;
  percent?: number | null;
  className?: string;
}

export const ChapterDownloadBadge: React.FC<ChapterDownloadBadgeProps> = ({
  isPulling,
  isDeletingFiles,
  isRemoving,
  isDownloaded,
  isPartiallyDownloaded,
  downloadedPages,
  pageCount,
  percent,
  className,
}) => {
  // 1. Removing state
  if (isRemoving) {
    return (
      <div
        className={cn('inline-flex items-center justify-center shrink-0', className)}
        title="Removing chapter from library..."
      >
        <Loader2 className="size-4 animate-spin text-destructive shrink-0" aria-label="Removing" />
      </div>
    );
  }

  // 2. Deleting files state
  if (isDeletingFiles) {
    return (
      <div
        className={cn('inline-flex items-center justify-center shrink-0', className)}
        title="Deleting chapter files..."
      >
        <Loader2 className="size-4 animate-spin text-destructive shrink-0" aria-label="Deleting files" />
      </div>
    );
  }

  // 3. Actively pulling or partial progress: download spinner immediately (percentage when available)
  if (isPulling || isPartiallyDownloaded) {
    const computedPercent =
      percent !== undefined && percent !== null
        ? percent
        : pageCount && pageCount > 0 && downloadedPages && downloadedPages > 0
          ? Math.round((downloadedPages / pageCount) * 100)
          : null;

    const hasPercentage = computedPercent !== null && computedPercent > 0;
    const tooltip = hasPercentage
      ? `Pulling: ${computedPercent}% (${downloadedPages || 0}/${pageCount || '?'} pages)`
      : 'Pulling chapter...';

    if (hasPercentage) {
      return (
        <div
          className={cn(
            'inline-flex items-center gap-1 text-[11px] font-semibold text-primary shrink-0',
            className
          )}
          title={tooltip}
        >
          <Loader2 className="size-3.5 animate-spin shrink-0 text-primary" aria-label="Pulling progress" />
          <span className="tabular-nums">{computedPercent}%</span>
        </div>
      );
    }

    return (
      <div
        className={cn('inline-flex items-center justify-center shrink-0', className)}
        title="Pulling chapter..."
      >
        <Loader2 className="size-4 animate-spin text-primary shrink-0" aria-label="Pulling chapter" />
      </div>
    );
  }

  // 4. Fully downloaded: simple disk icon
  if (isDownloaded) {
    const tooltip =
      downloadedPages && downloadedPages > 0
        ? `Downloaded (${downloadedPages}/${pageCount || downloadedPages} pages)`
        : 'Downloaded (Local storage)';

    return (
      <div
        className={cn(
          'inline-flex items-center justify-center shrink-0 text-muted-foreground/70 hover:text-foreground transition-colors',
          className
        )}
        title={tooltip}
      >
        <HardDrive
          className="size-4 text-emerald-500/80 dark:text-emerald-400/90 shrink-0"
          aria-label="Downloaded to disk"
        />
      </div>
    );
  }

  // 5. No indicators otherwise
  return null;
};

