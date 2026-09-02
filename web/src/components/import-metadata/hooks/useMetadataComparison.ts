import { useState, useMemo, useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../../api/client';
import { Manga, Source, ExternalLink, ProviderRef } from '../../../types/api';
import { queryKeys } from '../../../lib/queryKeys';
import { useToast } from '../../../context/ToastContext';
import {
  areArraySetsEqual,
  mergeStringArrays,
  areExternalLinkSetsEqual,
  mergeExternalLinkArrays,
} from '../../../utils/metadataCompare';
import { Choice, MultiChoice, MetadataValues, MetadataDiffs } from '../types';

interface UseMetadataComparisonOptions {
  manga: Manga;
  sources: Source[];
  selectedProviderId: string;
  onSuccess?: (manga: Manga) => void;
  onClose: () => void;
}

export const useMetadataComparison = ({
  manga,
  sources,
  selectedProviderId,
  onSuccess,
  onClose,
}: UseMetadataComparisonOptions) => {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const [selectedRemoteManga, setSelectedRemoteManga] = useState<Manga | null>(null);
  const [diffOnly, setDiffOnly] = useState(true);

  // Field selection states
  const [selectedTitle, setSelectedTitle] = useState<Choice>('incoming');
  const [selectedCover, setSelectedCover] = useState<Choice>('incoming');
  const [selectedDescription, setSelectedDescription] = useState<Choice>('incoming');
  const [selectedAuthorsMode, setSelectedAuthorsMode] = useState<MultiChoice>('incoming');
  const [selectedArtistsMode, setSelectedArtistsMode] = useState<MultiChoice>('incoming');
  const [selectedPublishersMode, setSelectedPublishersMode] = useState<MultiChoice>('incoming');
  const [selectedExternalLinksMode, setSelectedExternalLinksMode] = useState<MultiChoice>('incoming');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [selectedAliases, setSelectedAliases] = useState<string[]>([]);
  const [aliasInput, setAliasInput] = useState('');
  const [tagInput, setTagInput] = useState('');
  const [selectedPublisher, setSelectedPublisher] = useState<Choice>('incoming');
  const [selectedReleaseYear, setSelectedReleaseYear] = useState<Choice>('incoming');
  const [selectedStartDate, setSelectedStartDate] = useState<Choice>('incoming');
  const [selectedEndDate, setSelectedEndDate] = useState<Choice>('incoming');
  const [selectedContentRating, setSelectedContentRating] = useState<Choice>('incoming');
  const [selectedCountry, setSelectedCountry] = useState<Choice>('incoming');
  const [selectedReadingMode, setSelectedReadingMode] = useState<Choice>('incoming');

  const selectedProvider = sources.find((s) => s.id === selectedProviderId);

  // Normalized current values
  const currentValues: MetadataValues = useMemo(() => {
    return {
      title: manga.title || manga.meta?.title || '',
      coverUrl: manga.coverUrl || manga.cover || manga.coverAssetUrl || manga.meta?.cover_url || '',
      description: manga.description || manga.meta?.description || '',
      authors: mergeStringArrays(manga.authors, manga.author ? [manga.author] : (manga.meta?.authors || [])),
      artists: mergeStringArrays(manga.artists, manga.artist ? [manga.artist] : (manga.meta?.artists || [])),
      publishers: mergeStringArrays(
        manga.publishers,
        manga.publisher ? [manga.publisher] : (manga.meta?.publishers || (manga.meta?.publisher ? [manga.meta.publisher] : []))
      ),
      tags: mergeStringArrays(manga.tags, manga.genres || manga.meta?.tags),
      aliases: manga.aliases || manga.meta?.aliases || [],
      publisher: manga.publisher || manga.meta?.publisher || '',
      releaseYear: manga.releaseYear || manga.release_year || manga.meta?.release_year || manga.meta?.releaseYear || 0,
      startDate: manga.startDate || manga.start_date || manga.meta?.start_date || manga.meta?.startDate || '',
      endDate: manga.endDate || manga.end_date || manga.meta?.end_date || manga.meta?.endDate || '',
      contentRating: manga.contentRating || manga.meta?.content_rating || '',
      country: manga.country || manga.meta?.country || '',
      readingMode: manga.readingMode || manga.reading_mode || manga.readingDirection || manga.content?.reading_mode || manga.meta?.content?.reading_mode || '',
      externalLinks: ((manga.externalLinks && manga.externalLinks.length > 0)
        ? manga.externalLinks
        : (manga.meta?.external_links || [])) as ExternalLink[],
    };
  }, [manga]);

  // Normalized incoming values
  const incomingValues: MetadataValues = useMemo(() => {
    if (!selectedRemoteManga) {
      return {
        title: '',
        coverUrl: '',
        description: '',
        authors: [],
        artists: [],
        publishers: [],
        tags: [],
        aliases: [],
        publisher: '',
        releaseYear: 0,
        startDate: '',
        endDate: '',
        contentRating: '',
        country: '',
        readingMode: '',
        externalLinks: [],
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
      publishers: mergeStringArrays(
        selectedRemoteManga.publishers,
        selectedRemoteManga.publisher ? [selectedRemoteManga.publisher] : (selectedRemoteManga.meta?.publishers || (selectedRemoteManga.meta?.publisher ? [selectedRemoteManga.meta.publisher] : []))
      ),
      tags: mergeStringArrays(selectedRemoteManga.tags, selectedRemoteManga.genres),
      aliases: selectedRemoteManga.aliases || [],
      publisher: selectedRemoteManga.publisher?.trim() || '',
      releaseYear: selectedRemoteManga.releaseYear || selectedRemoteManga.release_year || selectedRemoteManga.meta?.release_year || selectedRemoteManga.meta?.releaseYear || 0,
      startDate: selectedRemoteManga.startDate || selectedRemoteManga.start_date || selectedRemoteManga.meta?.start_date || selectedRemoteManga.meta?.startDate || '',
      endDate: selectedRemoteManga.endDate || selectedRemoteManga.end_date || selectedRemoteManga.meta?.end_date || selectedRemoteManga.meta?.endDate || '',
      contentRating: selectedRemoteManga.contentRating || selectedRemoteManga.meta?.content_rating || '',
      country: selectedRemoteManga.country || selectedRemoteManga.meta?.country || '',
      readingMode: selectedRemoteManga.readingMode || selectedRemoteManga.reading_mode || selectedRemoteManga.readingDirection || '',
      externalLinks: incomingLinks,
    };
  }, [selectedRemoteManga, selectedProviderId, selectedProvider]);

  // Diff checks across fields
  const diffs: MetadataDiffs = useMemo(() => {
    return {
      cover: Boolean(incomingValues.coverUrl && incomingValues.coverUrl !== currentValues.coverUrl),
      title: Boolean(incomingValues.title && incomingValues.title.toLowerCase() !== currentValues.title.toLowerCase()),
      description: Boolean(incomingValues.description && incomingValues.description.trim() !== currentValues.description.trim()),
      authors: Boolean(incomingValues.authors.length > 0 && !areArraySetsEqual(currentValues.authors, incomingValues.authors)),
      artists: Boolean(incomingValues.artists.length > 0 && !areArraySetsEqual(currentValues.artists, incomingValues.artists)),
      publishers: Boolean(incomingValues.publishers.length > 0 && !areArraySetsEqual(currentValues.publishers, incomingValues.publishers)),
      tags: Boolean(incomingValues.tags.length > 0 && !areArraySetsEqual(currentValues.tags, incomingValues.tags)),
      aliases: Boolean(incomingValues.aliases.length > 0 && !areArraySetsEqual(currentValues.aliases, incomingValues.aliases)),
      publisher: Boolean(incomingValues.publisher && incomingValues.publisher.toLowerCase() !== currentValues.publisher.toLowerCase()),
      releaseYear: Boolean(incomingValues.releaseYear > 0 && incomingValues.releaseYear !== currentValues.releaseYear),
      startDate: Boolean(incomingValues.startDate && incomingValues.startDate.toLowerCase() !== currentValues.startDate.toLowerCase()),
      endDate: Boolean(incomingValues.endDate && incomingValues.endDate.toLowerCase() !== currentValues.endDate.toLowerCase()),
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

  // Initialize comparison selections from a remote manga
  const initComparisonState = useCallback((remote: Manga) => {
    setSelectedRemoteManga(remote);

    // Current
    const cTitle = manga.title || manga.meta?.title || '';
    const cCover = manga.coverUrl || manga.cover || manga.coverAssetUrl || manga.meta?.cover_url || '';
    const cDesc = manga.description || manga.meta?.description || '';
    const cAuthors = mergeStringArrays(manga.authors, manga.author ? [manga.author] : (manga.meta?.authors || []));
    const cArtists = mergeStringArrays(manga.artists, manga.artist ? [manga.artist] : (manga.meta?.artists || []));
    const cPublishers = mergeStringArrays(
      manga.publishers,
      manga.publisher ? [manga.publisher] : (manga.meta?.publishers || (manga.meta?.publisher ? [manga.meta.publisher] : []))
    );
    const cTags = mergeStringArrays(manga.tags, manga.genres || manga.meta?.tags);
    const cAliases = manga.aliases || manga.meta?.aliases || [];
    const cExtLinks: ExternalLink[] = (manga.externalLinks && manga.externalLinks.length > 0)
      ? manga.externalLinks
      : ((manga.meta as any)?.external_links || []);

    // Incoming
    const inTitle = remote.title?.trim() || '';
    const inCover = remote.coverUrl || remote.cover || remote.coverAssetUrl || '';
    const inDesc = remote.description?.trim() || '';
    const inAuthors = mergeStringArrays(remote.authors, remote.author ? [remote.author] : []);
    const inArtists = mergeStringArrays(remote.artists, remote.artist ? [remote.artist] : []);
    const inPublishers = mergeStringArrays(
      remote.publishers,
      remote.publisher ? [remote.publisher] : (remote.meta?.publishers || (remote.meta?.publisher ? [remote.meta.publisher] : []))
    );
    const inTags = mergeStringArrays(remote.tags, remote.genres);
    const inAliases = remote.aliases || [];
    const inExtLinks: ExternalLink[] = [];
    const inUrl = remote.url?.trim() || (remote.id?.startsWith('http') ? remote.id.trim() : '');
    if (inUrl) {
      inExtLinks.push({
        provider: selectedProviderId,
        label: selectedProvider?.name || selectedProviderId,
        url: inUrl,
      });
    }
    const remoteExtLinks: ExternalLink[] = remote.externalLinks || (remote.meta as any)?.external_links || [];
    for (const link of remoteExtLinks) {
      if (link && link.url?.trim() && !inExtLinks.some((l) => l.url.trim().toLowerCase() === link.url.trim().toLowerCase())) {
        inExtLinks.push(link);
      }
    }

    // Defaults
    setSelectedTitle(inTitle && inTitle !== cTitle ? 'incoming' : 'current');
    setSelectedCover(inCover && inCover !== cCover ? 'incoming' : 'current');
    setSelectedDescription(inDesc && inDesc !== cDesc ? 'incoming' : 'current');
    setSelectedAuthorsMode(inAuthors.length > 0 ? (areArraySetsEqual(cAuthors, inAuthors) ? 'current' : 'incoming') : 'current');
    setSelectedArtistsMode(inArtists.length > 0 ? (areArraySetsEqual(cArtists, inArtists) ? 'current' : 'incoming') : 'current');
    setSelectedPublishersMode(inPublishers.length > 0 ? (areArraySetsEqual(cPublishers, inPublishers) ? 'current' : 'incoming') : 'current');
    setSelectedExternalLinksMode(
      inExtLinks.length > 0
        ? areExternalLinkSetsEqual(cExtLinks, inExtLinks)
          ? 'current'
          : 'merged'
        : 'current'
    );
    setSelectedTags(mergeStringArrays(cTags, inTags));
    setSelectedAliases(mergeStringArrays(cAliases, inAliases));

    setSelectedPublisher(remote.publisher?.trim() ? 'incoming' : 'current');
    setSelectedReleaseYear((remote.releaseYear || remote.release_year || remote.meta?.release_year || remote.meta?.releaseYear) ? 'incoming' : 'current');
    const inStartDate = remote.startDate || remote.start_date || remote.meta?.start_date || remote.meta?.startDate || '';
    const inEndDate = remote.endDate || remote.end_date || remote.meta?.end_date || remote.meta?.endDate || '';
    setSelectedStartDate(inStartDate ? 'incoming' : 'current');
    setSelectedEndDate(inEndDate ? 'incoming' : 'current');
    setSelectedContentRating((remote.contentRating || remote.meta?.content_rating) ? 'incoming' : 'current');
    setSelectedCountry((remote.country || remote.meta?.country) ? 'incoming' : 'current');
    setSelectedReadingMode((remote.readingMode || remote.reading_mode || remote.readingDirection) ? 'incoming' : 'current');
  }, [manga, selectedProviderId, selectedProvider]);

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

  // Bulk Actions
  const handleAcceptIncoming = () => {
    if (!selectedRemoteManga) return;
    setSelectedTitle(incomingValues.title ? 'incoming' : 'current');
    setSelectedCover(incomingValues.coverUrl ? 'incoming' : 'current');
    setSelectedDescription(incomingValues.description ? 'incoming' : 'current');
    setSelectedAuthorsMode(incomingValues.authors.length > 0 ? 'incoming' : 'current');
    setSelectedArtistsMode(incomingValues.artists.length > 0 ? 'incoming' : 'current');
    setSelectedPublishersMode(incomingValues.publishers.length > 0 ? 'incoming' : 'current');
    setSelectedTags(incomingValues.tags.length > 0 ? [...incomingValues.tags] : [...currentValues.tags]);
    setSelectedAliases(mergeStringArrays(currentValues.aliases, incomingValues.aliases));
    setSelectedPublisher(incomingValues.publisher ? 'incoming' : 'current');
    setSelectedReleaseYear(incomingValues.releaseYear ? 'incoming' : 'current');
    setSelectedStartDate(incomingValues.startDate ? 'incoming' : 'current');
    setSelectedEndDate(incomingValues.endDate ? 'incoming' : 'current');
    setSelectedContentRating(incomingValues.contentRating ? 'incoming' : 'current');
    setSelectedCountry(incomingValues.country ? 'incoming' : 'current');
    setSelectedReadingMode(incomingValues.readingMode ? 'incoming' : 'current');
    setSelectedExternalLinksMode(incomingValues.externalLinks.length > 0 ? 'incoming' : 'current');
  };

  const handleKeepCurrent = () => {
    setSelectedTitle('current');
    setSelectedCover('current');
    setSelectedDescription('current');
    setSelectedAuthorsMode('current');
    setSelectedArtistsMode('current');
    setSelectedPublishersMode('current');
    setSelectedTags([...currentValues.tags]);
    setSelectedAliases([...currentValues.aliases]);
    setSelectedPublisher('current');
    setSelectedReleaseYear('current');
    setSelectedStartDate('current');
    setSelectedEndDate('current');
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
      if (selectedPublishersMode === 'incoming') {
        fieldsToPatch.publishers = incomingValues.publishers;
        if (incomingValues.publishers.length > 0) {
          fieldsToPatch.publisher = incomingValues.publishers[0];
        }
      } else if (selectedPublishersMode === 'merged') {
        const mergedPubs = mergeStringArrays(currentValues.publishers, incomingValues.publishers);
        fieldsToPatch.publishers = mergedPubs;
        if (mergedPubs.length > 0) {
          fieldsToPatch.publisher = mergedPubs[0];
        }
      }

      fieldsToPatch.tags = selectedTags;
      fieldsToPatch.aliases = selectedAliases;

      if (selectedPublisher === 'incoming' && incomingValues.publisher) {
        fieldsToPatch.publisher = incomingValues.publisher;
      }
      if (selectedReleaseYear === 'incoming' && incomingValues.releaseYear) {
        fieldsToPatch.release_year = incomingValues.releaseYear;
        fieldsToPatch.releaseYear = incomingValues.releaseYear;
      }
      if (selectedStartDate === 'incoming' && incomingValues.startDate) {
        fieldsToPatch.start_date = incomingValues.startDate;
        fieldsToPatch.startDate = incomingValues.startDate;
      }
      if (selectedEndDate === 'incoming' && incomingValues.endDate) {
        fieldsToPatch.end_date = incomingValues.endDate;
        fieldsToPatch.endDate = incomingValues.endDate;
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
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.providers(manga.id) });
      const providerName = sources.find((s) => s.id === selectedProviderId)?.name || 'Provider';
      showToast(`Metadata imported from "${providerName}"`, 'success');
      onClose();
      onSuccess?.(updatedManga);
    },
    onError: (err: any) => {
      showToast(`Failed to import: ${err.message}`, 'error');
    },
  });

  const resetComparisonState = () => {
    setSelectedRemoteManga(null);
    setDiffOnly(true);
    setAliasInput('');
    setTagInput('');
  };

  return {
    selectedRemoteManga,
    setSelectedRemoteManga,
    diffOnly,
    setDiffOnly,
    selectedTitle,
    setSelectedTitle,
    selectedCover,
    setSelectedCover,
    selectedDescription,
    setSelectedDescription,
    selectedAuthorsMode,
    setSelectedAuthorsMode,
    selectedArtistsMode,
    setSelectedArtistsMode,
    selectedPublishersMode,
    setSelectedPublishersMode,
    selectedExternalLinksMode,
    setSelectedExternalLinksMode,
    selectedTags,
    setSelectedTags,
    selectedAliases,
    setSelectedAliases,
    aliasInput,
    setAliasInput,
    tagInput,
    setTagInput,
    selectedPublisher,
    setSelectedPublisher,
    selectedReleaseYear,
    setSelectedReleaseYear,
    selectedStartDate,
    setSelectedStartDate,
    selectedEndDate,
    setSelectedEndDate,
    selectedContentRating,
    setSelectedContentRating,
    selectedCountry,
    setSelectedCountry,
    selectedReadingMode,
    setSelectedReadingMode,
    currentValues,
    incomingValues,
    diffs,
    mergedExternalLinks,
    initComparisonState,
    unselectedTags,
    addTag,
    removeTag,
    handleTagInputChange,
    handleTagInputKeyDown,
    unselectedAliases,
    addAlias,
    removeAlias,
    handleAliasInputChange,
    handleAliasInputKeyDown,
    handleAcceptIncoming,
    handleKeepCurrent,
    importMutation,
    resetComparisonState,
  };
};
