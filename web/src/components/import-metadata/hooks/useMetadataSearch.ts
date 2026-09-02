import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api } from '../../../api/client';
import { Manga, Source } from '../../../types/api';
import { SearchMode } from '../types';

interface UseMetadataSearchOptions {
  manga: Manga;
  sources: Source[];
  open: boolean;
  initialProviderId?: string;
  initialRemoteId?: string;
  onSelectRemoteManga: (remote: Manga) => void;
  onErrorFallbackToSearch?: () => void;
}

export const useMetadataSearch = ({
  manga,
  sources,
  open,
  initialProviderId,
  initialRemoteId,
  onSelectRemoteManga,
  onErrorFallbackToSearch,
}: UseMetadataSearchOptions) => {
  const boundProviders = useMemo(
    () => manga.meta?.providers || [],
    [manga.meta?.providers]
  );

  const defaultProviderId = useMemo(() => {
    if (initialProviderId && sources.some((s) => s.id === initialProviderId)) {
      return initialProviderId;
    }
    const bound = sources.find((s) => boundProviders.some((p) => p.provider_id === s.id));
    return bound?.id || sources[0]?.id || '';
  }, [initialProviderId, sources, boundProviders]);

  const [searchMode, setSearchMode] = useState<SearchMode>('keyword');
  const [selectedProviderId, setSelectedProviderId] = useState<string>(defaultProviderId);
  const [searchQuery, setSearchQuery] = useState(manga.title || manga.meta?.title || '');
  const [directIdOrUrl, setDirectIdOrUrl] = useState(initialRemoteId || '');
  const [searchResults, setSearchResults] = useState<Manga[]>([]);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [isLoadingDetails, setIsLoadingDetails] = useState(false);

  const selectedProvider = sources.find((s) => s.id === selectedProviderId);

  const onSelectRemoteMangaRef = useRef(onSelectRemoteManga);
  onSelectRemoteMangaRef.current = onSelectRemoteManga;
  const onErrorFallbackToSearchRef = useRef(onErrorFallbackToSearch);
  onErrorFallbackToSearchRef.current = onErrorFallbackToSearch;

  const loadMangaDetails = useCallback(
    async (providerId: string, remoteId: string, fallbackManga?: Manga) => {
      setIsLoadingDetails(true);
      setSearchError(null);
      try {
        const details = await api.getProviderMangaDetails(providerId, remoteId);
        onSelectRemoteMangaRef.current(details);
      } catch (err: any) {
        if (fallbackManga) {
          onSelectRemoteMangaRef.current(fallbackManga);
        } else {
          setSearchError(err.message || 'Failed to fetch manga details from provider');
          onErrorFallbackToSearchRef.current?.();
        }
      } finally {
        setIsLoadingDetails(false);
      }
    },
    []
  );

  // Reset / initialize state when dialog opens or props change
  useEffect(() => {
    if (open) {
      const pid = initialProviderId && sources.some((s) => s.id === initialProviderId)
        ? initialProviderId
        : defaultProviderId;
      setSelectedProviderId(pid);
      setSearchQuery(manga.title || manga.meta?.title || '');
      setDirectIdOrUrl(initialRemoteId || '');
      setSearchResults([]);
      setSearchError(null);

      if (initialProviderId && initialRemoteId) {
        loadMangaDetails(initialProviderId, initialRemoteId);
      }
    }
  }, [open, initialProviderId, initialRemoteId, defaultProviderId, manga.title, manga.meta?.title, sources, loadMangaDetails]);

  // Search Mutation
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

  // Direct Lookup Mutation
  const directLookupMutation = useMutation({
    mutationFn: async (idOrUrl: string) => {
      if (!selectedProviderId) throw new Error('No provider selected');
      const input = idOrUrl.trim();
      if (!input) throw new Error('Please enter a Remote ID or URL');
      // If it looks like a URL, treat it as a search query — provider's Search()
      // handles URL detection and extraction internally.
      if (/^https?:\/\//i.test(input)) {
        const results = await api.searchManga(selectedProviderId, input);
        if (!results.mangas?.length) throw new Error('No results found for URL');
        return results.mangas[0];
      }
      return api.getProviderMangaDetails(selectedProviderId, input);
    },
    onSuccess: (mangaResult) => {
      setSearchError(null);
      onSelectRemoteManga(mangaResult);
    },
    onError: (err: any) => {
      setSearchError(err.message || 'Remote ID/URL lookup failed');
    },
  });

  const handleSearch = () => {
    const q = searchQuery.trim();
    if (!q) return;
    searchMutation.mutate(q);
  };

  const handleDirectLookup = () => {
    const val = directIdOrUrl.trim();
    if (!val) return;
    directLookupMutation.mutate(val);
  };

  const handleSelectSearchResult = (m: Manga) => {
    const remoteId = m.id || m.contentRemoteId || m.url || '';
    if (selectedProviderId && remoteId) {
      loadMangaDetails(selectedProviderId, remoteId, m);
    } else {
      onSelectRemoteManga(m);
    }
  };

  const resetSearchState = () => {
    setSearchResults([]);
    setSearchError(null);
  };

  return {
    boundProviders,
    selectedProvider,
    selectedProviderId,
    setSelectedProviderId,
    searchMode,
    setSearchMode,
    searchQuery,
    setSearchQuery,
    directIdOrUrl,
    setDirectIdOrUrl,
    searchResults,
    searchError,
    setSearchError,
    isLoadingDetails,
    searchMutation,
    directLookupMutation,
    loadMangaDetails,
    handleSearch,
    handleDirectLookup,
    handleSelectSearchResult,
    resetSearchState,
  };
};
