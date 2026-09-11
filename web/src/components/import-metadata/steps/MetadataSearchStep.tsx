import React from 'react';
import { Search, Loader2, Link as LinkIcon } from 'lucide-react';
import { Manga, Source, ProviderRef } from '../../../types/api';
import { SearchMode } from '../types';
import {
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '../../ui/dialog';
import { Button } from '../../ui/button';
import { Input } from '../../ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../../ui/select';
import { AliasCombobox } from '../../AliasCombobox';
import { CoverImage } from '../../CoverImage';
import { cn, getProxyImageUrl } from '../../../lib/utils';

interface MetadataSearchStepProps {
  manga: Manga;
  sources: Source[];
  boundProviders: ProviderRef[];
  selectedProviderId: string;
  onSelectProviderId: (id: string) => void;
  selectedProvider?: Source;
  searchMode: SearchMode;
  onSelectSearchMode: (mode: SearchMode) => void;
  searchQuery: string;
  onSearchQueryChange: (q: string) => void;
  directIdOrUrl: string;
  onDirectIdOrUrlChange: (v: string) => void;
  isSearching: boolean;
  isLookingUp: boolean;
  isLoadingDetails: boolean;
  searchResults: Manga[];
  searchError: string | null;
  onSearch: () => void;
  onDirectLookup: () => void;
  onSelectSearchResult: (m: Manga) => void;
  onCancel: () => void;
}

export const MetadataSearchStep: React.FC<MetadataSearchStepProps> = ({
  manga,
  sources,
  boundProviders,
  selectedProviderId,
  onSelectProviderId,
  selectedProvider,
  searchMode,
  onSelectSearchMode,
  searchQuery,
  onSearchQueryChange,
  directIdOrUrl,
  onDirectIdOrUrlChange,
  isSearching,
  isLookingUp,
  isLoadingDetails,
  searchResults,
  searchError,
  onSearch,
  onDirectLookup,
  onSelectSearchResult,
  onCancel,
}) => {
  return (
    <div className="space-y-4">
      <DialogHeader className="pb-1">
        <DialogTitle className="text-base font-semibold">Import Metadata</DialogTitle>
      </DialogHeader>

      {/* Provider selector & Mode Toggle */}
      <div className="space-y-3.5">
        <div className="flex flex-col sm:flex-row gap-2 items-stretch sm:items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground shrink-0 font-medium">Provider:</span>
            <Select value={selectedProviderId} onValueChange={(val) => val && onSelectProviderId(val)}>
              <SelectTrigger className="text-xs h-8 w-52 shrink-0">
                <SelectValue placeholder="Choose provider..." />
              </SelectTrigger>
              <SelectContent>
                {sources.map((source) => {
                  const bound = boundProviders.filter((p) => p.provider_id === source.id);
                  const boundCount = bound.length;
                  return (
                    <SelectItem key={source.id} value={source.id} className="text-xs">
                      <div className="flex items-center justify-between gap-3 w-full">
                        <span>{source.name}</span>
                        {boundCount > 0 && (
                          <span className="inline-flex items-center gap-1 rounded bg-secondary px-1.5 py-0.2 text-[10px] text-muted-foreground">
                            <LinkIcon className="size-2.5 text-muted-foreground" />
                            {boundCount === 1 ? 'Bound' : `Bound ×${boundCount}`}
                          </span>
                        )}
                      </div>
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </div>

          {/* Mode Switcher */}
          <div className="flex items-center rounded-lg border border-border bg-muted/40 p-0.5 text-xs">
            <button
              type="button"
              onClick={() => onSelectSearchMode('keyword')}
              className={cn(
                'px-3 py-1 rounded-md text-xs transition-colors cursor-pointer',
                searchMode === 'keyword'
                  ? 'bg-background text-foreground font-medium shadow-xs'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              Keyword Search
            </button>
            <button
              type="button"
              onClick={() => onSelectSearchMode('direct')}
              className={cn(
                'px-3 py-1 rounded-md text-xs transition-colors cursor-pointer',
                searchMode === 'direct'
                  ? 'bg-background text-foreground font-medium shadow-xs'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              Remote ID / URL
            </button>
          </div>
        </div>

        {/* Search Input Bars */}
        {searchMode === 'keyword' ? (
          <div className="flex gap-2">
            <div className="flex-1">
              <AliasCombobox
                value={searchQuery}
                onChange={onSearchQueryChange}
                defaultValue={manga.title || manga.metadata?.title || ''}
                suggestions={manga.aliases || manga.metadata?.aliases || []}
                placeholder={`Search title or alias on ${selectedProvider?.name || 'provider'}...`}
                autoFocus
              />
            </div>
            <Button
              type="button"
              onClick={onSearch}
              disabled={!searchQuery.trim() || isSearching}
              className="gap-1.5 h-9 text-xs px-4 cursor-pointer shrink-0"
            >
              {isSearching ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Search className="size-3.5" />
              )}
              Search
            </Button>
          </div>
        ) : (
          <div className="flex gap-2">
            <div className="flex-1">
              <Input
                value={directIdOrUrl}
                onChange={(e) => onDirectIdOrUrlChange(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && onDirectLookup()}
                placeholder="Enter Remote ID or full URL..."
                className="text-xs h-9"
                autoFocus
              />
            </div>
            <Button
              type="button"
              onClick={onDirectLookup}
              disabled={!directIdOrUrl.trim() || isLookingUp}
              className="gap-1.5 h-9 text-xs px-4 cursor-pointer shrink-0"
            >
              {isLookingUp ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Search className="size-3.5" />
              )}
              Lookup
            </Button>
          </div>
        )}

        {searchError && <p className="text-xs text-destructive">{searchError}</p>}
      </div>

      {/* Results List */}
      {searchMode === 'keyword' && (
        <div className="max-h-80 overflow-y-auto rounded-xl border border-border divide-y divide-border">
          {isLoadingDetails || isSearching ? (
            <div className="flex items-center justify-center gap-2 py-12 text-xs text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              <span>Searching {selectedProvider?.name}...</span>
            </div>
          ) : searchResults.length === 0 ? (
            <p className="text-xs text-muted-foreground text-center py-10">
              {searchQuery.trim()
                ? 'No results found. Try typing a different search query.'
                : `Type a title or select an alias above to search ${selectedProvider?.name}.`}
            </p>
          ) : (
            searchResults.map((m) => (
              <button
                key={m.id || m.url || Math.random().toString()}
                type="button"
                onClick={() => onSelectSearchResult(m)}
                className="flex w-full items-start gap-3.5 p-3 text-left hover:bg-muted/50 transition-colors cursor-pointer"
              >
                <CoverImage
                  src={getProxyImageUrl(m.coverUrl || m.cover, m.url)}
                  alt=""
                  shape="auto"
                  containerClassName="w-12 h-16 rounded shrink-0 shadow-xs"
                  iconSize="size-5"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-semibold break-words line-clamp-1">{m.title}</p>
                  <p className="text-[11px] text-muted-foreground truncate mt-0.5">
                    {m.author || (m.authors && m.authors.length > 0 ? m.authors.join(', ') : '') || 'Unknown author'}
                  </p>
                  {(() => {
                    const tags = (m.tags && m.tags.length > 0 ? m.tags : m.genres || []).filter(
                      (t) => typeof t === 'string' && t.trim().length > 0
                    );
                    if (tags.length === 0) return null;
                    return (
                      <div className="flex items-center gap-1 mt-1.5 overflow-hidden">
                        {tags.slice(0, 3).map((tag) => (
                          <span
                            key={tag}
                            className="inline-block text-[9px] rounded bg-muted px-1.5 py-0.5 text-muted-foreground truncate"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    );
                  })()}
                </div>
              </button>
            ))
          )}
        </div>
      )}

      <DialogFooter className="-mx-6 -mb-6 mt-4 px-6 py-4 border-t bg-muted/50 flex flex-row justify-end">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          className="text-xs cursor-pointer"
        >
          Cancel
        </Button>
      </DialogFooter>
    </div>
  );
};
