import React, { useState } from 'react';
import { Search, Download, Loader2 } from 'lucide-react';
import { ProviderRef, Source, Manga, MangaMeta } from '../types/api';
import { api } from '../api/client';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../lib/queryKeys';
import { useToast } from '../context/ToastContext';
import { cn } from '../lib/utils';

interface ImportMetadataDialogProps {
  manga: Manga;
  sources: Source[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (manga: Manga) => void;
}

type Step = 'select-provider' | 'search' | 'results' | 'conflict' | 'confirm';

interface ConflictChoice {
  field: string;
  keep: 'existing' | 'imported';
}

export const ImportMetadataDialog: React.FC<ImportMetadataDialogProps> = ({
  manga,
  sources,
  open,
  onOpenChange,
  onSuccess,
}) => {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const [step, setStep] = useState<Step>('select-provider');
  const [selectedProvider, setSelectedProvider] = useState<Source | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<Manga[]>([]);
  const [selectedResult, setSelectedResult] = useState<Manga | null>(null);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [conflicts, setConflicts] = useState<ConflictChoice[]>([]);

  const resetState = () => {
    setStep('select-provider');
    setSelectedProvider(null);
    setSearchQuery('');
    setSearchResults([]);
    setSelectedResult(null);
    setSearchError(null);
    setConflicts([]);
  };

  const handleOpenChange = (open: boolean) => {
    if (!open) resetState();
    onOpenChange(open);
  };

  const searchMutation = useMutation({
    mutationFn: async () => {
      if (!selectedProvider) throw new Error('No provider selected');
      const results = await api.searchManga(selectedProvider.id, searchQuery);
      return results.mangas || [];
    },
    onSuccess: (results) => {
      setSearchResults(results);
      setStep('results');
      setSearchError(null);
    },
    onError: (err: any) => {
      setSearchError(err.message || 'Search failed');
      setSearchResults([]);
    },
  });

  const importMutation = useMutation({
    mutationFn: async () => {
      if (!selectedResult || !selectedProvider) throw new Error('No selection');

      // Build the fields to patch from conflict choices
      const fieldsToPatch: Partial<MangaMeta> = {};
      for (const conflict of conflicts) {
        if (conflict.keep === 'imported') {
          switch (conflict.field) {
            case 'title':
              fieldsToPatch.title = selectedResult.title;
              break;
            case 'description':
              fieldsToPatch.description = selectedResult.description;
              break;
            case 'authors':
              fieldsToPatch.authors = Array.from(new Set([selectedResult.author || '', ...(selectedResult.authors || [])])).filter(Boolean);
              break;
            case 'artists':
              fieldsToPatch.artists = Array.from(new Set([selectedResult.artist || '', ...(selectedResult.artists || [])])).filter(Boolean);
              break;
            case 'tags':
              fieldsToPatch.tags = selectedResult.tags || selectedResult.genres || [];
              break;
          }
        }
      }

      // Patch metadata if any fields chosen
      if (Object.keys(fieldsToPatch).length > 0) {
        await api.patchLibraryManga(manga.id, fieldsToPatch);
      }

      // Add provider binding
      const ref: ProviderRef = {
        provider_id: selectedProvider.id,
        provider_manga_id: selectedResult.id || selectedResult.contentRemoteId || selectedResult.url || '',
        manga_title: selectedResult.title,
      };
      return api.addProvider(manga.id, ref);
    },
    onSuccess: (updatedManga) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      showToast(`Metadata imported from "${selectedProvider?.name}"`, 'success');
      handleOpenChange(false);
      onSuccess?.(updatedManga);
    },
    onError: (err: any) => {
      showToast(`Failed to import: ${err.message}`, 'error');
    },
  });

