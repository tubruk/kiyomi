import { useState, useEffect, useCallback } from 'react';
import { Manga, ExternalLink } from '../../types/api';
import { useUpdateLibraryMangaMutation } from '../../api/hooks';

export const READING_MODE_OPTIONS: Record<string, string> = {
  rtl: 'Right to Left (Manga)',
  ltr: 'Left to Right (Comic)',
  vertical: 'Vertical (Gapped)',
  longstrip: 'Longstrip (Webtoon)',
};

export const CONTENT_RATING_OPTIONS: Record<string, string> = {
  safe: 'Safe',
  suggestive: 'Suggestive',
  mature: 'Mature',
  erotica: 'Erotica',
};

export const EXTERNAL_LINK_PROVIDER_OPTIONS: Record<string, string> = {
  custom: 'Custom',
  anilist: 'AniList',
  myanimelist: 'MyAnimeList',
  kitsu: 'Kitsu',
  mangadex: 'MangaDex',
  mangaupdates: 'MangaUpdates',
  animeplanet: 'Anime-Planet',
  amazon: 'Amazon',
  ebookjapan: 'eBookJapan',
  cdjapan: 'CDJapan',
};

export interface UseEditMetadataFormOptions {
  manga: Manga;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved?: () => void;
}

export const getInitialReadingMode = (m: Manga) =>
  m.bindings?.content?.reading_mode ||
  m.content?.reading_mode ||
  m.readingMode ||
  m.reading_mode ||
  m.readingDirection ||
  'rtl';

