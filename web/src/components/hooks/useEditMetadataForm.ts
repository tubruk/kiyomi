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
  m.content?.reading_mode ||
  m.meta?.content?.reading_mode ||
  m.readingMode ||
  m.reading_mode ||
  m.readingDirection ||
  m.meta?.reading_direction ||
  'rtl';

export function useEditMetadataForm({
  manga,
  open,
  onOpenChange,
  onSaved,
}: UseEditMetadataFormOptions) {
  const [title, setTitle] = useState(manga.title || '');
  const [aliases, setAliases] = useState<string[]>(
    manga.aliases || manga.meta?.aliases || []
  );
  const [description, setDescription] = useState(
    manga.description || manga.meta?.description || ''
  );
  const [authors, setAuthors] = useState<string[]>(
    manga.authors && manga.authors.length > 0
      ? manga.authors
      : manga.author
      ? [manga.author]
      : manga.meta?.authors || []
  );
  const [artists, setArtists] = useState<string[]>(
    manga.artists && manga.artists.length > 0
      ? manga.artists
      : manga.artist
      ? [manga.artist]
      : manga.meta?.artists || []
  );
  const [publishers, setPublishers] = useState<string[]>(
    manga.publishers && manga.publishers.length > 0
      ? manga.publishers
      : manga.publisher
      ? [manga.publisher]
      : manga.meta?.publishers || (manga.meta?.publisher ? [manga.meta.publisher] : [])
  );
  const [readingMode, setReadingMode] = useState(getInitialReadingMode(manga));
  const [contentRating, setContentRating] = useState(
    manga.contentRating || manga.meta?.content_rating || 'safe'
  );
  const [releaseYear, setReleaseYear] = useState<number>(
    manga.releaseYear || manga.release_year || manga.meta?.release_year || manga.meta?.releaseYear || 0
  );
  const [startDate, setStartDate] = useState(
    manga.startDate || manga.start_date || manga.meta?.start_date || manga.meta?.startDate || ''
  );
  const [endDate, setEndDate] = useState(
    manga.endDate || manga.end_date || manga.meta?.end_date || manga.meta?.endDate || ''
  );
  const [country, setCountry] = useState(manga.country || manga.meta?.country || 'JP');
  const [tags, setTags] = useState<string[]>(
    manga.tags || manga.genres || manga.meta?.tags || []
  );
  const [shelves, setShelves] = useState<string[]>(
    manga.shelves || manga.collections || manga.meta?.collections || []
  );
  const [externalLinks, setExternalLinks] = useState<ExternalLink[]>(
    manga.externalLinks || manga.meta?.external_links || []
  );

  useEffect(() => {
    if (open) {
      setTitle(manga.title || '');
      setAliases(manga.aliases || manga.meta?.aliases || []);
      setDescription(manga.description || manga.meta?.description || '');
      setAuthors(
        manga.authors && manga.authors.length > 0
          ? manga.authors
          : manga.author
          ? [manga.author]
          : manga.meta?.authors || []
      );
      setArtists(
        manga.artists && manga.artists.length > 0
          ? manga.artists
          : manga.artist
          ? [manga.artist]
          : manga.meta?.artists || []
      );
      setPublishers(
        manga.publishers && manga.publishers.length > 0
          ? manga.publishers
          : manga.publisher
          ? [manga.publisher]
          : manga.meta?.publishers || (manga.meta?.publisher ? [manga.meta.publisher] : [])
      );
      setReadingMode(getInitialReadingMode(manga));
      setContentRating(manga.contentRating || manga.meta?.content_rating || 'safe');
      setReleaseYear(manga.releaseYear || manga.release_year || manga.meta?.release_year || manga.meta?.releaseYear || 0);
      setStartDate(manga.startDate || manga.start_date || manga.meta?.start_date || manga.meta?.startDate || '');
      setEndDate(manga.endDate || manga.end_date || manga.meta?.end_date || manga.meta?.endDate || '');
      setCountry(manga.country || manga.meta?.country || 'JP');
      setTags(manga.tags || manga.genres || manga.meta?.tags || []);
      setShelves(manga.shelves || manga.collections || manga.meta?.collections || []);
      setExternalLinks(manga.externalLinks || manga.meta?.external_links || []);
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

      const payload: Partial<Manga> = {
        ...manga,
        title,
        aliases,
        description,
        authors,
        artists,
        publishers,
        publisher: publishers.length > 0 ? publishers[0] : '',
        content: {
          ...(manga.content || manga.meta?.content),
          provider_id:
            manga.content?.provider_id ||
            manga.meta?.content?.provider_id ||
            manga.contentProviderId ||
            manga.sourceId ||
            '',
          reading_mode: readingMode,
        },
        readingMode,
        reading_mode: readingMode,
        readingDirection: readingMode,
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
