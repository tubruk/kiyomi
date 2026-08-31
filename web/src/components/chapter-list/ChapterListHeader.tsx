import React from 'react';
import {
  ArrowDown,
  ArrowUp,
  HardDrive,
  ListChecks,
  RefreshCw,
  Search,
} from 'lucide-react';
import { Input } from '../ui/input';
import { Button } from '../ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { cn } from '../../lib/utils';
import { SORT_OPTIONS } from './hooks/useChapterFilters';

export interface ChapterListHeaderProps {
  chapterCount: number;
  downloadedCount?: number;
  contentProviderName?: string;
  isInLibrary?: boolean;
  onRefreshChapters?: () => void;
  isRefreshing?: boolean;
  isSelectionMode: boolean;
  onEnterSelectionMode: () => void;
  filterQuery: string;
  onFilterQueryChange: (query: string) => void;
  sortBy: string;
  order: 'asc' | 'desc';
  onSortByChange: (newSort: string) => void;
  onOrderToggle: () => void;
}

export const ChapterListHeader: React.FC<ChapterListHeaderProps> = ({
  chapterCount,
  downloadedCount = 0,
  contentProviderName,
  isInLibrary = false,
  onRefreshChapters,
  isRefreshing = false,
  isSelectionMode,
  onEnterSelectionMode,
  filterQuery,
  onFilterQueryChange,
  sortBy,
  order,
  onSortByChange,
  onOrderToggle,
}) => {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-border/50 pb-3">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-xl font-bold tracking-tight text-foreground">
          Chapters ({chapterCount})
        </h2>
        {isInLibrary && chapterCount > 0 && downloadedCount > 0 && (
          <span
            className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
            title={`${downloadedCount} of ${chapterCount} chapters downloaded`}
          >
            <HardDrive className="size-3 text-emerald-500" />
            {downloadedCount}/{chapterCount}
          </span>
        )}
        {contentProviderName && (
          <span className="text-xs text-muted-foreground">
            Provided by <span className="font-medium text-foreground">{contentProviderName}</span>
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
        {isInLibrary && chapterCount > 0 && !isSelectionMode && (
          <Button
            variant="outline"
            size="sm"
            onClick={onEnterSelectionMode}
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
            onChange={(e) => onFilterQueryChange(e.target.value)}
          />
        </div>

        {/* Sort Selector */}
        <Select
          value={sortBy}
          onValueChange={(val) => {
            if (val) {
              onSortByChange(val);
            }
          }}
        >
          <SelectTrigger className="h-9 w-[130px] text-xs bg-card border-border">
            <SelectValue placeholder="Sort order">
              {SORT_OPTIONS[sortBy] || sortBy}
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
  );
};
