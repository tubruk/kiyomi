import React, { useState, useEffect, useMemo } from 'react';
import {
  Search,
  Download,
  Loader2,
  Link as LinkIcon,
  X,
  Sparkles,
  RotateCcw,
  Plus,
  ArrowLeft,
  BookOpen,
  Image as ImageIcon,
  Building2,
  Calendar,
  ShieldAlert,
  Globe,
  Compass,
  Layers,
} from 'lucide-react';
import { ProviderRef, Source, Manga, ExternalLink } from '../types/api';
import { api } from '../api/client';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Badge } from './ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';
import { AliasCombobox } from './AliasCombobox';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../lib/queryKeys';
import { useToast } from '../context/ToastContext';
import { cn, getProxyImageUrl } from '../lib/utils';

export interface ImportMetadataDialogProps {
  manga: Manga;
  sources: Source[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialProviderId?: string;
  initialRemoteId?: string;
  onSuccess?: (manga: Manga) => void;
}

type DialogStep = 'search' | 'compare';
type SearchMode = 'keyword' | 'direct';
type Choice = 'current' | 'incoming';
type MultiChoice = 'current' | 'incoming' | 'merged';

const normalizeSet = (items?: string[]): Set<string> => {
  return new Set((items || []).map((s) => s.trim().toLowerCase()).filter(Boolean));
};

export const areArraySetsEqual = (a?: string[], b?: string[]): boolean => {
  const setA = normalizeSet(a);
  const setB = normalizeSet(b);
  if (setA.size !== setB.size) return false;
  for (const item of setA) {
    if (!setB.has(item)) return false;
  }
  return true;
};

export const mergeStringArrays = (...arrays: (string[] | undefined)[]): string[] => {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const arr of arrays) {
    if (!arr) continue;
    for (const item of arr) {
      const trimmed = (item ?? '').trim();
      if (!trimmed) continue;
      const lower = trimmed.toLowerCase();
      if (!seen.has(lower)) {
        seen.add(lower);
        result.push(trimmed);
      }
    }
  }
  return result;
};

export const areExternalLinkSetsEqual = (a?: ExternalLink[], b?: ExternalLink[]): boolean => {
  const linksA = (a || []).filter((l) => Boolean(l && l.url?.trim()));
  const linksB = (b || []).filter((l) => Boolean(l && l.url?.trim()));
  if (linksA.length !== linksB.length) return false;
  const setA = new Set(linksA.map((l) => l.url.trim().toLowerCase()));
  const setB = new Set(linksB.map((l) => l.url.trim().toLowerCase()));
  if (setA.size !== setB.size) return false;
  for (const url of setA) {
    if (!setB.has(url)) return false;
  }
  return true;
};

export const mergeExternalLinkArrays = (...arrays: (ExternalLink[] | undefined)[]): ExternalLink[] => {
  const seen = new Set<string>();
  const result: ExternalLink[] = [];
  for (const arr of arrays) {
    if (!arr) continue;
    for (const item of arr) {
      if (!item || !item.url?.trim()) continue;
      const key = item.url.trim().toLowerCase();
      if (!seen.has(key)) {
        seen.add(key);
        result.push({
          provider: item.provider || 'custom',
          label: item.label || item.provider || 'Link',
          url: item.url.trim(),
        });
      }
    }
  }
  return result;
};

