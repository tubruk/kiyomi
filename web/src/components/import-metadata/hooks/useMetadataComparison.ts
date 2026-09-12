import { useState, useMemo, useCallback, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../../api/client';
import { Manga, MangaMetadata, Source, ExternalLink, ProviderRef } from '../../../types/api';
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
  mode?: 'import' | 'merge';
  incomingManga?: Manga;
  sourceMangaIds?: string[];
}

export const useMetadataComparison = ({
  manga,
  sources,
  selectedProviderId,
  onSuccess,
  onClose,
  mode = 'import',
  incomingManga,
  sourceMangaIds,
}: UseMetadataComparisonOptions) => {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const [selectedRemoteManga, setSelectedRemoteManga] = useState<Manga | null>(
    mode === 'merge' && incomingManga ? incomingManga : null
  );
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
  const [selectedProvidersMode, setSelectedProvidersMode] = useState<MultiChoice>('merged');

  const selectedProvider = sources.find((s) => s.id === selectedProviderId);

  // Normalized current values — read from manga.metadata (canonical per-concern store).
  // Description now lives at manga.metadata.description (was incorrectly read from
  // manga.meta?.description previously, which was always empty).
  const currentValues: MetadataValues = useMemo(() => {
    return {
      title: manga.metadata.title,
      coverUrl:
        manga.metadata.cover_url ??
        manga.metadata.coverUrl ??
        manga.coverUrl ??
        manga.cover ??
        manga.coverAssetUrl ??
        '',
      description: manga.metadata.description,
      authors: mergeStringArrays(manga.metadata.authors, manga.authors, manga.author ? [manga.author] : []),
      artists: mergeStringArrays(manga.metadata.artists, manga.artists, manga.artist ? [manga.artist] : []),
      publishers: mergeStringArrays(manga.metadata.publishers, manga.publishers, manga.publisher ? [manga.publisher] : []),
      tags: mergeStringArrays(manga.metadata.tags, manga.tags, manga.genres),
      aliases: manga.metadata.aliases ?? manga.aliases ?? [],
      publisher: (manga.metadata.publishers && manga.metadata.publishers.length > 0 ? manga.metadata.publishers[0] : manga.publisher) ?? '',
      releaseYear:
        manga.metadata.releaseYear ??
        manga.metadata.release_year ??
        manga.releaseYear ??
        manga.release_year ??
        0,
      startDate:
        manga.metadata.startDate ??
        manga.metadata.start_date ??
        manga.startDate ??
        manga.start_date ??
        '',
      endDate:
        manga.metadata.endDate ??
        manga.metadata.end_date ??
        manga.endDate ??
        manga.end_date ??
        '',
      contentRating:
        manga.metadata.content_rating ??
        manga.metadata.contentRating ??
        manga.contentRating ??
        '',
      country: manga.metadata.country ?? manga.country ?? '',
      readingMode:
        manga.bindings?.content?.reading_mode ??
        manga.readingMode ??
        manga.reading_mode ??
        manga.readingDirection ??
        '',
      externalLinks: ((manga.metadata.externalLinks && manga.metadata.externalLinks.length > 0)
        ? manga.metadata.externalLinks
        : (manga.externalLinks && manga.externalLinks.length > 0)
        ? manga.externalLinks
        : []) as ExternalLink[],
      providers: manga.bindings.providers,
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
        providers: [],
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
    const remoteExtLinks: ExternalLink[] = selectedRemoteManga.externalLinks ?? [];
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
        selectedRemoteManga.publisher ? [selectedRemoteManga.publisher] : []
      ),
      tags: mergeStringArrays(selectedRemoteManga.tags, selectedRemoteManga.genres),
      aliases: selectedRemoteManga.aliases || [],
      publisher: selectedRemoteManga.publisher?.trim() || '',
      releaseYear: selectedRemoteManga.releaseYear || selectedRemoteManga.release_year || 0,
      startDate: selectedRemoteManga.startDate || selectedRemoteManga.start_date || '',
      endDate: selectedRemoteManga.endDate || selectedRemoteManga.end_date || '',
      contentRating: selectedRemoteManga.contentRating || '',
      country: selectedRemoteManga.country || '',
      readingMode: selectedRemoteManga.readingMode || selectedRemoteManga.reading_mode || selectedRemoteManga.readingDirection || '',
      externalLinks: incomingLinks,
      providers: selectedRemoteManga.bindings?.providers ?? [],
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
      providers: Boolean(
        mode === 'merge' &&
        incomingValues.providers.length > 0 &&
        !areArraySetsEqual(
          currentValues.providers.map((p) => `${p.provider_id}:${p.provider_manga_id}`),
          incomingValues.providers.map((p) => `${p.provider_id}:${p.provider_manga_id}`)
        )
      ),
    };
  }, [currentValues, incomingValues, mode]);

  const mergedExternalLinks = useMemo(
    () => mergeExternalLinkArrays(currentValues.externalLinks, incomingValues.externalLinks),
    [currentValues.externalLinks, incomingValues.externalLinks]
  );

  // Initialize comparison selections from a remote manga
  const initComparisonState = useCallback((remote: Manga) => {
    setSelectedRemoteManga(remote);

    // Current — read from manga.metadata
    const cTitle = manga.metadata.title;
    const cCover =
      manga.metadata.cover_url ??
      manga.metadata.coverUrl ??
      manga.coverUrl ??
      manga.cover ??
      manga.coverAssetUrl ??
      '';
    const cDesc = manga.metadata.description;
    const cAuthors = mergeStringArrays(manga.metadata.authors, manga.authors, manga.author ? [manga.author] : []);
    const cArtists = mergeStringArrays(manga.metadata.artists, manga.artists, manga.artist ? [manga.artist] : []);
    const cPublishers = mergeStringArrays(manga.metadata.publishers, manga.publishers, manga.publisher ? [manga.publisher] : []);
    const cTags = mergeStringArrays(manga.metadata.tags, manga.tags, manga.genres);
    const cAliases = manga.metadata.aliases ?? manga.aliases ?? [];
    const cExtLinks: ExternalLink[] = (manga.metadata.externalLinks && manga.metadata.externalLinks.length > 0)
      ? manga.metadata.externalLinks
      : (manga.externalLinks && manga.externalLinks.length > 0)
      ? manga.externalLinks
      : [];

    // Incoming
    const inTitle = remote.title?.trim() || '';
    const inCover = remote.coverUrl || remote.cover || remote.coverAssetUrl || '';
    const inDesc = remote.description?.trim() || '';
    const inAuthors = mergeStringArrays(remote.authors, remote.author ? [remote.author] : []);
    const inArtists = mergeStringArrays(remote.artists, remote.artist ? [remote.artist] : []);
    const inPublishers = mergeStringArrays(remote.publishers, remote.publisher ? [remote.publisher] : []);
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
    const remoteExtLinks: ExternalLink[] = remote.externalLinks ?? [];
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
    setSelectedReleaseYear((remote.releaseYear || remote.release_year) ? 'incoming' : 'current');
    const inStartDate = remote.startDate || remote.start_date || '';
    const inEndDate = remote.endDate || remote.end_date || '';
    setSelectedStartDate(inStartDate ? 'incoming' : 'current');
    setSelectedEndDate(inEndDate ? 'incoming' : 'current');
    setSelectedContentRating((remote.contentRating) ? 'incoming' : 'current');
    setSelectedCountry((remote.country) ? 'incoming' : 'current');
    setSelectedReadingMode((remote.readingMode || remote.reading_mode || remote.readingDirection) ? 'incoming' : 'current');
  }, [manga, selectedProviderId, selectedProvider]);

  // In merge mode, run initialization once when the incoming manga is provided
  // so that field-by-field selections reflect the actual diffs.
  useEffect(() => {
    if (mode === 'merge' && incomingManga) {
      initComparisonState(incomingManga);
      setSelectedProvidersMode('merged');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, incomingManga?.id]);

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
        currentValues.title && incomingValues.title.toLowerCase() !== incomingValues.title.toLowerCase()
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
    setSelectedProvidersMode('merged');
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
    setSelectedProvidersMode('merged');
  };

  // Import & Save Mutation
  const importMutation = useMutation({
    mutationFn: async () => {
      // Merge mode: skip the selection check — the source manga is provided
      // via incomingManga/sourceMangaIds props and the search step is bypassed.
      if (mode !== 'merge') {
        if (!selectedRemoteManga || !selectedProviderId) throw new Error('No selection');
      }

      // Merge mode: build merge payload and call the merge endpoint
      if (mode === 'merge') {
        const incoming = selectedRemoteManga ?? incomingManga;
        if (!incoming) throw new Error('No incoming manga');
        const incomingId = incoming.id;
        const mergeMapScalar = (choice: Choice): 'keep' | `source:${string}` =>
          choice === 'current' ? 'keep' : `source:${incomingId}`;

        // For array fields that only accept 'merge' (aliases/tags/authors/artists/publishers)
        // we always send 'merge' regardless of UI selection.
        const mergePayload = {
          keep_manga_id: manga.id,
          source_manga_ids: sourceMangaIds && sourceMangaIds.length > 0 ? sourceMangaIds : [incomingId],
          metadata: {
            title: mergeMapScalar(selectedTitle),
            description: mergeMapScalar(selectedDescription),
            aliases: 'merge' as const,
            tags: 'merge' as const,
            authors: 'merge' as const,
            artists: 'merge' as const,
            publishers: 'merge' as const,
            release_year: mergeMapScalar(selectedReleaseYear),
            cover_url: mergeMapScalar(selectedCover),
            providers: 'merge' as const,
          },
        };

        return api.mergeLibraryManga(mergePayload);
      }

      const fieldsToPatch: Partial<MangaMetadata> = {};

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
      } else if (selectedPublishersMode === 'merged') {
        const mergedPubs = mergeStringArrays(currentValues.publishers, incomingValues.publishers);
        fieldsToPatch.publishers = mergedPubs;
      }

      fieldsToPatch.tags = selectedTags;
      fieldsToPatch.aliases = selectedAliases;

      if (selectedReleaseYear === 'incoming' && incomingValues.releaseYear) {
        fieldsToPatch.release_year = incomingValues.releaseYear;
      }
      if (selectedStartDate === 'incoming' && incomingValues.startDate) {
        fieldsToPatch.start_date = incomingValues.startDate;
      }
      if (selectedEndDate === 'incoming' && incomingValues.endDate) {
        fieldsToPatch.end_date = incomingValues.endDate;
      }
      if (selectedContentRating === 'incoming' && incomingValues.contentRating) {
        fieldsToPatch.content_rating = incomingValues.contentRating;
      }
      if (selectedCountry === 'incoming' && incomingValues.country) {
        fieldsToPatch.country = incomingValues.country;
      }
      if (selectedExternalLinksMode === 'incoming') {
        fieldsToPatch.external_links = incomingValues.externalLinks;
      } else if (selectedExternalLinksMode === 'merged') {
        const merged = mergeExternalLinkArrays(currentValues.externalLinks, incomingValues.externalLinks);
        fieldsToPatch.external_links = merged;
      }

      // 1. Patch metadata (per-concern endpoint)
      if (Object.keys(fieldsToPatch).length > 0) {
        await api.patchLibraryMangaMetadata(manga.id, fieldsToPatch);
      }

      // 2. Add provider binding (per-concern endpoint)
      const providerMangaId =
        selectedRemoteManga!.id ||
        selectedRemoteManga!.contentRemoteId ||
        selectedRemoteManga!.url ||
        '';
      const ref: ProviderRef = {
        provider_id: selectedProviderId!,
        provider_manga_id: providerMangaId,
        manga_title: selectedRemoteManga!.title || incomingValues.title,
      };

      // 3. Update reading_mode on content binding (per-concern endpoint)
      if (
        selectedReadingMode === 'incoming' &&
        incomingValues.readingMode &&
        incomingValues.readingMode.toLowerCase() !== currentValues.readingMode.toLowerCase()
      ) {
        await api.setActiveContentSource(manga.id, {
          provider_id: selectedProviderId!,
          provider_manga_id: providerMangaId,
          reading_mode: incomingValues.readingMode,
        });
      }

      return api.addBinding(manga.id, ref);
    },
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.metadata(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.bindings(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(manga.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.providers(manga.id) });
      if (mode === 'merge') {
        showToast('Manga merged successfully', 'success');
      } else {
        const providerName = sources.find((s) => s.id === selectedProviderId)?.name || 'Provider';
        showToast(`Metadata imported from "${providerName}"`, 'success');
      }
      onClose();
      // Forward the result so consumers know the operation completed. The
      // shape is merge-mode-Manga for merge calls and partial bindings for
      // import calls — callers should treat it as opaque or re-fetch.
      onSuccess?.(result as unknown as Manga);
    },
    onError: (err: any) => {
      const verb = mode === 'merge' ? 'merge' : 'import';
      showToast(`Failed to ${verb}: ${err.message}`, 'error');
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
    selectedProvidersMode,
    setSelectedProvidersMode,
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
    mode,
    incomingManga,
  };
};