  const detectConflicts = (remote: Manga): ConflictChoice[] => {
    const detected: ConflictChoice[] = [];

    const existing = manga.meta || {};

    if (remote.title && remote.title !== existing.title && remote.title !== manga.title) {
      detected.push({ field: 'title', keep: 'existing' });
    }
    if (remote.description && remote.description !== existing.description && remote.description !== manga.description) {
      detected.push({ field: 'description', keep: 'existing' });
    }

    const existingAuthors = existing.authors || [];
    const remoteAuthors = Array.from(new Set([remote.author, ...(remote.authors || [])])).filter(Boolean);
    if (remoteAuthors.length > 0 && JSON.stringify(existingAuthors) !== JSON.stringify(remoteAuthors)) {
      detected.push({ field: 'authors', keep: 'existing' });
    }

    const existingArtists = existing.artists || [];
    const remoteArtists = Array.from(new Set([remote.artist, ...(remote.artists || [])])).filter(Boolean);
    if (remoteArtists.length > 0 && JSON.stringify(existingArtists) !== JSON.stringify(remoteArtists)) {
      detected.push({ field: 'artists', keep: 'existing' });
    }

    const existingTags = existing.tags || manga.tags || manga.genres || [];
    const remoteTags = remote.tags || remote.genres || [];
    if (remoteTags.length > 0 && JSON.stringify(existingTags) !== JSON.stringify(remoteTags)) {
      detected.push({ field: 'tags', keep: 'existing' });
    }

    return detected;
  };

  const handleProviderSelect = (providerId: string | null) => {
    if (!providerId) return;
    const provider = sources.find((s) => s.id === providerId);
    setSelectedProvider(provider || null);
    setStep('search');
  };

  const handleSearch = () => {
    if (!searchQuery.trim()) return;
    searchMutation.mutate();
  };

  const handleResultSelect = (mangaResult: Manga) => {
    setSelectedResult(mangaResult);
    const detectedConflicts = detectConflicts(mangaResult);
    setConflicts(detectedConflicts);
    setStep(detectedConflicts.length > 0 ? 'conflict' : 'confirm');
  };

  const handleConflictChoice = (field: string, keep: 'existing' | 'imported') => {
    setConflicts((prev) =>
      prev.map((c) => (c.field === field ? { ...c, keep } : c))
    );
  };

  const handleConfirm = () => {
    importMutation.mutate();
  };

  const allConflictsResolved = conflicts.every((c) => c.keep !== undefined);

  const renderExistingValue = (field: string) => {
    switch (field) {
      case 'title': return manga.title || manga.meta?.title || '—';
      case 'description': return manga.description || manga.meta?.description || '—';
      case 'authors': return (manga.authors || manga.meta?.authors || []).join(', ') || '—';
      case 'artists': return (manga.artists || manga.meta?.artists || []).join(', ') || '—';
      case 'tags': return (manga.tags || manga.genres || manga.meta?.tags || []).join(', ') || '—';
      default: return '—';
    }
  };

  const renderImportedValue = (field: string) => {
    if (!selectedResult) return '—';
    switch (field) {
      case 'title': return selectedResult.title || '—';
      case 'description': return selectedResult.description || '—';
      case 'authors': return (Array.from(new Set([selectedResult.author, ...(selectedResult.authors || [])])).filter(Boolean).join(', ') || '—');
      case 'artists': return (Array.from(new Set([selectedResult.artist, ...(selectedResult.artists || [])])).filter(Boolean).join(', ') || '—');
      case 'tags': return (selectedResult.tags || selectedResult.genres || []).join(', ') || '—';
      default: return '—';
    }
  };

