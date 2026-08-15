import React, { useState, useEffect } from 'react';
import { Plus, Trash2, Save } from 'lucide-react';
import { Manga, ExternalLink } from '../types/api';
import { useUpdateLibraryMangaMutation } from '../api/hooks';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

const readingModeOptions: Record<string, string> = {
  rtl: 'Right to Left (Manga)',
  ltr: 'Left to Right (Comic)',
  vertical: 'Vertical (Gapped)',
  longstrip: 'Longstrip (Webtoon)',
};

const contentRatingOptions: Record<string, string> = {
  safe: 'Safe',
  suggestive: 'Suggestive',
  mature: 'Mature',
  erotica: 'Erotica',
};

const externalLinkProviderOptions: Record<string, string> = {
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

interface EditMetadataDialogProps {
  manga: Manga;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved?: () => void;
}

export const EditMetadataDialog: React.FC<EditMetadataDialogProps> = ({
  manga,
  open,
  onOpenChange,
  onSaved,
}) => {
  const getInitialReadingMode = (m: Manga) =>
    m.content?.reading_mode ||
    m.meta?.content?.reading_mode ||
    m.readingMode ||
    m.reading_mode ||
    m.readingDirection ||
    m.meta?.reading_direction ||
    'rtl';

  const [title, setTitle] = useState(manga.title || '');
  const [aliasesInput, setAliasesInput] = useState((manga.aliases || manga.meta?.aliases || []).join(', '));
  const [description, setDescription] = useState(manga.description || manga.meta?.description || '');
  const [readingMode, setReadingMode] = useState(getInitialReadingMode(manga));
  const [contentRating, setContentRating] = useState(manga.contentRating || manga.meta?.content_rating || 'safe');
  const [publisher, setPublisher] = useState(manga.publisher || manga.meta?.publisher || '');
  const [releaseYear, setReleaseYear] = useState<number>(manga.releaseYear || manga.meta?.release_year || 0);
  const [country, setCountry] = useState(manga.country || 'JP');
  const [tagsInput, setTagsInput] = useState((manga.tags || manga.genres || manga.meta?.tags || []).join(', '));
  const [shelvesInput, setShelvesInput] = useState((manga.shelves || []).join(', '));
  const [externalLinks, setExternalLinks] = useState<ExternalLink[]>(manga.externalLinks || []);

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

  const handleSubmit = (e: React.FormEvent) => {
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
        provider_id: manga.content?.provider_id || manga.meta?.content?.provider_id || manga.contentProviderId || manga.sourceId || '',
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
  };

  const handleAddLink = () => {
    setExternalLinks((prev) => [...prev, { provider: 'custom', label: 'Custom Link', url: '' }]);
  };

  const handleRemoveLink = (index: number) => {
    setExternalLinks((prev) => prev.filter((_, i) => i !== index));
  };

  const handleLinkChange = (index: number, key: keyof ExternalLink, val: string) => {
    setExternalLinks((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], [key]: val };
      return next;
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl sm:max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-xl font-bold">Edit Series Metadata</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          {/* Title */}
          <div>
            <label className="text-xs font-semibold text-foreground mb-1 block">Title</label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Primary Manga Title"
              className="text-xs"
              required
            />
          </div>

          {/* Aliases */}
          <div>
            <label className="text-xs font-semibold text-foreground mb-1 block">Title Aliases (comma separated)</label>
            <Input
              value={aliasesInput}
              onChange={(e) => setAliasesInput(e.target.value)}
              placeholder="Alternative title 1, Alternative title 2"
              className="text-xs"
            />
          </div>

          {/* Description */}
          <div>
            <label className="text-xs font-semibold text-foreground mb-1 block">Synopsis / Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Series synopsis..."
              className="w-full h-24 rounded-md border border-input bg-background px-3 py-2 text-xs ring-offset-background placeholder:text-muted-foreground focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring"
            />
          </div>

          {/* Reading Mode & Content Rating */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">Reading Mode</label>
              <Select value={readingMode} onValueChange={(v) => v && setReadingMode(v)}>
                <SelectTrigger className="text-xs">
                  <SelectValue placeholder="Reading mode">
                    {readingModeOptions[readingMode] || readingMode}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="rtl" className="text-xs">Right to Left (Manga)</SelectItem>
                  <SelectItem value="ltr" className="text-xs">Left to Right (Comic)</SelectItem>
                  <SelectItem value="vertical" className="text-xs">Vertical (Gapped)</SelectItem>
                  <SelectItem value="longstrip" className="text-xs">Longstrip (Webtoon)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">Content Rating</label>
              <Select value={contentRating} onValueChange={(v) => v && setContentRating(v)}>
                <SelectTrigger className="text-xs">
                  <SelectValue placeholder="Content rating">
                    {contentRatingOptions[contentRating] || contentRating}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="safe" className="text-xs">Safe</SelectItem>
                  <SelectItem value="suggestive" className="text-xs">Suggestive</SelectItem>
                  <SelectItem value="mature" className="text-xs">Mature</SelectItem>
                  <SelectItem value="erotica" className="text-xs">Erotica</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Publisher, Release Year, Country */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">Publisher</label>
              <Input
                value={publisher}
                onChange={(e) => setPublisher(e.target.value)}
                placeholder="e.g. Shueisha"
                className="text-xs"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">Release Year</label>
              <Input
                type="number"
                value={releaseYear || ''}
                onChange={(e) => setReleaseYear(parseInt(e.target.value) || 0)}
                placeholder="e.g. 2020"
                className="text-xs"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">Country</label>
              <Input
                value={country}
                onChange={(e) => setCountry(e.target.value)}
                placeholder="JP, KR, CN"
                className="text-xs"
              />
            </div>
          </div>

          {/* Tags & Shelves */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">Tags (comma separated)</label>
              <Input
                value={tagsInput}
                onChange={(e) => setTagsInput(e.target.value)}
                placeholder="Fantasy, Action, type:manga"
                className="text-xs"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">Shelves (comma separated)</label>
              <Input
                value={shelvesInput}
                onChange={(e) => setShelvesInput(e.target.value)}
                placeholder="Favorites, Must Read"
                className="text-xs"
              />
            </div>
          </div>

          {/* External Links */}
          <div className="space-y-2 border-t border-border/50 pt-3">
            <div className="flex items-center justify-between">
              <label className="text-xs font-semibold text-foreground">External Links</label>
              <Button type="button" variant="outline" size="sm" onClick={handleAddLink} className="gap-1 text-xs h-7 cursor-pointer">
                <Plus className="size-3" /> Add Link
              </Button>
            </div>

            {externalLinks.length === 0 ? (
              <p className="text-xs text-muted-foreground">No external links linked.</p>
            ) : (
              <div className="space-y-2">
                {externalLinks.map((link, idx) => (
                  <div key={idx} className="flex items-center gap-2">
                    <Select
                      value={link.provider}
                      onValueChange={(val) => handleLinkChange(idx, 'provider', val || 'custom')}
                    >
                      <SelectTrigger className="w-28 text-xs bg-background border-border">
                        <SelectValue placeholder="provider">
                          {externalLinkProviderOptions[link.provider] || link.provider}
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {Object.entries(externalLinkProviderOptions).map(([val, label]) => (
                          <SelectItem key={val} value={val} className="text-xs">
                            {label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      value={link.label}
                      onChange={(e) => handleLinkChange(idx, 'label', e.target.value)}
                      placeholder="label"
                      className="w-28 text-xs"
                    />
                    <Input
                      value={link.url}
                      onChange={(e) => handleLinkChange(idx, 'url', e.target.value)}
                      placeholder="https://..."
                      className="flex-1 text-xs"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemoveLink(idx)}
                      className="h-8 w-8 p-0 text-destructive hover:text-destructive cursor-pointer"
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <DialogFooter className="pt-4 border-t border-border/50">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} className="cursor-pointer">
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending} className="gap-2 cursor-pointer">
              <Save className="size-4" />
              {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