export const ImportMetadataDialog: React.FC<ImportMetadataDialogProps> = ({
  manga,
  sources,
  open,
  onOpenChange,
  initialProviderId,
  initialRemoteId,
  onSuccess,
}) => {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

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

  const [step, setStep] = useState<DialogStep>('search');
  const [searchMode, setSearchMode] = useState<SearchMode>('keyword');
  const [selectedProviderId, setSelectedProviderId] = useState<string>(defaultProviderId);
  const [searchQuery, setSearchQuery] = useState(manga.title || manga.meta?.title || '');
  const [directIdOrUrl, setDirectIdOrUrl] = useState(initialRemoteId || '');
  const [searchResults, setSearchResults] = useState<Manga[]>([]);
  const [selectedRemoteManga, setSelectedRemoteManga] = useState<Manga | null>(null);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [isLoadingDetails, setIsLoadingDetails] = useState(false);

  // Comparison State
  const [diffOnly, setDiffOnly] = useState(true);
  const [selectedTitle, setSelectedTitle] = useState<Choice>('incoming');
  const [selectedCover, setSelectedCover] = useState<Choice>('incoming');
  const [selectedDescription, setSelectedDescription] = useState<Choice>('incoming');
  const [selectedAuthorsMode, setSelectedAuthorsMode] = useState<MultiChoice>('incoming');
  const [selectedArtistsMode, setSelectedArtistsMode] = useState<MultiChoice>('incoming');
  const [selectedExternalLinksMode, setSelectedExternalLinksMode] = useState<MultiChoice>('incoming');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [selectedAliases, setSelectedAliases] = useState<string[]>([]);
  const [aliasInput, setAliasInput] = useState('');
  const [tagInput, setTagInput] = useState('');
  const [selectedPublisher, setSelectedPublisher] = useState<Choice>('incoming');
  const [selectedReleaseYear, setSelectedReleaseYear] = useState<Choice>('incoming');
  const [selectedContentRating, setSelectedContentRating] = useState<Choice>('incoming');
  const [selectedCountry, setSelectedCountry] = useState<Choice>('incoming');
  const [selectedReadingMode, setSelectedReadingMode] = useState<Choice>('incoming');

  // Reset when dialog opens/closes
  useEffect(() => {
    if (open) {
      const pid = initialProviderId && sources.some((s) => s.id === initialProviderId)
        ? initialProviderId
        : defaultProviderId;
      setSelectedProviderId(pid);
      setSearchQuery(manga.title || manga.meta?.title || '');
      setDirectIdOrUrl(initialRemoteId || '');
      setSearchResults([]);
      setSelectedRemoteManga(null);
      setSearchError(null);
      setDiffOnly(true);
      setAliasInput('');
      setTagInput('');

      if (initialProviderId && initialRemoteId) {
        // Auto-fetch and go to comparison
        loadMangaDetails(initialProviderId, initialRemoteId);
      } else {
        setStep('search');
      }
    }
  }, [open, initialProviderId, initialRemoteId, defaultProviderId, manga, sources]);

  const loadMangaDetails = async (providerId: string, remoteId: string, fallbackManga?: Manga) => {
    setIsLoadingDetails(true);
    setSearchError(null);
    try {
      const details = await api.getProviderMangaDetails(providerId, remoteId);
      initComparisonState(details);
    } catch (err: any) {
      if (fallbackManga) {
        initComparisonState(fallbackManga);
      } else {
        setSearchError(err.message || 'Failed to fetch manga details from provider');
        setStep('search');
      }
    } finally {
      setIsLoadingDetails(false);
    }
  };

  const initComparisonState = (remote: Manga) => {
    setSelectedRemoteManga(remote);

    // Current metadata values
    const currentTitle = manga.title || manga.meta?.title || '';
    const currentCover = manga.coverUrl || manga.cover || manga.coverAssetUrl || manga.meta?.cover_url || '';
    const currentDesc = manga.description || manga.meta?.description || '';
    const currentAuthors = mergeStringArrays(manga.authors, manga.author ? [manga.author] : (manga.meta?.authors || []));
    const currentArtists = mergeStringArrays(manga.artists, manga.artist ? [manga.artist] : (manga.meta?.artists || []));
    const currentTagsList = mergeStringArrays(manga.tags, manga.genres || manga.meta?.tags);
    const currentAliasesList = manga.aliases || manga.meta?.aliases || [];
    const currentExternalLinks: ExternalLink[] = (manga.externalLinks && manga.externalLinks.length > 0)
      ? manga.externalLinks
      : ((manga.meta as any)?.external_links || []);

    // Incoming metadata values
    const incomingTitle = remote.title?.trim() || '';
    const incomingCover = remote.coverUrl || remote.cover || remote.coverAssetUrl || '';
    const incomingDesc = remote.description?.trim() || '';
    const incomingAuthors = mergeStringArrays(remote.authors, remote.author ? [remote.author] : []);
    const incomingArtists = mergeStringArrays(remote.artists, remote.artist ? [remote.artist] : []);
    const incomingTagsList = mergeStringArrays(remote.tags, remote.genres);
    const incomingAliasesList = remote.aliases || [];
    const incomingExternalLinks: ExternalLink[] = [];
    const incomingUrl = remote.url?.trim() || (remote.id?.startsWith('http') ? remote.id.trim() : '');
    if (incomingUrl) {
      incomingExternalLinks.push({
        provider: selectedProviderId,
        label: selectedProvider?.name || selectedProviderId,
        url: incomingUrl,
      });
    }
    const remoteExtLinks: ExternalLink[] = remote.externalLinks || (remote.meta as any)?.external_links || [];
    for (const link of remoteExtLinks) {
      if (link && link.url?.trim() && !incomingExternalLinks.some((l) => l.url.trim().toLowerCase() === link.url.trim().toLowerCase())) {
        incomingExternalLinks.push(link);
      }
    }

    // Defaults
    setSelectedTitle(incomingTitle && incomingTitle !== currentTitle ? 'incoming' : 'current');
    setSelectedCover(incomingCover && incomingCover !== currentCover ? 'incoming' : 'current');
    setSelectedDescription(incomingDesc && incomingDesc !== currentDesc ? 'incoming' : 'current');
    setSelectedAuthorsMode(incomingAuthors.length > 0 ? (areArraySetsEqual(currentAuthors, incomingAuthors) ? 'current' : 'incoming') : 'current');
    setSelectedArtistsMode(incomingArtists.length > 0 ? (areArraySetsEqual(currentArtists, incomingArtists) ? 'current' : 'incoming') : 'current');
    setSelectedExternalLinksMode(
      incomingExternalLinks.length > 0
        ? areExternalLinkSetsEqual(currentExternalLinks, incomingExternalLinks)
          ? 'current'
          : 'merged'
        : 'current'
    );
    setSelectedTags(mergeStringArrays(currentTagsList, incomingTagsList));
    setSelectedAliases(mergeStringArrays(currentAliasesList, incomingAliasesList));

    setSelectedPublisher(remote.publisher?.trim() ? 'incoming' : 'current');
    setSelectedReleaseYear((remote.releaseYear || remote.meta?.release_year) ? 'incoming' : 'current');
    setSelectedContentRating((remote.contentRating || remote.meta?.content_rating) ? 'incoming' : 'current');
    setSelectedCountry((remote.country || remote.meta?.country) ? 'incoming' : 'current');
    setSelectedReadingMode((remote.readingMode || remote.reading_mode || remote.readingDirection) ? 'incoming' : 'current');

    setStep('compare');
  };

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      setStep('search');
      setSelectedRemoteManga(null);
      setSearchResults([]);
      setSearchError(null);
      setAliasInput('');
      setTagInput('');
    }
    onOpenChange(isOpen);
  };

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
      if (!idOrUrl.trim()) throw new Error('Please enter a Remote ID or URL');
      return api.getProviderMangaDetails(selectedProviderId, idOrUrl.trim());
    },
    onSuccess: (mangaResult) => {
      setSearchError(null);
      initComparisonState(mangaResult);
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
      initComparisonState(m);
    }
  };

  // Quick Action: Accept Incoming
  const handleAcceptIncoming = () => {
    if (!selectedRemoteManga) return;
    setSelectedTitle(incomingValues.title ? 'incoming' : 'current');
    setSelectedCover(incomingValues.coverUrl ? 'incoming' : 'current');
    setSelectedDescription(incomingValues.description ? 'incoming' : 'current');
    setSelectedAuthorsMode(incomingValues.authors.length > 0 ? 'incoming' : 'current');
    setSelectedArtistsMode(incomingValues.artists.length > 0 ? 'incoming' : 'current');
    setSelectedTags(incomingValues.tags.length > 0 ? [...incomingValues.tags] : [...currentValues.tags]);
    setSelectedAliases(mergeStringArrays(currentValues.aliases, incomingValues.aliases));
    setSelectedPublisher(incomingValues.publisher ? 'incoming' : 'current');
    setSelectedReleaseYear(incomingValues.releaseYear ? 'incoming' : 'current');
    setSelectedContentRating(incomingValues.contentRating ? 'incoming' : 'current');
    setSelectedCountry(incomingValues.country ? 'incoming' : 'current');
    setSelectedReadingMode(incomingValues.readingMode ? 'incoming' : 'current');
    setSelectedExternalLinksMode(incomingValues.externalLinks.length > 0 ? 'incoming' : 'current');
  };

  // Quick Action: Keep Current
  const handleKeepCurrent = () => {
    setSelectedTitle('current');
    setSelectedCover('current');
    setSelectedDescription('current');
    setSelectedAuthorsMode('current');
    setSelectedArtistsMode('current');
    setSelectedTags([...currentValues.tags]);
    setSelectedAliases([...currentValues.aliases]);
    setSelectedPublisher('current');
    setSelectedReleaseYear('current');
    setSelectedContentRating('current');
    setSelectedCountry('current');
    setSelectedReadingMode('current');
    setSelectedExternalLinksMode('current');
  };

  // Import & Save Mutation
  const importMutation = useMutation({
    mutationFn: async () => {
      if (!selectedRemoteManga || !selectedProviderId) throw new Error('No selection');

      const fieldsToPatch: Record<string, any> = {};

      if (selectedTitle === 'incoming' && incomingValues.title && incomingValues.title !== currentValues.title) {
        fieldsToPatch.title = incomingValues.title;
      }
      if (selectedCover === 'incoming' && incomingValues.coverUrl) {
        fieldsToPatch.cover_url = incomingValues.coverUrl;
      }
      if (selectedDescription === 'incoming' && incomingValues.description) {
        fieldsToPatch.description = incomingValues.description;
      }
      if (selectedAuthorsMode === 'incoming') {
        fieldsToPatch.authors = incomingValues.authors;
      } else if (selectedAuthorsMode === 'merged') {
        fieldsToPatch.authors = mergeStringArrays(currentValues.authors, incomingValues.authors);
      }
      if (selectedArtistsMode === 'incoming') {
        fieldsToPatch.artists = incomingValues.artists;
      } else if (selectedArtistsMode === 'merged') {
        fieldsToPatch.artists = mergeStringArrays(currentValues.artists, incomingValues.artists);
      }

      fieldsToPatch.tags = selectedTags;
      fieldsToPatch.aliases = selectedAliases;

      if (selectedPublisher === 'incoming' && incomingValues.publisher) {
        fieldsToPatch.publisher = incomingValues.publisher;
      }
      if (selectedReleaseYear === 'incoming' && incomingValues.releaseYear) {
        fieldsToPatch.release_year = incomingValues.releaseYear;
      }
      if (selectedContentRating === 'incoming' && incomingValues.contentRating) {
        fieldsToPatch.content_rating = incomingValues.contentRating;
      }
      if (selectedCountry === 'incoming' && incomingValues.country) {
        fieldsToPatch.country = incomingValues.country;
      }
      if (selectedReadingMode === 'incoming' && incomingValues.readingMode) {
        fieldsToPatch.reading_mode = incomingValues.readingMode;
      }
      if (selectedExternalLinksMode === 'incoming') {
        fieldsToPatch.external_links = incomingValues.externalLinks;
        fieldsToPatch.externalLinks = incomingValues.externalLinks;
      } else if (selectedExternalLinksMode === 'merged') {
        const merged = mergeExternalLinkArrays(currentValues.externalLinks, incomingValues.externalLinks);
        fieldsToPatch.external_links = merged;
        fieldsToPatch.externalLinks = merged;
      }

      // 1. Patch metadata
      if (Object.keys(fieldsToPatch).length > 0) {
        await api.patchLibraryManga(manga.id, fieldsToPatch);
      }

      // 2. Add provider binding
      const providerMangaId =
        selectedRemoteManga.id ||
        selectedRemoteManga.contentRemoteId ||
        selectedRemoteManga.url ||
        '';
      const ref: ProviderRef = {
        provider_id: selectedProviderId,
        provider_manga_id: providerMangaId,
        manga_title: selectedRemoteManga.title || incomingValues.title,
      };
      return api.addProvider(manga.id, ref);
    },
    onSuccess: (updatedManga) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.providers(manga.id) });
      const providerName = sources.find((s) => s.id === selectedProviderId)?.name || 'Provider';
      showToast(`Metadata imported from "${providerName}"`, 'success');
      handleOpenChange(false);
      onSuccess?.(updatedManga);
    },
    onError: (err: any) => {
      showToast(`Failed to import: ${err.message}`, 'error');
    },
  });

  const selectedProvider = sources.find((s) => s.id === selectedProviderId);

  // Normalized values
  const currentValues = useMemo(() => {
    return {
      title: manga.title || manga.meta?.title || '',
      coverUrl: manga.coverUrl || manga.cover || manga.coverAssetUrl || manga.meta?.cover_url || '',
      description: manga.description || manga.meta?.description || '',
      authors: mergeStringArrays(manga.authors, manga.author ? [manga.author] : (manga.meta?.authors || [])),
      artists: mergeStringArrays(manga.artists, manga.artist ? [manga.artist] : (manga.meta?.artists || [])),
      tags: mergeStringArrays(manga.tags, manga.genres || manga.meta?.tags),
      aliases: manga.aliases || manga.meta?.aliases || [],
      publisher: manga.publisher || manga.meta?.publisher || '',
      releaseYear: manga.releaseYear || manga.meta?.release_year || 0,
      contentRating: manga.contentRating || manga.meta?.content_rating || '',
      country: manga.country || manga.meta?.country || '',
      readingMode: manga.readingMode || manga.reading_mode || manga.readingDirection || manga.content?.reading_mode || manga.meta?.content?.reading_mode || '',
      externalLinks: ((manga.externalLinks && manga.externalLinks.length > 0)
        ? manga.externalLinks
        : (manga.meta?.external_links || [])) as ExternalLink[],
    };
  }, [manga]);

  const incomingValues = useMemo(() => {
    if (!selectedRemoteManga) {
      return {
        title: '',
        coverUrl: '',
        description: '',
        authors: [] as string[],
        artists: [] as string[],
        tags: [] as string[],
        aliases: [] as string[],
        publisher: '',
        releaseYear: 0,
        contentRating: '',
        country: '',
        readingMode: '',
        externalLinks: [] as ExternalLink[],
      };
    }
    const incomingLinks: ExternalLink[] = [];
    const incomingUrl = selectedRemoteManga.url?.trim() || (selectedRemoteManga.id?.startsWith('http') ? selectedRemoteManga.id.trim() : '');
    if (incomingUrl) {
      incomingLinks.push({
        provider: selectedProviderId,
        label: selectedProvider?.name || selectedProviderId,
        url: incomingUrl,
      });
    }
    const remoteExtLinks: ExternalLink[] = selectedRemoteManga.externalLinks || (selectedRemoteManga.meta as any)?.external_links || [];
    for (const link of remoteExtLinks) {
      if (link && link.url?.trim() && !incomingLinks.some((l) => l.url.trim().toLowerCase() === link.url.trim().toLowerCase())) {
        incomingLinks.push(link);
      }
    }

    return {
      title: selectedRemoteManga.title?.trim() || '',
      coverUrl: selectedRemoteManga.coverUrl || selectedRemoteManga.cover || selectedRemoteManga.coverAssetUrl || '',
      description: selectedRemoteManga.description?.trim() || '',
      authors: mergeStringArrays(selectedRemoteManga.authors, selectedRemoteManga.author ? [selectedRemoteManga.author] : []),
      artists: mergeStringArrays(selectedRemoteManga.artists, selectedRemoteManga.artist ? [selectedRemoteManga.artist] : []),
      tags: mergeStringArrays(selectedRemoteManga.tags, selectedRemoteManga.genres),
      aliases: selectedRemoteManga.aliases || [],
      publisher: selectedRemoteManga.publisher?.trim() || '',
      releaseYear: selectedRemoteManga.releaseYear || selectedRemoteManga.meta?.release_year || 0,
      contentRating: selectedRemoteManga.contentRating || selectedRemoteManga.meta?.content_rating || '',
      country: selectedRemoteManga.country || selectedRemoteManga.meta?.country || '',
      readingMode: selectedRemoteManga.readingMode || selectedRemoteManga.reading_mode || selectedRemoteManga.readingDirection || '',
      externalLinks: incomingLinks,
    };
  }, [selectedRemoteManga, selectedProviderId, selectedProvider]);

  // Diff checks
  const diffs = useMemo(() => {
    return {
      cover: Boolean(incomingValues.coverUrl && incomingValues.coverUrl !== currentValues.coverUrl),
      title: Boolean(incomingValues.title && incomingValues.title.toLowerCase() !== currentValues.title.toLowerCase()),
      description: Boolean(incomingValues.description && incomingValues.description.trim() !== currentValues.description.trim()),
      authors: Boolean(incomingValues.authors.length > 0 && !areArraySetsEqual(currentValues.authors, incomingValues.authors)),
      artists: Boolean(incomingValues.artists.length > 0 && !areArraySetsEqual(currentValues.artists, incomingValues.artists)),
      tags: Boolean(incomingValues.tags.length > 0 && !areArraySetsEqual(currentValues.tags, incomingValues.tags)),
      aliases: Boolean(incomingValues.aliases.length > 0 && !areArraySetsEqual(currentValues.aliases, incomingValues.aliases)),
      publisher: Boolean(incomingValues.publisher && incomingValues.publisher.toLowerCase() !== currentValues.publisher.toLowerCase()),
      releaseYear: Boolean(incomingValues.releaseYear > 0 && incomingValues.releaseYear !== currentValues.releaseYear),
      contentRating: Boolean(incomingValues.contentRating && incomingValues.contentRating.toLowerCase() !== currentValues.contentRating.toLowerCase()),
      country: Boolean(incomingValues.country && incomingValues.country.toLowerCase() !== currentValues.country.toLowerCase()),
      readingMode: Boolean(incomingValues.readingMode && incomingValues.readingMode.toLowerCase() !== currentValues.readingMode.toLowerCase()),
      externalLinks: Boolean(
        incomingValues.externalLinks.length > 0 &&
        !areExternalLinkSetsEqual(currentValues.externalLinks, incomingValues.externalLinks)
      ),
    };
  }, [currentValues, incomingValues]);

  const mergedExternalLinks = useMemo(
    () => mergeExternalLinkArrays(currentValues.externalLinks, incomingValues.externalLinks),
    [currentValues.externalLinks, incomingValues.externalLinks]
  );

  // Tag Helpers
  const allAvailableTags = useMemo(
    () => mergeStringArrays(currentValues.tags, incomingValues.tags),
    [currentValues.tags, incomingValues.tags]
  );
  const unselectedTags = useMemo(
    () => allAvailableTags.filter((t) => !selectedTags.some((st) => st.toLowerCase() === t.toLowerCase())),
    [allAvailableTags, selectedTags]
  );

  const removeTag = (tag: string) => {
    setSelectedTags((prev) => prev.filter((t) => t.toLowerCase() !== tag.toLowerCase()));
  };

  const addTag = (tag: string) => {
    const trimmed = tag.trim();
    if (!trimmed) return;
    if (!selectedTags.some((t) => t.toLowerCase() === trimmed.toLowerCase())) {
      setSelectedTags((prev) => [...prev, trimmed]);
    }
  };

  const handleTagInputChange = (val: string) => {
    if (val.includes(',')) {
      const parts = val.split(',');
      for (const part of parts.slice(0, -1)) {
        addTag(part);
      }
      setTagInput(parts[parts.length - 1]);
    } else {
      setTagInput(val);
    }
  };

  const handleTagInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      if (tagInput.trim()) {
        addTag(tagInput);
        setTagInput('');
      }
    }
  };

  // Alias Helpers
  const allAvailableAliases = useMemo(
    () =>
      mergeStringArrays(
        currentValues.aliases,
        incomingValues.aliases,
        incomingValues.title && incomingValues.title.toLowerCase() !== currentValues.title.toLowerCase()
          ? [incomingValues.title]
          : [],
        currentValues.title && incomingValues.title.toLowerCase() !== currentValues.title.toLowerCase()
          ? [currentValues.title]
          : []
      ),
    [currentValues.aliases, incomingValues.aliases, currentValues.title, incomingValues.title]
  );
  const unselectedAliases = useMemo(
    () => allAvailableAliases.filter((a) => !selectedAliases.some((sa) => sa.toLowerCase() === a.toLowerCase())),
    [allAvailableAliases, selectedAliases]
  );

  const removeAlias = (alias: string) => {
    setSelectedAliases((prev) => prev.filter((a) => a.toLowerCase() !== alias.toLowerCase()));
  };

  const addAlias = (alias: string) => {
    const trimmed = alias.trim();
    if (!trimmed) return;
    if (!selectedAliases.some((a) => a.toLowerCase() === trimmed.toLowerCase())) {
      setSelectedAliases((prev) => [...prev, trimmed]);
    }
  };

  const handleAliasInputChange = (val: string) => {
    if (val.includes(',')) {
      const parts = val.split(',');
      for (const part of parts.slice(0, -1)) {
        addAlias(part);
      }
      setAliasInput(parts[parts.length - 1]);
    } else {
      setAliasInput(val);
    }
  };

  const handleAliasInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      if (aliasInput.trim()) {
        addAlias(aliasInput);
        setAliasInput('');
      }
    }
  };

  // Render Step 1: Search / Direct Lookup
  const renderSearchStep = () => {
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
              <Select value={selectedProviderId} onValueChange={(val) => val && setSelectedProviderId(val)}>
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
                onClick={() => setSearchMode('keyword')}
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
                onClick={() => setSearchMode('direct')}
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
                  onChange={setSearchQuery}
                  defaultValue={manga.title || manga.meta?.title || ''}
                  suggestions={manga.aliases || manga.meta?.aliases || []}
                  placeholder={`Search title or alias on ${selectedProvider?.name || 'provider'}...`}
                  autoFocus
                />
              </div>
              <Button
                type="button"
                onClick={handleSearch}
                disabled={!searchQuery.trim() || searchMutation.isPending}
                className="gap-1.5 h-9 text-xs px-4 cursor-pointer shrink-0"
              >
                {searchMutation.isPending ? (
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
                  onChange={(e) => setDirectIdOrUrl(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleDirectLookup()}
                  placeholder="Enter Remote ID or full URL..."
                  className="text-xs h-9"
                  autoFocus
                />
              </div>
              <Button
                type="button"
                onClick={handleDirectLookup}
                disabled={!directIdOrUrl.trim() || directLookupMutation.isPending}
                className="gap-1.5 h-9 text-xs px-4 cursor-pointer shrink-0"
              >
                {directLookupMutation.isPending ? (
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
            {isLoadingDetails || searchMutation.isPending ? (
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
                  onClick={() => handleSelectSearchResult(m)}
                  className="flex w-full items-start gap-3.5 p-3 text-left hover:bg-muted/50 transition-colors cursor-pointer"
                >
                  {m.coverUrl || m.cover ? (
                    <img
                      src={getProxyImageUrl(m.coverUrl || m.cover, m.url)}
                      alt=""
                      className="w-12 h-16 rounded object-cover shrink-0 bg-muted shadow-xs"
                      onError={(e) => {
                        e.currentTarget.style.display = 'none';
                      }}
                    />
                  ) : (
                    <div className="w-12 h-16 rounded bg-muted flex items-center justify-center shrink-0">
                      <BookOpen className="size-5 text-muted-foreground" />
                    </div>
                  )}
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
            onClick={() => handleOpenChange(false)}
            className="text-xs cursor-pointer"
          >
            Cancel
          </Button>
        </DialogFooter>
      </div>
    );
  };

  // Render Step 2: Side-by-Side Comparison Screen
  const renderCompareStep = () => {
    const isTitleDiff = diffs.title;
    const isCoverDiff = diffs.cover;
    const isDescDiff = diffs.description;
    const isAuthorsDiff = diffs.authors;
    const isArtistsDiff = diffs.artists;
    const isTagsDiff = diffs.tags;
    const isAliasesDiff = diffs.aliases;
    const isExternalLinksDiff = diffs.externalLinks;

    const isPublisherDiff = diffs.publisher;
    const isYearDiff = diffs.releaseYear;
    const isRatingDiff = diffs.contentRating;
    const isCountryDiff = diffs.country;
    const isReadingModeDiff = diffs.readingMode;

    const hasAnyAttributesDiff =
      isPublisherDiff || isYearDiff || isRatingDiff || isCountryDiff || isReadingModeDiff;

    const hasAnyDiff =
      isCoverDiff ||
      isTitleDiff ||
      isDescDiff ||
      isAuthorsDiff ||
      isArtistsDiff ||
      isTagsDiff ||
      isAliasesDiff ||
      isExternalLinksDiff ||
      hasAnyAttributesDiff;

    const showCoverSection = incomingValues.coverUrl && (!diffOnly || isCoverDiff);
    const showTitleSection = incomingValues.title && (!diffOnly || isTitleDiff);
    const showDescSection = incomingValues.description && (!diffOnly || isDescDiff);
    const showAuthorsSection = incomingValues.authors.length > 0 && (!diffOnly || isAuthorsDiff);
    const showArtistsSection = incomingValues.artists.length > 0 && (!diffOnly || isArtistsDiff);
    const showTagsSection = incomingValues.tags.length > 0 && (!diffOnly || isTagsDiff);
    const showAliasesSection =
      currentValues.aliases.length > 0 ||
      incomingValues.aliases.length > 0 ||
      selectedAliases.length > 0 ||
      !diffOnly;
    const showExternalLinksSection =
      (incomingValues.externalLinks.length > 0 || currentValues.externalLinks.length > 0) &&
      (!diffOnly || isExternalLinksDiff);
    const showAttributesSection = !diffOnly || hasAnyAttributesDiff;

    const showNothingInDiff = diffOnly && !hasAnyDiff;

    const targetTitleAlias = (selectedTitle === 'current' ? incomingValues.title : currentValues.title)?.trim() || '';
    const isTargetTitleInAliases = Boolean(
      targetTitleAlias &&
      selectedAliases.some((a) => a.toLowerCase() === targetTitleAlias.toLowerCase())
    );
    const showAddTitleToAliases = Boolean(
      isTitleDiff &&
      targetTitleAlias &&
      !isTargetTitleInAliases
    );

    return (
      <div className="flex flex-col h-full min-h-0 flex-1">
        {/* Top Header & Quick Actions */}
        <div className="px-6 py-4 border-b border-border bg-card/60 shrink-0 space-y-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div className="flex items-center gap-2 min-w-0">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setStep('search')}
                className="size-8 p-0 cursor-pointer shrink-0"
                title="Back to search"
              >
                <ArrowLeft className="size-4" />
              </Button>
              <div className="min-w-0">
                <DialogTitle className="text-base font-semibold truncate flex items-center gap-2">
                  <span>Compare & Import Metadata</span>
                  <Badge variant="outline" className="text-[10px] font-normal shrink-0">
                    {selectedProvider?.name}
                  </Badge>
                </DialogTitle>
              </div>
            </div>

            {/* Quick Action Toolbar */}
            <div className="flex flex-wrap items-center gap-2 shrink-0">
              <div className="flex items-center rounded-lg border border-border bg-muted/40 p-0.5 text-xs">
                <button
                  type="button"
                  onClick={() => setDiffOnly(true)}
                  className={cn(
                    'px-2.5 py-1 rounded-md text-xs transition-colors cursor-pointer',
                    diffOnly
                      ? 'bg-background text-foreground font-medium shadow-xs'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  Diff only
                </button>
                <button
                  type="button"
                  onClick={() => setDiffOnly(false)}
                  className={cn(
                    'px-2.5 py-1 rounded-md text-xs transition-colors cursor-pointer',
                    !diffOnly
                      ? 'bg-background text-foreground font-medium shadow-xs'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  Show all
                </button>
              </div>

              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleKeepCurrent}
                className="h-8 text-xs gap-1.5 px-3 cursor-pointer"
                title="Keep all current metadata"
              >
                <RotateCcw className="size-3.5" />
                Keep Current
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAcceptIncoming}
                className="h-8 text-xs gap-1.5 px-3 cursor-pointer text-primary border-primary/40 hover:bg-primary/10"
                title="Accept all incoming metadata"
              >
                <Sparkles className="size-3.5" />
                Accept Incoming
              </Button>
            </div>
          </div>
        </div>

        {/* Scrollable Comparative Form */}
        <div className="px-6 py-5 space-y-6 flex-1 min-h-0 overflow-y-auto">
          {showNothingInDiff ? (
            <div className="rounded-xl border border-dashed border-border p-8 text-center space-y-2.5">
              <Sparkles className="size-6 text-primary mx-auto" />
              <p className="text-xs font-medium">All incoming metadata matches your current manga.</p>
              <p className="text-[11px] text-muted-foreground">
                No differences detected. Switch to &quot;Show all&quot; to review identical fields, or proceed to bind the provider.
              </p>
            </div>
          ) : (
            <>
              {/* 1. Cover Image Comparison */}
              {showCoverSection && (
                <div className="space-y-2.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold flex items-center gap-1.5">
                      <ImageIcon className="size-4 text-muted-foreground" />
                      Cover Image
                    </span>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <button
                      type="button"
                      onClick={() => setSelectedCover('current')}
                      className={cn(
                        'flex items-center gap-4 rounded-xl border p-3.5 text-left transition-all cursor-pointer',
                        selectedCover === 'current'
                          ? 'border-primary bg-primary/5 ring-1 ring-primary'
                          : 'border-border bg-card hover:bg-muted/40'
                      )}
                    >
                      {currentValues.coverUrl ? (
                        <img
                          src={manga.coverAssetUrl || getProxyImageUrl(currentValues.coverUrl, manga.url)}
                          alt=""
                          className="w-16 h-24 rounded-md object-cover shrink-0 bg-muted shadow-xs"
                          onError={(e) => {
                            e.currentTarget.style.display = 'none';
                          }}
                        />
                      ) : (
                        <div className="w-16 h-24 rounded-md bg-muted flex items-center justify-center shrink-0">
                          <ImageIcon className="size-6 text-muted-foreground" />
                        </div>
                      )}
                      <div className="min-w-0 space-y-1">
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                          Keep Current
                        </span>
                        <p className="text-sm font-medium truncate">Current Cover</p>
                      </div>
                    </button>

                    <button
                      type="button"
                      onClick={() => setSelectedCover('incoming')}
                      className={cn(
                        'flex items-center gap-4 rounded-xl border p-3.5 text-left transition-all cursor-pointer',
                        selectedCover === 'incoming'
                          ? 'border-primary bg-primary/5 ring-1 ring-primary'
                          : 'border-border bg-card hover:bg-muted/40'
                      )}
                    >
                      {incomingValues.coverUrl ? (
                        <img
                          src={getProxyImageUrl(incomingValues.coverUrl, selectedRemoteManga?.url)}
                          alt=""
                          className="w-16 h-24 rounded-md object-cover shrink-0 bg-muted shadow-xs"
                          onError={(e) => {
                            e.currentTarget.style.display = 'none';
                          }}
                        />
                      ) : (
                        <div className="w-16 h-24 rounded-md bg-muted flex items-center justify-center shrink-0">
                          <ImageIcon className="size-6 text-muted-foreground" />
                        </div>
                      )}
                      <div className="min-w-0 space-y-1">
                        <span className="text-[10px] font-bold uppercase tracking-wider text-primary">
                          Use Incoming
                        </span>
                        <p className="text-sm font-medium truncate">Incoming Cover</p>
                      </div>
                    </button>
                  </div>
                </div>
              )}

              {/* 2. Title Comparison */}
              {showTitleSection && (
                <div className="space-y-2.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold flex items-center gap-1.5">
                      <BookOpen className="size-4 text-muted-foreground" />
                      Title
                    </span>
                    {showAddTitleToAliases && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => addAlias(targetTitleAlias)}
                        className="h-6 text-[11px] gap-1 px-2 text-muted-foreground hover:text-foreground cursor-pointer"
                        title="Add alternate title to aliases"
                      >
                        <Plus className="size-3" />
                        Add {selectedTitle === 'current' ? 'Incoming' : 'Current'} to Aliases
                      </Button>
                    )}
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <button
                      type="button"
                      onClick={() => setSelectedTitle('current')}
                      className={cn(
                        'flex flex-col gap-1.5 rounded-xl border p-3.5 text-left transition-all cursor-pointer',
                        selectedTitle === 'current'
                          ? 'border-primary bg-primary/5 ring-1 ring-primary'
                          : 'border-border bg-card hover:bg-muted/40'
                      )}
                    >
                      <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                        Keep Current
                      </span>
                      <p className="text-sm font-medium break-words leading-snug">{currentValues.title || '—'}</p>
                    </button>

                    <button
                      type="button"
                      onClick={() => setSelectedTitle('incoming')}
                      className={cn(
                        'flex flex-col gap-1.5 rounded-xl border p-3.5 text-left transition-all cursor-pointer',
                        selectedTitle === 'incoming'
                          ? 'border-primary bg-primary/5 ring-1 ring-primary'
                          : 'border-border bg-card hover:bg-muted/40'
                      )}
                    >
                      <span className="text-[10px] font-bold uppercase tracking-wider text-primary">
                        Use Incoming
                      </span>
                      <p className="text-sm font-medium break-words leading-snug">{incomingValues.title || '—'}</p>
                    </button>
                  </div>
                </div>
              )}

              {/* 3. Aliases Management */}
              {showAliasesSection && (
                <div className="space-y-2.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold flex items-center gap-1.5">
                      <Layers className="size-4 text-muted-foreground" />
                      Aliases ({selectedAliases.length})
                    </span>
                  </div>

                  {/* Active Aliases Typeable Pill Box */}
                  <div className="flex flex-wrap items-center gap-2 p-3 rounded-xl border border-border bg-card focus-within:ring-1 focus-within:ring-ring focus-within:border-ring min-h-11 cursor-text">
                    {selectedAliases.map((alias) => (
                      <Badge
                        key={alias}
                        variant="secondary"
                        className="text-xs gap-1.5 py-1 px-2.5 font-normal"
                      >
                        <span className="max-w-xs truncate">{alias}</span>
                        <button
                          type="button"
                          onClick={() => removeAlias(alias)}
                          className="rounded-full hover:bg-muted-foreground/20 p-0.5 cursor-pointer ml-0.5"
                          title={`Remove "${alias}"`}
                        >
                          <X className="size-3" />
                        </button>
                      </Badge>
                    ))}
                    <input
                      type="text"
                      value={aliasInput}
                      onChange={(e) => handleAliasInputChange(e.target.value)}
                      onKeyDown={handleAliasInputKeyDown}
                      placeholder={selectedAliases.length === 0 ? "Type an alias and press Enter..." : "Add alias..."}
                      aria-label="Add alias"
                      className="flex-1 min-w-[140px] bg-transparent text-xs outline-none placeholder:text-muted-foreground"
                    />
                  </div>

                  {/* Click-to-re-add unselected aliases */}
                  {unselectedAliases.length > 0 && (
                    <div className="flex flex-wrap items-center gap-2 pt-1">
                      <span className="text-xs text-muted-foreground">Available to add:</span>
                      {unselectedAliases.map((alias) => (
                        <button
                          key={alias}
                          type="button"
                          onClick={() => addAlias(alias)}
                          className="text-xs px-2.5 py-1 rounded-full border border-dashed border-muted-foreground/40 hover:border-foreground text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-1"
                        >
                          <Plus className="size-3" />
                          <span className="max-w-xs truncate">{alias}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* 4. Tags / Genres Management */}
              {showTagsSection && (
                <div className="space-y-2.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold flex items-center gap-1.5">
                      <Compass className="size-4 text-muted-foreground" />
                      Tags & Genres ({selectedTags.length})
                    </span>
                    <div className="flex items-center gap-1.5">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedTags(mergeStringArrays(currentValues.tags, incomingValues.tags))}
                        className="h-7 text-xs px-2.5 cursor-pointer"
                      >
                        Merge
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedTags([...currentValues.tags])}
                        className="h-7 text-xs px-2.5 cursor-pointer"
                      >
                        Current
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedTags([...incomingValues.tags])}
                        className="h-7 text-xs px-2.5 cursor-pointer"
                      >
                        Incoming
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedTags([])}
                        className="h-7 text-xs px-2.5 text-muted-foreground hover:text-destructive cursor-pointer"
                      >
                        Clear
                      </Button>
                    </div>
                  </div>

                  {/* Active Tags Typeable Pill Box */}
                  <div className="flex flex-wrap items-center gap-2 p-3 rounded-xl border border-border bg-card focus-within:ring-1 focus-within:ring-ring focus-within:border-ring min-h-11 cursor-text">
                    {selectedTags.map((tag) => (
                      <Badge
                        key={tag}
                        variant="secondary"
                        className="text-xs gap-1.5 py-1 px-2.5 font-normal"
                      >
                        <span>{tag}</span>
                        <button
                          type="button"
                          onClick={() => removeTag(tag)}
                          className="rounded-full hover:bg-muted-foreground/20 p-0.5 cursor-pointer ml-0.5"
                          title={`Remove tag "${tag}"`}
                        >
                          <X className="size-3" />
                        </button>
                      </Badge>
                    ))}
                    <input
                      type="text"
                      value={tagInput}
                      onChange={(e) => handleTagInputChange(e.target.value)}
                      onKeyDown={handleTagInputKeyDown}
                      placeholder={selectedTags.length === 0 ? "Type a tag and press Enter..." : "Add tag..."}
                      aria-label="Add tag"
                      className="flex-1 min-w-[120px] bg-transparent text-xs outline-none placeholder:text-muted-foreground"
                    />
                  </div>

                  {/* Click-to-re-add unselected tags */}
                  {unselectedTags.length > 0 && (
                    <div className="flex flex-wrap items-center gap-2 pt-1">
                      <span className="text-xs text-muted-foreground">Unselected tags:</span>
                      {unselectedTags.map((tag) => (
                        <button
                          key={tag}
                          type="button"
                          onClick={() => addTag(tag)}
                          className="text-xs px-2.5 py-1 rounded-full border border-dashed border-muted-foreground/40 hover:border-foreground text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-1"
                        >
                          <Plus className="size-3" />
                          <span>{tag}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* 5. Synopsis / Description Comparison */}
              {showDescSection && (
                <div className="space-y-2.5">
                  <span className="text-xs font-semibold">Synopsis / Description</span>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <button
                      type="button"
                      onClick={() => setSelectedDescription('current')}
                      className={cn(
                        'flex flex-col gap-2 rounded-xl border p-4 text-left transition-all cursor-pointer',
                        selectedDescription === 'current'
                          ? 'border-primary bg-primary/5 ring-1 ring-primary'
                          : 'border-border bg-card hover:bg-muted/40'
                      )}
                    >
                      <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                        Keep Current
                      </span>
                      <p className="text-xs text-muted-foreground line-clamp-6 leading-relaxed whitespace-pre-wrap">
                        {currentValues.description || '—'}
                      </p>
                    </button>

                    <button
                      type="button"
                      onClick={() => setSelectedDescription('incoming')}
                      className={cn(
                        'flex flex-col gap-2 rounded-xl border p-4 text-left transition-all cursor-pointer',
                        selectedDescription === 'incoming'
                          ? 'border-primary bg-primary/5 ring-1 ring-primary'
                          : 'border-border bg-card hover:bg-muted/40'
                      )}
                    >
                      <span className="text-[10px] font-bold uppercase tracking-wider text-primary">
                        Use Incoming
                      </span>
                      <p className="text-xs text-foreground line-clamp-6 leading-relaxed whitespace-pre-wrap">
                        {incomingValues.description || '—'}
                      </p>
                    </button>
                  </div>
                </div>
              )}

              {/* 6. Authors & Artists Comparison */}
              {(showAuthorsSection || showArtistsSection) && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
                  {showAuthorsSection && (
                    <div className="space-y-2.5">
                      <span className="text-xs font-semibold">Authors</span>
                      <div className="space-y-2">
                        <button
                          type="button"
                          onClick={() => setSelectedAuthorsMode('current')}
                          className={cn(
                            'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
                            selectedAuthorsMode === 'current'
                              ? 'border-primary bg-primary/10 text-foreground font-medium'
                              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                          )}
                        >
                          <span className="text-[10px] uppercase font-bold text-muted-foreground">Current:</span>
                          <span className="truncate max-w-[220px]">{currentValues.authors.join(', ') || '—'}</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedAuthorsMode('incoming')}
                          className={cn(
                            'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
                            selectedAuthorsMode === 'incoming'
                              ? 'border-primary bg-primary/10 text-foreground font-medium'
                              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                          )}
                        >
                          <span className="text-[10px] uppercase font-bold text-primary">Incoming:</span>
                          <span className="truncate max-w-[220px]">{incomingValues.authors.join(', ') || '—'}</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedAuthorsMode('merged')}
                          className={cn(
                            'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
                            selectedAuthorsMode === 'merged'
                              ? 'border-primary bg-primary/10 text-foreground font-medium'
                              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                          )}
                        >
                          <span className="text-[10px] uppercase font-bold text-muted-foreground">Merged:</span>
                          <span className="truncate max-w-[220px]">
                            {mergeStringArrays(currentValues.authors, incomingValues.authors).join(', ') || '—'}
                          </span>
                        </button>
                      </div>
                    </div>
                  )}

                  {showArtistsSection && (
                    <div className="space-y-2.5">
                      <span className="text-xs font-semibold">Artists</span>
                      <div className="space-y-2">
                        <button
                          type="button"
                          onClick={() => setSelectedArtistsMode('current')}
                          className={cn(
                            'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
                            selectedArtistsMode === 'current'
                              ? 'border-primary bg-primary/10 text-foreground font-medium'
                              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                          )}
                        >
                          <span className="text-[10px] uppercase font-bold text-muted-foreground">Current:</span>
                          <span className="truncate max-w-[220px]">{currentValues.artists.join(', ') || '—'}</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedArtistsMode('incoming')}
                          className={cn(
                            'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
                            selectedArtistsMode === 'incoming'
                              ? 'border-primary bg-primary/10 text-foreground font-medium'
                              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                          )}
                        >
                          <span className="text-[10px] uppercase font-bold text-primary">Incoming:</span>
                          <span className="truncate max-w-[220px]">{incomingValues.artists.join(', ') || '—'}</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedArtistsMode('merged')}
                          className={cn(
                            'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
                            selectedArtistsMode === 'merged'
                              ? 'border-primary bg-primary/10 text-foreground font-medium'
                              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                          )}
                        >
                          <span className="text-[10px] uppercase font-bold text-muted-foreground">Merged:</span>
                          <span className="truncate max-w-[220px]">
                            {mergeStringArrays(currentValues.artists, incomingValues.artists).join(', ') || '—'}
                          </span>
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* 7. External Links Comparison */}
              {showExternalLinksSection && (
                <div className="space-y-2.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold flex items-center gap-1.5">
                      <LinkIcon className="size-4 text-muted-foreground" />
                      External Links
                    </span>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <button
                      type="button"
                      onClick={() => setSelectedExternalLinksMode('current')}
                      className={cn(
                        'flex flex-col gap-2 rounded-xl border p-3.5 text-left text-xs transition-colors cursor-pointer',
                        selectedExternalLinksMode === 'current'
                          ? 'border-primary bg-primary/10 text-foreground font-medium ring-1 ring-primary/40'
                          : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                      )}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase font-bold text-muted-foreground">Current:</span>
                        <span className="text-[10px] text-muted-foreground">({currentValues.externalLinks.length})</span>
                      </div>
                      <div className="flex flex-wrap gap-1">
                        {currentValues.externalLinks.length === 0 ? (
                          <span className="text-xs text-muted-foreground">—</span>
                        ) : (
                          currentValues.externalLinks.map((link: ExternalLink, idx: number) => (
                            <span
                              key={`${link.url}-${idx}`}
                              className="inline-flex items-center gap-1 text-[11px] bg-secondary/80 rounded px-1.5 py-0.5 max-w-full truncate"
                            >
                              <LinkIcon className="size-2.5 shrink-0" />
                              <span className="truncate">{link.label || link.provider || link.url}</span>
                            </span>
                          ))
                        )}
                      </div>
                    </button>

                    <button
                      type="button"
                      onClick={() => setSelectedExternalLinksMode('incoming')}
                      className={cn(
                        'flex flex-col gap-2 rounded-xl border p-3.5 text-left text-xs transition-colors cursor-pointer',
                        selectedExternalLinksMode === 'incoming'
                          ? 'border-primary bg-primary/10 text-foreground font-medium ring-1 ring-primary/40'
                          : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                      )}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase font-bold text-primary">Incoming:</span>
                        <span className="text-[10px] text-muted-foreground">({incomingValues.externalLinks.length})</span>
                      </div>
                      <div className="flex flex-wrap gap-1">
                        {incomingValues.externalLinks.length === 0 ? (
                          <span className="text-xs text-muted-foreground">—</span>
                        ) : (
                          incomingValues.externalLinks.map((link: ExternalLink, idx: number) => (
                            <span
                              key={`${link.url}-${idx}`}
                              className="inline-flex items-center gap-1 text-[11px] bg-secondary/80 rounded px-1.5 py-0.5 max-w-full truncate"
                            >
                              <LinkIcon className="size-2.5 shrink-0" />
                              <span className="truncate">{link.label || link.provider || link.url}</span>
                            </span>
                          ))
                        )}
                      </div>
                    </button>

                    <button
                      type="button"
                      onClick={() => setSelectedExternalLinksMode('merged')}
                      className={cn(
                        'flex flex-col gap-2 rounded-xl border p-3.5 text-left text-xs transition-colors cursor-pointer',
                        selectedExternalLinksMode === 'merged'
                          ? 'border-primary bg-primary/10 text-foreground font-medium ring-1 ring-primary/40'
                          : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
                      )}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] uppercase font-bold text-muted-foreground">Merged:</span>
                        <span className="text-[10px] text-muted-foreground">
                          ({mergedExternalLinks.length})
                        </span>
                      </div>
                      <div className="flex flex-wrap gap-1">
                        {mergedExternalLinks.length === 0 ? (
                          <span className="text-xs text-muted-foreground">—</span>
                        ) : (
                          mergedExternalLinks.map((link: ExternalLink, idx: number) => (
                            <span
                              key={`${link.url}-${idx}`}
                              className="inline-flex items-center gap-1 text-[11px] bg-secondary/80 rounded px-1.5 py-0.5 max-w-full truncate"
                            >
                              <LinkIcon className="size-2.5 shrink-0" />
                              <span className="truncate">{link.label || link.provider || link.url}</span>
                            </span>
                          ))
                        )}
                      </div>
                    </button>
                  </div>
                </div>
              )}

              {/* 8. Attributes Comparative Grid */}
              {showAttributesSection && (
                <div className="space-y-2.5">
                  <span className="text-xs font-semibold">Attributes</span>
                  <div className="rounded-xl border border-border divide-y divide-border bg-card overflow-hidden">
                    {/* Header Row */}
                    <div className="grid grid-cols-3 bg-muted/50 py-2.5 px-4 text-xs font-semibold text-muted-foreground">
                      <span>Field</span>
                      <span>Current</span>
                      <span>Incoming ({selectedProvider?.name})</span>
                    </div>

                    {/* Publisher */}
                    {(incomingValues.publisher || !diffOnly) && (
                      <div className="grid grid-cols-3 py-3 px-4 items-center text-xs gap-3">
                        <span className="font-medium text-muted-foreground flex items-center gap-2">
                          <Building2 className="size-3.5" /> Publisher
                        </span>
                        <button
                          type="button"
                          onClick={() => setSelectedPublisher('current')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer',
                            selectedPublisher === 'current'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {currentValues.publisher || '—'}
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedPublisher('incoming')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer',
                            selectedPublisher === 'incoming'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {incomingValues.publisher || '—'}
                        </button>
                      </div>
                    )}

                    {/* Year */}
                    {(incomingValues.releaseYear > 0 || !diffOnly) && (
                      <div className="grid grid-cols-3 py-3 px-4 items-center text-xs gap-3">
                        <span className="font-medium text-muted-foreground flex items-center gap-2">
                          <Calendar className="size-3.5" /> Release Year
                        </span>
                        <button
                          type="button"
                          onClick={() => setSelectedReleaseYear('current')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer',
                            selectedReleaseYear === 'current'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {currentValues.releaseYear > 0 ? currentValues.releaseYear : '—'}
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedReleaseYear('incoming')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer',
                            selectedReleaseYear === 'incoming'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {incomingValues.releaseYear > 0 ? incomingValues.releaseYear : '—'}
                        </button>
                      </div>
                    )}

                    {/* Content Rating */}
                    {(incomingValues.contentRating || !diffOnly) && (
                      <div className="grid grid-cols-3 py-3 px-4 items-center text-xs gap-3">
                        <span className="font-medium text-muted-foreground flex items-center gap-2">
                          <ShieldAlert className="size-3.5" /> Content Rating
                        </span>
                        <button
                          type="button"
                          onClick={() => setSelectedContentRating('current')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer capitalize',
                            selectedContentRating === 'current'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {currentValues.contentRating || '—'}
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedContentRating('incoming')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer capitalize',
                            selectedContentRating === 'incoming'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {incomingValues.contentRating || '—'}
                        </button>
                      </div>
                    )}

                    {/* Country */}
                    {(incomingValues.country || !diffOnly) && (
                      <div className="grid grid-cols-3 py-3 px-4 items-center text-xs gap-3">
                        <span className="font-medium text-muted-foreground flex items-center gap-2">
                          <Globe className="size-3.5" /> Country
                        </span>
                        <button
                          type="button"
                          onClick={() => setSelectedCountry('current')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer uppercase',
                            selectedCountry === 'current'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {currentValues.country || '—'}
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedCountry('incoming')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer uppercase',
                            selectedCountry === 'incoming'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {incomingValues.country || '—'}
                        </button>
                      </div>
                    )}

                    {/* Reading Mode */}
                    {(incomingValues.readingMode || !diffOnly) && (
                      <div className="grid grid-cols-3 py-3 px-4 items-center text-xs gap-3">
                        <span className="font-medium text-muted-foreground flex items-center gap-2">
                          <BookOpen className="size-3.5" /> Reading Mode
                        </span>
                        <button
                          type="button"
                          onClick={() => setSelectedReadingMode('current')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer uppercase',
                            selectedReadingMode === 'current'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {currentValues.readingMode || '—'}
                        </button>
                        <button
                          type="button"
                          onClick={() => setSelectedReadingMode('incoming')}
                          className={cn(
                            'rounded-md p-2 text-left truncate transition-colors cursor-pointer uppercase',
                            selectedReadingMode === 'incoming'
                              ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
                              : 'hover:bg-muted text-muted-foreground'
                          )}
                        >
                          {incomingValues.readingMode || '—'}
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Dialog Footer */}
        <div className="px-6 py-4 border-t border-border bg-card/60 shrink-0 flex items-center justify-between gap-3">
          <Button
            type="button"
            variant="outline"
            onClick={() => setStep('search')}
            className="text-xs cursor-pointer"
          >
            Back
          </Button>

          <Button
            type="button"
            onClick={() => importMutation.mutate()}
            disabled={importMutation.isPending}
            className="gap-2 text-xs cursor-pointer"
          >
            {importMutation.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Download className="size-3.5" />
            )}
            Import & Bind Provider
          </Button>
        </div>
      </div>
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className={cn(
          'transition-all duration-200',
          step === 'compare'
            ? 'sm:max-w-4xl md:max-w-4xl lg:max-w-5xl w-[95vw] max-h-[90vh] h-[85vh] p-0 overflow-hidden flex flex-col'
            : 'sm:max-w-3xl md:max-w-3xl w-[90vw] p-6'
        )}
      >
        {isLoadingDetails && !selectedRemoteManga ? (
          <div className="flex flex-col items-center justify-center py-20 space-y-3">
            <Loader2 className="size-8 animate-spin text-primary" />
            <p className="text-xs font-medium text-muted-foreground">
              Fetching metadata from {selectedProvider?.name || selectedProviderId}...
            </p>
          </div>
        ) : step === 'search' ? (
          renderSearchStep()
        ) : (
          renderCompareStep()
        )}
      </DialogContent>
    </Dialog>
  );
};
