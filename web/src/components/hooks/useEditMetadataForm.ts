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
  const [aliasesInput, setAliasesInput] = useState(
    (manga.aliases || manga.meta?.aliases || []).join(', ')
  );
  const [description, setDescription] = useState(
    manga.description || manga.meta?.description || ''
  );
  const [readingMode, setReadingMode] = useState(getInitialReadingMode(manga));
  const [contentRating, setContentRating] = useState(
    manga.contentRating || manga.meta?.content_rating || 'safe'
  );
  const [publisher, setPublisher] = useState(manga.publisher || manga.meta?.publisher || '');
  const [releaseYear, setReleaseYear] = useState<number>(
    manga.releaseYear || manga.meta?.release_year || 0
  );
  const [country, setCountry] = useState(manga.country || 'JP');
  const [tagsInput, setTagsInput] = useState(
    (manga.tags || manga.genres || manga.meta?.tags || []).join(', ')
  );
  const [shelvesInput, setShelvesInput] = useState((manga.shelves || []).join(', '));
  const [externalLinks, setExternalLinks] = useState<ExternalLink[]>(
    manga.externalLinks || []
  );

  useEffect(() => {
    if (open) {
      setTitle(manga.title || '');
      setAliasesInput((manga.aliases || manga.meta?.aliases || []).join(', '));
      setDescription(manga.description || manga.meta?.description || '');
      setReadingMode(getInitialReadingMode(manga));
      setContentRating(manga.contentRating || manga.meta?.content_rating || 'safe');
      setPublisher(manga.publisher || manga.meta?.publisher || '');
      setReleaseYear(manga.releaseYear || manga.meta?.release_year || 0);
      setCountry(manga.country || 'JP');
      setTagsInput((manga.tags || manga.genres || manga.meta?.tags || []).join(', '));
      setShelvesInput((manga.shelves || []).join(', '));
      setExternalLinks(manga.externalLinks || []);
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

      const seenAliases = new Set<string>();
      const parsedAliases: string[] = [];
      aliasesInput
        .split(',')
        .map((a) => a.trim())
        .filter(Boolean)
        .forEach((a) => {
          const lower = a.toLowerCase();
          if (!seenAliases.has(lower)) {
            seenAliases.add(lower);
            parsedAliases.push(a);
          }
        });

      const parsedTags = tagsInput
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean);

      const parsedShelves = shelvesInput
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);

      const payload: Partial<Manga> = {
        ...manga,
        title,
        aliases: parsedAliases,
        description,
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
        publisher,
        releaseYear: Number(releaseYear),
        country,
        tags: parsedTags,
        shelves: parsedShelves,
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
      aliasesInput,
      description,
      readingMode,
      contentRating,
      publisher,
      releaseYear,
      country,
      tagsInput,
      shelvesInput,
      externalLinks,
      updateMutation,
      onOpenChange,
      onSaved,
    ]
  );

  return {
    title,
    setTitle,
    aliasesInput,
    setAliasesInput,
    description,
    setDescription,
    readingMode,
    setReadingMode,
    contentRating,
    setContentRating,
    publisher,
    setPublisher,
    releaseYear,
    setReleaseYear,
    country,
    setCountry,
    tagsInput,
    setTagsInput,
    shelvesInput,
    setShelvesInput,
    externalLinks,
    setExternalLinks,
    handleAddLink,
    handleRemoveLink,
    handleLinkChange,
    handleSubmit,
    isUpdating: updateMutation.isPending,
  };
}