export function useEditMetadataForm({
  manga,
  open,
  onOpenChange,
  onSaved,
}: UseEditMetadataFormOptions) {
  const [title, setTitle] = useState(manga.title || '');
  const [aliases, setAliases] = useState<string[]>(
    manga.metadata?.aliases?.length ? manga.metadata.aliases : manga.aliases ?? []
  );
  const [description, setDescription] = useState(
    manga.metadata?.description ?? manga.description ?? ''
  );
  const [authors, setAuthors] = useState<string[]>(
    manga.metadata?.authors && manga.metadata.authors.length > 0
      ? manga.metadata.authors
      : manga.authors && manga.authors.length > 0
      ? manga.authors
      : manga.author
      ? [manga.author]
      : []
  );
  const [artists, setArtists] = useState<string[]>(
    manga.metadata?.artists && manga.metadata.artists.length > 0
      ? manga.metadata.artists
      : manga.artists && manga.artists.length > 0
      ? manga.artists
      : manga.artist
      ? [manga.artist]
      : []
  );
  const [publishers, setPublishers] = useState<string[]>(
    manga.metadata?.publishers && manga.metadata.publishers.length > 0
      ? manga.metadata.publishers
      : manga.publishers && manga.publishers.length > 0
      ? manga.publishers
      : manga.publisher
      ? [manga.publisher]
      : []
  );
  const [readingMode, setReadingMode] = useState(getInitialReadingMode(manga));
  const [contentRating, setContentRating] = useState(
    manga.metadata?.content_rating ?? manga.contentRating ?? 'safe'
  );
  const [releaseYear, setReleaseYear] = useState<number>(
    manga.metadata?.releaseYear ?? manga.metadata?.release_year ?? manga.releaseYear ?? manga.release_year ?? 0
  );
  const [startDate, setStartDate] = useState(
    manga.metadata?.startDate ?? manga.metadata?.start_date ?? manga.startDate ?? manga.start_date ?? ''
  );
  const [endDate, setEndDate] = useState(
    manga.metadata?.endDate ?? manga.metadata?.end_date ?? manga.endDate ?? manga.end_date ?? ''
  );
  const [country, setCountry] = useState(manga.metadata?.country ?? manga.country ?? 'JP');
  const [tags, setTags] = useState<string[]>(
    manga.metadata?.tags?.length ? manga.metadata.tags : manga.tags?.length ? manga.tags : manga.genres ?? []
  );
  const [shelves, setShelves] = useState<string[]>(
    manga.shelves || manga.collections || manga.metadata?.collections || []
  );
  const [externalLinks, setExternalLinks] = useState<ExternalLink[]>(
    manga.metadata?.externalLinks?.length ? manga.metadata.externalLinks : manga.externalLinks ?? []
  );

  useEffect(() => {
    if (open) {
      setTitle(manga.title || '');
      setAliases(manga.metadata?.aliases?.length ? manga.metadata.aliases : manga.aliases ?? []);
      setDescription(manga.metadata?.description ?? manga.description ?? '');
      setAuthors(
        manga.metadata?.authors && manga.metadata.authors.length > 0
          ? manga.metadata.authors
          : manga.authors && manga.authors.length > 0
          ? manga.authors
          : manga.author
          ? [manga.author]
          : []
      );
      setArtists(
        manga.metadata?.artists && manga.metadata.artists.length > 0
          ? manga.metadata.artists
          : manga.artists && manga.artists.length > 0
          ? manga.artists
          : manga.artist
          ? [manga.artist]
          : []
      );
      setPublishers(
        manga.metadata?.publishers && manga.metadata.publishers.length > 0
          ? manga.metadata.publishers
          : manga.publishers && manga.publishers.length > 0
          ? manga.publishers
          : manga.publisher
          ? [manga.publisher]
          : []
      );
      setReadingMode(getInitialReadingMode(manga));
      setContentRating(manga.metadata?.content_rating ?? manga.contentRating ?? 'safe');
      setReleaseYear(manga.metadata?.releaseYear ?? manga.metadata?.release_year ?? manga.releaseYear ?? manga.release_year ?? 0);
      setStartDate(manga.metadata?.startDate ?? manga.metadata?.start_date ?? manga.startDate ?? manga.start_date ?? '');
      setEndDate(manga.metadata?.endDate ?? manga.metadata?.end_date ?? manga.endDate ?? manga.end_date ?? '');
      setCountry(manga.metadata?.country ?? manga.country ?? 'JP');
      setTags(manga.metadata?.tags?.length ? manga.metadata.tags : manga.tags?.length ? manga.tags : manga.genres ?? []);
      setShelves(manga.shelves || manga.collections || manga.metadata?.collections || []);
      setExternalLinks(manga.metadata?.externalLinks?.length ? manga.metadata.externalLinks : manga.externalLinks ?? []);
    }
  }, [open, manga]);

  const updateMutation = useUpdateLibraryMangaMutation();

  const handleAddLink = useCallback(() => {
    setExternalLinks((prev) => [...prev, { provider: 'custom', label: 'Custom Link', url: '' }]);
  }, []);

  const handleRemoveLink = useCallback((index: number) => {
    setExternalLinks((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const handleLinkChange = useCallback(
    (index: number, key: keyof ExternalLink, val: string) => {
      setExternalLinks((prev) => {
        const next = [...prev];
        next[index] = { ...next[index], [key]: val };
        return next;
      });
    },
    []
  );

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();

      const payload: Record<string, any> = {
        metadata: {
          title,
          aliases,
          description,
          authors,
          artists,
          publishers,
          publisher: publishers.length > 0 ? publishers[0] : '',
          contentRating,
          content_rating: contentRating,
          releaseYear: Number(releaseYear) || 0,
          release_year: Number(releaseYear) || 0,
          startDate,
          start_date: startDate,
          endDate,
          end_date: endDate,
          country,
          tags,
          shelves,
          collections: shelves,
          externalLinks: externalLinks.filter((l) => l.url.trim() !== ''),
          external_links: externalLinks.filter((l) => l.url.trim() !== ''),
        },
        bindings: {
          ...(manga.bindings || {}),
          content: {
            ...(manga.bindings?.content || manga.content || {}),
            provider_id:
              manga.bindings?.content?.provider_id ||
              manga.content?.provider_id ||
              manga.contentProviderId ||
              manga.sourceId ||
              '',
            reading_mode: readingMode,
          },
        },
      };

      updateMutation.mutate(
        { mangaId: manga.id, fields: payload },
        {
          onSuccess: () => {
            onOpenChange(false);
            if (onSaved) onSaved();
          },
        }
      );
    },
    [
      manga,
      title,
      aliases,
      description,
      authors,
      artists,
      publishers,
      readingMode,
      contentRating,
      releaseYear,
      startDate,
      endDate,
      country,
      tags,
      shelves,
      externalLinks,
      updateMutation,
      onOpenChange,
      onSaved,
    ]
  );

  return {
    title,
    setTitle,
    aliases,
    setAliases,
    description,
    setDescription,
    authors,
    setAuthors,
    artists,
    setArtists,
    publishers,
    setPublishers,
    readingMode,
    setReadingMode,
    contentRating,
    setContentRating,
    releaseYear,
    setReleaseYear,
    startDate,
    setStartDate,
    endDate,
    setEndDate,
    country,
    setCountry,
    tags,
    setTags,
    shelves,
    setShelves,
    externalLinks,
    setExternalLinks,
    handleAddLink,
    handleRemoveLink,
    handleLinkChange,
    handleSubmit,
    isUpdating: updateMutation.isPending,
  };
}