  const renderStep = () => {
    switch (step) {
      case 'select-provider':
        return (
          <>
            <DialogHeader>
              <DialogTitle>Import Metadata</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <p className="text-xs text-muted-foreground">
                Select a provider to import metadata from.
              </p>
              <Select onValueChange={handleProviderSelect}>
                <SelectTrigger className="text-xs">
                  <SelectValue placeholder="Choose a provider..." />
                </SelectTrigger>
                <SelectContent>
                  {sources.map((source) => (
                    <SelectItem key={source.id} value={source.id} className="text-xs">
                      {source.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => handleOpenChange(false)} className="cursor-pointer">
                Cancel
              </Button>
            </DialogFooter>
          </>
        );

      case 'search':
        return (
          <>
            <DialogHeader>
              <DialogTitle>Search {selectedProvider?.name}</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <p className="text-xs text-muted-foreground">
                Search for this manga on {selectedProvider?.name}.
              </p>
              <div className="flex gap-2">
                <Input
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                  placeholder="Search manga title..."
                  className="text-xs flex-1"
                  autoFocus
                />
                <Button
                  onClick={handleSearch}
                  disabled={!searchQuery.trim() || searchMutation.isPending}
                  className="gap-2 cursor-pointer"
                >
                  {searchMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
                  Search
                </Button>
              </div>
              {searchError && <p className="text-xs text-destructive">{searchError}</p>}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep('select-provider')} className="cursor-pointer">
                Back
              </Button>
            </DialogFooter>
          </>
        );

      case 'results':
        return (
          <>
            <DialogHeader>
              <DialogTitle>Search Results</DialogTitle>
            </DialogHeader>
            <div className="space-y-2 py-2 max-h-64 overflow-y-auto">
              {searchResults.length === 0 ? (
                <p className="text-xs text-muted-foreground text-center py-4">No results found.</p>
              ) : (
                searchResults.map((m) => (
                  <button
                    key={m.id || m.url || Math.random().toString()}
                    onClick={() => handleResultSelect(m)}
                    className="flex w-full items-center gap-3 rounded-lg border border-border p-2 text-left hover:bg-muted/50 transition-colors cursor-pointer"
                  >
                    {(m.coverUrl || m.cover) && (
                      <img src={m.coverUrl || m.cover} alt="" className="size-10 rounded object-cover" />
                    )}
                    <div className="flex-1 min-w-0">
                      <p className="text-xs font-semibold truncate">{m.title}</p>
                      {m.author && <p className="text-xs text-muted-foreground truncate">{m.author}</p>}
                    </div>
                  </button>
                ))
              )}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep('search')} className="cursor-pointer">
                Back
              </Button>
            </DialogFooter>
          </>
        );

      case 'conflict':
        return (
          <>
            <DialogHeader>
              <DialogTitle>Resolve Conflicts</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2 max-h-80 overflow-y-auto">
              <p className="text-xs text-muted-foreground">
                Choose which value to keep for each conflicting field.
              </p>
              {conflicts.map((conflict) => (
                <div key={conflict.field} className="space-y-1.5">
                  <p className="text-xs font-semibold capitalize">{conflict.field}</p>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => handleConflictChoice(conflict.field, 'existing')}
                      className={cn(
                        'flex flex-col gap-1 rounded-lg border p-2 text-left text-xs transition-colors cursor-pointer',
                        conflict.keep === 'existing'
                          ? 'border-primary bg-primary/10 text-foreground'
                          : 'border-border bg-muted/50 text-muted-foreground hover:bg-muted'
                      )}
                    >
                      <span className="text-[10px] font-bold uppercase text-muted-foreground">Keep Existing</span>
                      <span className="truncate">{renderExistingValue(conflict.field)}</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => handleConflictChoice(conflict.field, 'imported')}
                      className={cn(
                        'flex flex-col gap-1 rounded-lg border p-2 text-left text-xs transition-colors cursor-pointer',
                        conflict.keep === 'imported'
                          ? 'border-primary bg-primary/10 text-foreground'
                          : 'border-border bg-muted/50 text-muted-foreground hover:bg-muted'
                      )}
                    >
                      <span className="text-[10px] font-bold uppercase text-muted-foreground">Use Imported</span>
                      <span className="truncate">{renderImportedValue(conflict.field)}</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep('results')} className="cursor-pointer">
                Back
              </Button>
              <Button
                onClick={handleConfirm}
                disabled={!allConflictsResolved || importMutation.isPending}
                className="gap-2 cursor-pointer"
              >
                {importMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                Import & Add Provider
              </Button>
            </DialogFooter>
          </>
        );

      case 'confirm':
        return (
          <>
            <DialogHeader>
              <DialogTitle>Confirm Import</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2">
              {selectedResult && (
                <div className="flex items-center gap-3 rounded-lg border border-border p-3">
                  {(selectedResult.coverUrl || selectedResult.cover) && (
                    <img src={selectedResult.coverUrl || selectedResult.cover} alt="" className="size-14 rounded object-cover" />
                  )}
                  <div>
                    <p className="text-sm font-semibold">{selectedResult.title}</p>
                    <p className="text-xs text-muted-foreground">
                      {selectedProvider?.name} • {selectedResult.author || 'Unknown author'}
                    </p>
                  </div>
                </div>
              )}
              <p className="text-xs text-muted-foreground">
                This will add the provider binding without overwriting any existing metadata.
              </p>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep('results')} className="cursor-pointer">
                Back
              </Button>
              <Button
                onClick={handleConfirm}
                disabled={importMutation.isPending}
                className="gap-2 cursor-pointer"
              >
                {importMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                Import & Add Provider
              </Button>
            </DialogFooter>
          </>
        );
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        {renderStep()}
      </DialogContent>
    </Dialog>
  );
};
