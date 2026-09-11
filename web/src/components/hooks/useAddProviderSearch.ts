import { useState, useEffect, useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ProviderRef, Source, Manga } from '../../types/api';
import { api } from '../../api/client';
import { queryKeys } from '../../lib/queryKeys';
import { useToast } from '../../context/ToastContext';

export type AddProviderStep = 'pick' | 'confirm';

export interface UseAddProviderSearchOptions {
  mangaId: string;
  sources: Source[];
  existingProviders?: ProviderRef[];
  mangaTitle?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function useAddProviderSearch({
  mangaId,
  sources,
  existingProviders = [],
  mangaTitle,
  open,
  onOpenChange,
}: UseAddProviderSearchOptions) {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const [step, setStep] = useState<AddProviderStep>('pick');

  const getInitialProviderId = useCallback(() => {
    const boundIds = new Set(existingProviders.map((p) => p.provider_id));
    const unbound = sources.find((s) => !boundIds.has(s.id));
    return unbound?.id ?? sources[0]?.id ?? '';
  }, [existingProviders, sources]);

  const [selectedProviderId, setSelectedProviderId] = useState<string | null>(getInitialProviderId());
  const [searchQuery, setSearchQuery] = useState(mangaTitle ?? '');
  const [searchResults, setSearchResults] = useState<Manga[]>([]);
  const [selectedResult, setSelectedResult] = useState<Manga | null>(null);
  const [searchError, setSearchError] = useState<string | null>(null);

  const resetState = useCallback(() => {
    setStep('pick');
    setSelectedProviderId(getInitialProviderId());
    setSearchQuery(mangaTitle ?? '');
    setSearchResults([]);
    setSelectedResult(null);
    setSearchError(null);
  }, [getInitialProviderId, mangaTitle]);

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) resetState();
      onOpenChange(nextOpen);
    },
    [onOpenChange, resetState]
  );

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

  const addBindingMutation = useMutation({
    mutationFn: async () => {
      if (!selectedResult || !selectedProviderId) throw new Error('No selection');
      const providerMangaId = selectedResult.id || selectedResult.contentRemoteId || '';
      return api.addBinding(mangaId, {
        provider_id: selectedProviderId,
        provider_manga_id: providerMangaId,
        manga_title: selectedResult.title,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(mangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(mangaId) });
      const name = sources.find((s) => s.id === selectedProviderId)?.name;
      showToast(`Provider "${name}" added`, 'success');
      handleOpenChange(false);
    },
    onError: (err: any) => {
      showToast(`Failed to add provider: ${err.message}`, 'error');
    },
  });

  // Auto-search when query changes (debounced) after a provider is selected.
  useEffect(() => {
    if (step !== 'pick' || !selectedProviderId || !open) return;
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
  }, [searchQuery, step, selectedProviderId, open]);

  const handleResultSelect = useCallback((manga: Manga) => {
    setSelectedResult(manga);
    setStep('confirm');
  }, []);

  const handleConfirm = useCallback(() => {
    addBindingMutation.mutate();
  }, [addBindingMutation]);

  const selectedProvider = sources.find((s) => s.id === selectedProviderId) || null;

  return {
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
    isSearching: searchMutation.isPending,
    isAdding: addBindingMutation.isPending,
    handleResultSelect,
    handleConfirm,
    handleOpenChange,
    resetState,
  };
}
