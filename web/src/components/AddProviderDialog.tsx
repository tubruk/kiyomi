import React from 'react';
import { Plus, Loader2, Link as LinkIcon } from 'lucide-react';
import { ProviderRef, Source, Manga } from '../types/api';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from './ui/dialog';
import { Button } from './ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { AliasCombobox } from './AliasCombobox';
import { CoverImage } from './CoverImage';
import { getProxyImageUrl } from '../lib/utils';
import { useAddProviderSearch } from './hooks/useAddProviderSearch';

interface AddProviderDialogProps {
  mangaId: string;
  sources: Source[];
  existingProviders?: ProviderRef[];
  mangaTitle?: string;
  mangaAliases?: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (manga: Manga) => void;
}

export const AddProviderDialog: React.FC<AddProviderDialogProps> = ({
  mangaId,
  sources,
  existingProviders = [],
  mangaTitle,
  mangaAliases = [],
  open,
  onOpenChange,
  onSuccess,
}) => {
  const {
    step,
    setStep,
    selectedProviderId,
    setSelectedProviderId,
    searchQuery,
    setSearchQuery,
    searchResults,
    selectedResult,
    searchError,
    selectedProvider,
    isSearching,
    isAdding,
    handleResultSelect,
    handleConfirm,
    handleOpenChange,
  } = useAddProviderSearch({
    mangaId,
    sources,
    existingProviders,
    mangaTitle,
    open,
    onOpenChange,
    onSuccess,
  });

  const renderStep = () => {
    switch (step) {
      case 'pick':
        return (
          <>
            <DialogHeader>
              <DialogTitle>Add Provider</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <div className="flex gap-2">
                <Select value={selectedProviderId} onValueChange={setSelectedProviderId}>
                  <SelectTrigger className="text-xs w-48 shrink-0">
                    <SelectValue placeholder="Choose a provider..." />
                  </SelectTrigger>
                  <SelectContent>
                    {sources.map((source) => {
                      const bound = existingProviders.filter((p) => p.provider_id === source.id);
                      const boundCount = bound.length;
                      return (
                        <SelectItem key={source.id} value={source.id} className="text-xs">
                          <div className="flex items-center justify-between gap-2">
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
                <div className="flex-1">
                  <AliasCombobox
                    value={searchQuery}
                    onChange={setSearchQuery}
                    defaultValue={mangaTitle ?? ''}
                    suggestions={mangaAliases}
                    placeholder={`Search ${selectedProvider?.name || 'manga'}...`}
                    autoFocus
                  />
                </div>
              </div>
              {searchError && <p className="text-xs text-destructive">{searchError}</p>}
              <div className="max-h-96 overflow-y-auto rounded-lg border border-border">
                {isSearching ? (
                  <div className="flex items-center justify-center gap-2 py-8 text-xs text-muted-foreground">
                    <Loader2 className="size-4 animate-spin" />
                    Searching...
                  </div>
                ) : searchResults.length === 0 ? (
                  <p className="text-xs text-muted-foreground text-center py-6">
                    {searchQuery.trim()
                      ? 'No results found. Try a different search term.'
                      : `Type to search on ${selectedProvider?.name}.`}
                  </p>
                ) : (
                  <ul className="divide-y divide-border">
                    {searchResults.map((manga) => (
                      <li key={manga.id || manga.url || Math.random().toString()}>
                        <button
                          onClick={() => handleResultSelect(manga)}
                          className="flex w-full items-start gap-3 p-2 text-left hover:bg-muted/50 transition-colors cursor-pointer"
                        >
                          <CoverImage
                            src={getProxyImageUrl(manga.coverUrl || manga.cover, manga.url)}
                            alt=""
                            shape="auto"
                            containerClassName="size-12 rounded shrink-0"
                            iconSize="size-5"
                          />
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-semibold break-words">{manga.title}</p>
                            {manga.author && (
                              <p className="text-xs text-muted-foreground break-words">{manga.author}</p>
                            )}
                          </div>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => handleOpenChange(false)}
                className="cursor-pointer"
              >
                Cancel
              </Button>
            </DialogFooter>
          </>
        );

      case 'confirm':
        return (
          <>
            <DialogHeader>
              <DialogTitle>Confirm Add Provider</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2">
              {selectedResult && (
                <div className="flex items-start gap-3 rounded-lg border border-border p-3">
                  <CoverImage
                    src={getProxyImageUrl(
                      selectedResult.coverUrl || selectedResult.cover,
                      selectedResult.url
                    )}
                    alt=""
                    shape="auto"
                    containerClassName="size-16 rounded shrink-0"
                    iconSize="size-7"
                  />
                  <div className="min-w-0">
                    <p className="text-sm font-semibold break-words">{selectedResult.title}</p>
                    <p className="text-xs text-muted-foreground break-words">
                      {selectedProvider?.name}
                      {selectedResult.author ? ` • ${selectedResult.author}` : ''}
                    </p>
                  </div>
                </div>
              )}
              <p className="text-xs text-muted-foreground">
                This will link this provider to the manga in your library. Chapters and metadata can be
                synced from this provider.
              </p>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep('pick')} className="cursor-pointer">
                Back
              </Button>
              <Button
                onClick={handleConfirm}
                disabled={isAdding}
                className="gap-2 cursor-pointer"
              >
                {isAdding ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Plus className="size-4" />
                )}
                Add Provider
              </Button>
            </DialogFooter>
          </>
        );
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-2xl">{renderStep()}</DialogContent>
    </Dialog>
  );
};
