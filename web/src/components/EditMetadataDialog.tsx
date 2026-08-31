import React from 'react';
import { Plus, Trash2, Save } from 'lucide-react';
import { Manga } from '../types/api';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import {
  useEditMetadataForm,
  READING_MODE_OPTIONS,
  CONTENT_RATING_OPTIONS,
  EXTERNAL_LINK_PROVIDER_OPTIONS,
} from './hooks/useEditMetadataForm';

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
  const {
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
    handleAddLink,
    handleRemoveLink,
    handleLinkChange,
    handleSubmit,
    isUpdating,
  } = useEditMetadataForm({
    manga,
    open,
    onOpenChange,
    onSaved,
  });

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
            <label className="text-xs font-semibold text-foreground mb-1 block">
              Title Aliases (comma separated)
            </label>
            <Input
              value={aliasesInput}
              onChange={(e) => setAliasesInput(e.target.value)}
              placeholder="Alternative title 1, Alternative title 2"
              className="text-xs"
            />
          </div>

          {/* Description */}
          <div>
            <label className="text-xs font-semibold text-foreground mb-1 block">
              Synopsis / Description
            </label>
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
                    {READING_MODE_OPTIONS[readingMode] || readingMode}
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
                    {CONTENT_RATING_OPTIONS[contentRating] || contentRating}
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
              <label className="text-xs font-semibold text-foreground mb-1 block">
                Tags (comma separated)
              </label>
              <Input
                value={tagsInput}
                onChange={(e) => setTagsInput(e.target.value)}
                placeholder="Fantasy, Action, type:manga"
                className="text-xs"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground mb-1 block">
                Shelves (comma separated)
              </label>
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
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAddLink}
                className="gap-1 text-xs h-7 cursor-pointer"
              >
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
                          {EXTERNAL_LINK_PROVIDER_OPTIONS[link.provider] || link.provider}
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {Object.entries(EXTERNAL_LINK_PROVIDER_OPTIONS).map(([val, label]) => (
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
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              className="cursor-pointer"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isUpdating}
              className="gap-2 cursor-pointer"
            >
              <Save className="size-4" />
              {isUpdating ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
