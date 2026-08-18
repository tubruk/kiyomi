import React, { useState, useMemo, useEffect } from 'react';
import { Link } from '@tanstack/react-router';
import { ArrowDown, ArrowUp, BookOpen, Search, MoreVertical, RefreshCw, Trash2, AlertCircle, Check } from 'lucide-react';
import { Chapter, Source } from '../types/api';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Skeleton } from './ui/skeleton';
import { Tooltip, TooltipTrigger, TooltipContent } from './ui/tooltip';
import { cn } from '../lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';

const sortOptions: Record<string, string> = {
  source: 'Source Order',
  number: 'Chapter Number',
  date: 'Upload Date',
};

interface ChapterListProps {
  chapters: Chapter[];
  mangaId: string;
  sortBy: string;
  order: 'asc' | 'desc';
  onSortByChange: (newSort: string) => void;
  onOrderToggle: () => void;
  isLoading: boolean;
  isError: boolean;
  contentProviderName?: string;
  contentProviderTitle?: string;
  providerName?: string;
  isUnavailable?: boolean;
  hasNoContentProvider?: boolean;
  isInLibrary?: boolean;
  providerId?: string;
  remoteId?: string;
  onRefreshChapters?: () => void;
  isRefreshing?: boolean;
  onRemoveChapter?: (chapterId: string) => void;
  contentProviders?: Source[];
  selectedContentProviderId?: string;
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
  contentProviderTitle,
  providerName,
  isUnavailable = false,
  hasNoContentProvider = false,
  isInLibrary = false,
  providerId,
  remoteId,
  onRefreshChapters,
  isRefreshing = false,
  onRemoveChapter,
  contentProviders = [],
  selectedContentProviderId,
}) => {
  const [filterQuery, setFilterQuery] = useState('');
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 50;

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

  // Reset to page 1 when filtered list shrinks
  useEffect(() => {
    setPage(1);
  }, [filteredChapters.length]);

  const totalPages = Math.ceil(filteredChapters.length / PAGE_SIZE);
  const safePage = Math.min(Math.max(1, page), totalPages || 1);
  const paginatedChapters = filteredChapters.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  const memoizedProviderName = useMemo(() => {
    if (!selectedContentProviderId || !contentProviders) return undefined;
    const cp = contentProviders.find((p) => p.id === selectedContentProviderId);
    if (!cp) return selectedContentProviderId;
    return `${cp.name}${cp.language || cp.lang ? ` (${(cp.language || cp.lang)!.toUpperCase()})` : ''}`;
  }, [contentProviders, selectedContentProviderId]);

  const resolvedProviderName = contentProviderName || memoizedProviderName;

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
      {/* Header & Controls Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-border/50 pb-3">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="text-xl font-bold tracking-tight text-foreground">
            Chapters ({chapters.length})
          </h2>
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
        </div>

        <div className="flex flex-wrap items-center gap-3">
          {/* Content Provider Badge with Tooltip */}
          {resolvedProviderName && (
            <div className="flex items-center gap-1.5">
              <span className="text-xs font-medium text-muted-foreground hidden md:inline">Provider:</span>
              <Tooltip>
                <TooltipTrigger
                  className="inline-flex items-center justify-center h-9 px-3 rounded-md border border-border bg-card text-xs font-medium text-foreground hover:bg-accent hover:text-accent-foreground focus-visible:outline-hidden cursor-help transition-all shadow-xs active:scale-[0.98]"
                >
                  {resolvedProviderName}
                </TooltipTrigger>
                <TooltipContent
                  side="top"
                  align="center"
                  className="flex flex-col items-start gap-0.5 p-2.5"
                >
                  <span className="font-semibold opacity-70 text-[9px] uppercase tracking-wider text-background">
                    Title on Provider
                  </span>
                  <span className="font-medium text-xs break-words select-text text-background">
                    {contentProviderTitle || 'Loading original title...'}
                  </span>
                </TooltipContent>
              </Tooltip>
            </div>
          )}

          {/* Filter Input */}
          <div className="relative min-w-[180px] flex-1 sm:flex-none">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" aria-hidden />
            <Input
              type="text"
              placeholder="Filter chapters..."
              className="h-9 pl-9 text-xs bg-card border-border"
              value={filterQuery}
              onChange={(e) => setFilterQuery(e.target.value)}
            />
          </div>

          {/* Sort Selector */}
          <Select value={sortBy} onValueChange={(val) => val && onSortByChange(val)}>
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
            onClick={onOrderToggle}
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
            const dateVal = c.uploadDate || c.uploadedAt;
            const dateStr = dateVal
              ? new Date(dateVal).toLocaleDateString()
              : '';
            const chapterTitleDisplay = c.title || c.name || `Chapter ${c.number}`;
            const isRead = Boolean(c.meta?.is_read ?? (c as any).is_read);
            const lastReadPage = c.meta?.last_read_page ?? (c as any).last_read_page ?? 0;
            const pageCount = c.meta?.page_count ?? (c as any).page_count ?? 0;
            const isInProgress = !isRead && lastReadPage > 1;

            return (
              <div
                key={c.id}
                className={cn(
                  "group flex items-center justify-between rounded-lg border border-border bg-card px-4 py-3 text-sm transition-colors hover:border-primary/50 hover:bg-accent/50",
                  isRead && "opacity-60 bg-card/60"
                )}
              >
                {!isInLibrary && providerId && remoteId ? (
                  <Link
                    to="/providers/$providerId/manga/$remoteId/chapter/$chapterId"
                    params={{ providerId, remoteId, chapterId: c.id }}
                    search={isInProgress ? { page: lastReadPage } : undefined}
                    className="flex items-center gap-2.5 flex-1 min-w-0"
                  >
                    <span className={cn(
                      "inline-flex items-center justify-center font-mono font-semibold text-xs px-2 py-0.5 rounded shrink-0 border border-border/50 gap-1",
                      isRead ? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20" : "bg-secondary text-secondary-foreground"
                    )}>
                      {c.number}
                      {isRead && <Check className="size-3 text-emerald-500 stroke-[3]" aria-label="Read" />}
                    </span>

                    <span className={cn(
                      "font-medium truncate",
                      isRead ? "text-muted-foreground" : "text-foreground group-hover:text-primary"
                    )}>
                      {chapterTitleDisplay}
                    </span>

                    {isInProgress && (
                      <span className="inline-flex items-center rounded bg-primary/10 border border-primary/25 px-1.5 py-0.5 text-[10px] font-semibold text-primary shrink-0">
                        Page {lastReadPage}{pageCount > 0 ? `/${pageCount}` : ''}
                      </span>
                    )}
                  </Link>
                ) : (
                  <Link
                    to="/manga/$mangaId/chapter/$chapterId"
                    params={{ mangaId, chapterId: c.id }}
                    search={isInProgress ? { page: lastReadPage } : undefined}
                    className="flex items-center gap-2.5 flex-1 min-w-0"
                  >
                    <span className={cn(
                      "inline-flex items-center justify-center font-mono font-semibold text-xs px-2 py-0.5 rounded shrink-0 border border-border/50 gap-1",
                      isRead ? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20" : "bg-secondary text-secondary-foreground"
                    )}>
                      {c.number}
                      {isRead && <Check className="size-3 text-emerald-500 stroke-[3]" aria-label="Read" />}
                    </span>

                    <span className={cn(
                      "font-medium truncate",
                      isRead ? "text-muted-foreground" : "text-foreground group-hover:text-primary"
                    )}>
                      {chapterTitleDisplay}
                    </span>

                    {isInProgress && (
                      <span className="inline-flex items-center rounded bg-primary/10 border border-primary/25 px-1.5 py-0.5 text-[10px] font-semibold text-primary shrink-0">
                        Page {lastReadPage}{pageCount > 0 ? `/${pageCount}` : ''}
                      </span>
                    )}
                  </Link>
                )}

                <div className="flex items-center gap-3 shrink-0 ml-4">
                  {dateStr && (
                    <span className="text-xs text-muted-foreground font-mono">
                      {dateStr}
                    </span>
                  )}

                  {isInLibrary && onRemoveChapter && (
                    <DropdownMenu open={openMenuId === c.id} onOpenChange={(open) => setOpenMenuId(open ? c.id : null)}>
                      <DropdownMenuTrigger
                        className="inline-flex items-center justify-center rounded-md hover:bg-muted text-muted-foreground shrink-0 size-8 transition-colors cursor-pointer"
                        aria-label="Chapter Actions"
                      >
                        <MoreVertical className="size-4" />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-44 p-1">
                        <DropdownMenuItem
                          onClick={() => {
                            onRemoveChapter(c.id);
                            setOpenMenuId(null);
                          }}
                          className="w-full text-xs cursor-pointer justify-center font-medium h-8 text-destructive focus:text-destructive"
                        >
                          <Trash2 className="size-3.5 mr-1.5" />
                          Remove Chapter
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </div>
              </div>
            );
          })
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-2 pt-4 pb-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={safePage <= 1}
              className="h-8 px-3 text-xs cursor-pointer"
            >
              Prev
            </Button>
            <span className="text-xs text-muted-foreground font-mono">
              {safePage} / {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={safePage >= totalPages}
              className="h-8 px-3 text-xs cursor-pointer"
            >
              Next
            </Button>
          </div>
        )}
      </div>
    </div>
  );
};
