import React, { useState, useEffect } from 'react';
import { Plus, Loader2, Check } from 'lucide-react';
import { ProviderRef, Source, Manga } from '../types/api';
import { api } from '../api/client';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from './ui/dialog';
import { Button } from './ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { AliasCombobox } from './AliasCombobox';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../lib/queryKeys';
import { useToast } from '../context/ToastContext';

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

type Step = 'pick' | 'confirm';

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
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const [step, setStep] = useState<Step>('pick');
  const initialProviderId = (() => {
    const boundIds = new Set(existingProviders.map((p) => p.provider_id));
    const unbound = sources.find((s) => !boundIds.has(s.id));
    return unbound?.id ?? sources[0]?.id ?? '';
  })();
  const [selectedProviderId, setSelectedProviderId] = useState<string | null>(initialProviderId);
  const [searchQuery, setSearchQuery] = useState(mangaTitle ?? '');
  const [searchResults, setSearchResults] = useState<Manga[]>([]);
  const [selectedResult, setSelectedResult] = useState<Manga | null>(null);
  const [searchError, setSearchError] = useState<string | null>(null);

  const resetState = () => {
    setStep('pick');
    setSelectedProviderId(initialProviderId);
    setSearchQuery(mangaTitle ?? '');
    setSearchResults([]);
    setSelectedResult(null);
    setSearchError(null);
  };

  const handleOpenChange = (open: boolean) => {
    if (!open) resetState();
    onOpenChange(open);
  };

  const searchMutation = useMutation({
    mutationFn: async (query: string) => {
      if (!selectedProviderId) throw new Error('No provider selected');
      const results = await api.searchManga(selectedProviderId, query);
      return results.mangas || [];
    },
    onSuccess: (results) => {
      setSearchResults(results);
      setSearchError(null);
    },
    onError: (err: any) => {
      setSearchError(err.message || 'Search failed');
      setSearchResults([]);
    },
  });

  const addProviderMutation = useMutation({
    mutationFn: async () => {
      if (!selectedResult || !selectedProviderId) throw new Error('No selection');
      const providerMangaId = selectedResult.id || selectedResult.contentRemoteId || selectedResult.url || '';
      return api.addProvider(
        mangaId,
        {
          provider_id: selectedProviderId,
          provider_manga_id: providerMangaId,
          manga_title: selectedResult.title,
        },
        false
      );
    },
    onSuccess: (manga) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      const name = sources.find((s) => s.id === selectedProviderId)?.name;
      showToast(`Provider "${name}" added`, 'success');
      handleOpenChange(false);
      onSuccess?.(manga);
    },
    onError: (err: any) => {
      showToast(`Failed to add provider: ${err.message}`, 'error');
    },
  });

  // Auto-search when query changes (debounced) after a provider is selected.
  useEffect(() => {
    if (step !== 'pick' || !selectedProviderId) return;
    const q = searchQuery.trim();
    if (!q) {
      setSearchResults([]);
      setSearchError(null);
      return;
    }
    const handle = setTimeout(() => {
      searchMutation.mutate(q);
    }, 300);
    return () => clearTimeout(handle);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery, step, selectedProviderId]);

  const handleResultSelect = (manga: Manga) => {
    setSelectedResult(manga);
    setStep('confirm');
  };

  const handleConfirm = () => {
    addProviderMutation.mutate();
  };

  const selectedProvider = sources.find((s) => s.id === selectedProviderId) || null;

  const renderStep = () => {
    switch (step) {
      case 'pick':
        return (
          <>
            <DialogHeader>
              <DialogTitle>
                Add Provider
              </DialogTitle>
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
                              <span className="inline-flex items-center gap-1 text-[10px] text-muted-foreground">
                                <Check className="size-3" />
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
              {searchError && (
                <p className="text-xs text-destructive">{searchError}</p>
              )}
              <div className="max-h-96 overflow-y-auto rounded-lg border border-border">
                {searchMutation.isPending ? (
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
                          {manga.coverUrl || manga.cover ? (
                            <img
                              src={manga.coverUrl || manga.cover}
                              alt=""
                              className="size-12 rounded object-cover shrink-0"
                            />
                          ) : (
                            <div className="size-12 rounded bg-muted shrink-0" />
                          )}
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
              <Button variant="outline" onClick={() => handleOpenChange(false)} className="cursor-pointer">
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
                  {selectedResult.coverUrl || selectedResult.cover ? (
                    <img
                      src={selectedResult.coverUrl || selectedResult.cover}
                      alt=""
                      className="size-16 rounded object-cover shrink-0"
                    />
                  ) : (
                    <div className="size-16 rounded bg-muted shrink-0" />
                  )}
                  <div className="min-w-0">
                    <p className="text-sm font-semibold break-words">{selectedResult.title}</p>
                    <p className="text-xs text-muted-foreground break-words">
                      {selectedProvider?.name}{selectedResult.author ? ` • ${selectedResult.author}` : ''}
                    </p>
                  </div>
                </div>
              )}
              <p className="text-xs text-muted-foreground">
                This will link this provider to the manga in your library. Chapters and metadata can be synced from this provider.
              </p>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep('pick')} className="cursor-pointer">
                Back
              </Button>
              <Button
                onClick={handleConfirm}
                disabled={addProviderMutation.isPending}
                className="gap-2 cursor-pointer"
              >
                {addProviderMutation.isPending ? (
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
      <DialogContent className="sm:max-w-2xl">
        {renderStep()}
      </DialogContent>
    </Dialog>
  );
};
