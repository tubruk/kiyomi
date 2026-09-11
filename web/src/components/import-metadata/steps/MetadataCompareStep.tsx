import React from 'react';
import {
  Download,
  Loader2,
  Sparkles,
  RotateCcw,
  Plus,
  ArrowLeft,
  BookOpen,
  Calendar,
  ShieldAlert,
  Globe,
  Plug,
} from 'lucide-react';
import { Manga, Source, ExternalLink, ProviderRef } from '../../../types/api';
import { Choice, MultiChoice, MetadataValues, MetadataDiffs } from '../types';
import { DialogTitle } from '../../ui/dialog';
import { Button } from '../../ui/button';
import { Badge } from '../../ui/badge';
import { cn } from '../../../lib/utils';
import { MetadataCoverCompare } from '../fields/MetadataCoverCompare';
import { MetadataAliasesEditor } from '../fields/MetadataAliasesEditor';
import { MetadataTagsEditor } from '../fields/MetadataTagsEditor';
import { MetadataMultiChoiceField } from '../fields/MetadataMultiChoiceField';
import { MetadataFieldRow } from '../fields/MetadataFieldRow';

interface MetadataCompareStepProps {
  manga: Manga;
  selectedRemoteManga: Manga | null;
  selectedProvider?: Source;
  diffOnly: boolean;
  onSetDiffOnly: (diffOnly: boolean) => void;
  onKeepCurrent: () => void;
  onAcceptIncoming: () => void;
  onBack: () => void;
  onImport: () => void;
  isImporting: boolean;

  currentValues: MetadataValues;
  incomingValues: MetadataValues;
  diffs: MetadataDiffs;
  mergedExternalLinks: ExternalLink[];

  selectedTitle: Choice;
  onSelectTitle: (c: Choice) => void;
  selectedCover: Choice;
  onSelectCover: (c: Choice) => void;
  selectedDescription: Choice;
  onSelectDescription: (c: Choice) => void;
  selectedAuthorsMode: MultiChoice;
  onSelectAuthorsMode: (m: MultiChoice) => void;
  selectedArtistsMode: MultiChoice;
  onSelectArtistsMode: (m: MultiChoice) => void;
  selectedPublishersMode: MultiChoice;
  onSelectPublishersMode: (m: MultiChoice) => void;
  selectedExternalLinksMode: MultiChoice;
  onSelectExternalLinksMode: (m: MultiChoice) => void;
  selectedProvidersMode: MultiChoice;
  onSelectProvidersMode: (m: MultiChoice) => void;

  selectedTags: string[];
  unselectedTags: string[];
  tagInput: string;
  onTagInputChange: (val: string) => void;
  onTagInputKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  onAddTag: (tag: string) => void;
  onRemoveTag: (tag: string) => void;
  onSetSelectedTags: (tags: string[]) => void;

  selectedAliases: string[];
  unselectedAliases: string[];
  aliasInput: string;
  onAliasInputChange: (val: string) => void;
  onAliasInputKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  onAddAlias: (alias: string) => void;
  onRemoveAlias: (alias: string) => void;

  selectedReleaseYear: Choice;
  onSelectReleaseYear: (c: Choice) => void;
  selectedStartDate: Choice;
  onSelectStartDate: (c: Choice) => void;
  selectedEndDate: Choice;
  onSelectEndDate: (c: Choice) => void;
  selectedContentRating: Choice;
  onSelectContentRating: (c: Choice) => void;
  selectedCountry: Choice;
  onSelectCountry: (c: Choice) => void;
  selectedReadingMode: Choice;
  onSelectReadingMode: (c: Choice) => void;

  mode?: 'import' | 'merge';
}

export const MetadataCompareStep: React.FC<MetadataCompareStepProps> = ({
  manga,
  selectedRemoteManga,
  selectedProvider,
  diffOnly,
  onSetDiffOnly,
  onKeepCurrent,
  onAcceptIncoming,
  onBack,
  onImport,
  isImporting,

  currentValues,
  incomingValues,
  diffs,
  mergedExternalLinks,

  selectedTitle,
  onSelectTitle,
  selectedCover,
  onSelectCover,
  selectedDescription,
  onSelectDescription,
  selectedAuthorsMode,
  onSelectAuthorsMode,
  selectedArtistsMode,
  onSelectArtistsMode,
  selectedPublishersMode,
  onSelectPublishersMode,
  selectedExternalLinksMode,
  onSelectExternalLinksMode,
  // Providers field is locked to 'merged' in merge mode; state is held in the
  // parent for API consistency but the UI does not allow switching it.
  selectedProvidersMode: _selectedProvidersMode,
  onSelectProvidersMode: _onSelectProvidersMode,

  selectedTags,
  unselectedTags,
  tagInput,
  onTagInputChange,
  onTagInputKeyDown,
  onAddTag,
  onRemoveTag,
  onSetSelectedTags,

  selectedAliases,
  unselectedAliases,
  aliasInput,
  onAliasInputChange,
  onAliasInputKeyDown,
  onAddAlias,
  onRemoveAlias,

  selectedReleaseYear,
  onSelectReleaseYear,
  selectedStartDate,
  onSelectStartDate,
  selectedEndDate,
  onSelectEndDate,
  selectedContentRating,
  onSelectContentRating,
  selectedCountry,
  onSelectCountry,
  selectedReadingMode,
  onSelectReadingMode,

  mode = 'import',
}) => {
  const isTitleDiff = diffs.title;
  const isCoverDiff = diffs.cover;
  const isDescDiff = diffs.description;
  const isAuthorsDiff = diffs.authors;
  const isArtistsDiff = diffs.artists;
  const isPublishersDiff = diffs.publishers;
  const isTagsDiff = diffs.tags;
  const isAliasesDiff = diffs.aliases;
  const isExternalLinksDiff = diffs.externalLinks;
  const isProvidersDiff = diffs.providers;

  const isYearDiff = diffs.releaseYear;
  const isStartDateDiff = diffs.startDate;
  const isEndDateDiff = diffs.endDate;
  const isRatingDiff = diffs.contentRating;
  const isCountryDiff = diffs.country;
  const isReadingModeDiff = diffs.readingMode;

  const hasAnyAttributesDiff =
    isYearDiff || isStartDateDiff || isEndDateDiff || isRatingDiff || isCountryDiff || isReadingModeDiff;

  const hasAnyDiff =
    isCoverDiff ||
    isTitleDiff ||
    isDescDiff ||
    isAuthorsDiff ||
    isArtistsDiff ||
    isPublishersDiff ||
    isTagsDiff ||
    isAliasesDiff ||
    isExternalLinksDiff ||
    isProvidersDiff ||
    hasAnyAttributesDiff;

  const showCoverSection = incomingValues.coverUrl && (!diffOnly || isCoverDiff);
  const showTitleSection = incomingValues.title && (!diffOnly || isTitleDiff);
  const showDescSection = incomingValues.description && (!diffOnly || isDescDiff);
  const showAuthorsSection = incomingValues.authors.length > 0 && (!diffOnly || isAuthorsDiff);
  const showArtistsSection = incomingValues.artists.length > 0 && (!diffOnly || isArtistsDiff);
  const showPublishersSection = incomingValues.publishers.length > 0 && (!diffOnly || isPublishersDiff);
  const showTagsSection = incomingValues.tags.length > 0 && (!diffOnly || isTagsDiff);
  const showAliasesSection =
    currentValues.aliases.length > 0 ||
    incomingValues.aliases.length > 0 ||
    selectedAliases.length > 0 ||
    !diffOnly;
  const showExternalLinksSection =
    (incomingValues.externalLinks.length > 0 || currentValues.externalLinks.length > 0) &&
    (!diffOnly || isExternalLinksDiff);
  const showProvidersSection =
    mode === 'merge' &&
    (incomingValues.providers.length > 0 || currentValues.providers.length > 0) &&
    (!diffOnly || isProvidersDiff);
  const showAttributesSection = !diffOnly || hasAnyAttributesDiff;

  const showNothingInDiff = diffOnly && !hasAnyDiff;

  const renderProviderPills = (refs: ProviderRef[]) => {
    if (refs.length === 0) {
      return <span className="text-xs text-muted-foreground">—</span>;
    }
    return refs.map((ref, idx) => (
      <span
        key={`${ref.provider_id}-${ref.provider_manga_id}-${idx}`}
        className="inline-flex items-center gap-1 text-[11px] bg-secondary/80 rounded px-1.5 py-0.5 max-w-full truncate"
        title={`${ref.provider_id}: ${ref.provider_manga_id}`}
      >
        <Plug className="size-2.5 shrink-0" />
        <span className="truncate">{ref.manga_title || ref.provider_manga_id}</span>
      </span>
    ));
  };

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
              onClick={onBack}
              className="size-8 p-0 cursor-pointer shrink-0"
              title="Back to search"
            >
              <ArrowLeft className="size-4" />
            </Button>
            <div className="min-w-0">
              <DialogTitle className="text-base font-semibold truncate flex items-center gap-2">
                <span>{mode === 'merge' ? 'Merge Metadata' : 'Compare & Import Metadata'}</span>
                {selectedProvider?.name && (
                  <Badge variant="outline" className="text-[10px] font-normal shrink-0">
                    {selectedProvider.name}
                  </Badge>
                )}
              </DialogTitle>
            </div>
          </div>

          {/* Quick Action Toolbar */}
          <div className="flex flex-wrap items-center gap-2 shrink-0">
            <div className="flex items-center rounded-lg border border-border bg-muted/40 p-0.5 text-xs">
              <button
                type="button"
                onClick={() => onSetDiffOnly(true)}
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
                onClick={() => onSetDiffOnly(false)}
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
              onClick={onKeepCurrent}
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
              onClick={onAcceptIncoming}
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
              <MetadataCoverCompare
                currentCoverUrl={currentValues.coverUrl}
                currentCoverAssetUrl={manga.coverAssetUrl}
                mangaUrl={manga.url}
                incomingCoverUrl={incomingValues.coverUrl}
                remoteMangaUrl={selectedRemoteManga?.url}
                selectedCover={selectedCover}
                onSelectCover={onSelectCover}
              />
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
                      onClick={() => onAddAlias(targetTitleAlias)}
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
                    onClick={() => onSelectTitle('current')}
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
                    onClick={() => onSelectTitle('incoming')}
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
              <MetadataAliasesEditor
                selectedAliases={selectedAliases}
                unselectedAliases={unselectedAliases}
                aliasInput={aliasInput}
                onAliasInputChange={onAliasInputChange}
                onAliasInputKeyDown={onAliasInputKeyDown}
                onAddAlias={onAddAlias}
                onRemoveAlias={onRemoveAlias}
              />
            )}

            {/* 4. Tags / Genres Management */}
            {showTagsSection && (
              <MetadataTagsEditor
                currentTags={currentValues.tags}
                incomingTags={incomingValues.tags}
                selectedTags={selectedTags}
                unselectedTags={unselectedTags}
                tagInput={tagInput}
                onTagInputChange={onTagInputChange}
                onTagInputKeyDown={onTagInputKeyDown}
                onAddTag={onAddTag}
                onRemoveTag={onRemoveTag}
                onSetSelectedTags={onSetSelectedTags}
              />
            )}

            {/* 5. Synopsis / Description Comparison */}
            {showDescSection && (
              <div className="space-y-2.5">
                <span className="text-xs font-semibold">Synopsis / Description</span>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <button
                    type="button"
                    onClick={() => onSelectDescription('current')}
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
                    onClick={() => onSelectDescription('incoming')}
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

            {/* 6. Authors, Artists & Publishers Comparison */}
            {(showAuthorsSection || showArtistsSection || showPublishersSection) && (
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-5">
                {showAuthorsSection && (
                  <MetadataMultiChoiceField
                    type="strings"
                    title="Authors"
                    currentItems={currentValues.authors}
                    incomingItems={incomingValues.authors}
                    selectedMode={selectedAuthorsMode}
                    onSelectMode={onSelectAuthorsMode}
                  />
                )}

                {showArtistsSection && (
                  <MetadataMultiChoiceField
                    type="strings"
                    title="Artists"
                    currentItems={currentValues.artists}
                    incomingItems={incomingValues.artists}
                    selectedMode={selectedArtistsMode}
                    onSelectMode={onSelectArtistsMode}
                  />
                )}

                {showPublishersSection && (
                  <MetadataMultiChoiceField
                    type="strings"
                    title="Publishers"
                    currentItems={currentValues.publishers}
                    incomingItems={incomingValues.publishers}
                    selectedMode={selectedPublishersMode}
                    onSelectMode={onSelectPublishersMode}
                  />
                )}
              </div>
            )}

            {/* 7. External Links Comparison */}
            {showExternalLinksSection && (
              <MetadataMultiChoiceField
                type="links"
                currentLinks={currentValues.externalLinks}
                incomingLinks={incomingValues.externalLinks}
                mergedLinks={mergedExternalLinks}
                selectedMode={selectedExternalLinksMode}
                onSelectMode={onSelectExternalLinksMode}
              />
            )}

            {/* 7b. Provider Bindings Comparison (merge mode only) */}
            {showProvidersSection && (
              <div className="space-y-2.5">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold flex items-center gap-1.5">
                    <Plug className="size-4 text-muted-foreground" />
                    Provider Bindings
                  </span>
                  <span className="text-[10px] text-muted-foreground italic">
                    Provider bindings are always merged.
                  </span>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div className="rounded-xl border border-border bg-card p-3.5 space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase font-bold text-muted-foreground">Current:</span>
                      <span className="text-[10px] text-muted-foreground">({currentValues.providers.length})</span>
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {renderProviderPills(currentValues.providers)}
                    </div>
                  </div>
                  <div className="rounded-xl border border-primary bg-primary/10 p-3.5 space-y-2 ring-1 ring-primary/40">
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] uppercase font-bold text-primary">Merged (locked):</span>
                      <span className="text-[10px] text-muted-foreground">({incomingValues.providers.length + currentValues.providers.length})</span>
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {renderProviderPills([
                        ...currentValues.providers,
                        ...incomingValues.providers.filter(
                          (ip) => !currentValues.providers.some(
                            (cp) => cp.provider_id === ip.provider_id && cp.provider_manga_id === ip.provider_manga_id
                          )
                        ),
                      ])}
                    </div>
                  </div>
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

                  {/* Year */}
                  {(incomingValues.releaseYear > 0 || !diffOnly) && (
                    <MetadataFieldRow
                      label="Release Year"
                      icon={<Calendar className="size-3.5" />}
                      currentValue={currentValues.releaseYear}
                      incomingValue={incomingValues.releaseYear}
                      selectedValue={selectedReleaseYear}
                      onSelect={onSelectReleaseYear}
                    />
                  )}

                  {/* Start Date */}
                  {(incomingValues.startDate || !diffOnly) && (
                    <MetadataFieldRow
                      label="Start Date"
                      icon={<Calendar className="size-3.5" />}
                      currentValue={currentValues.startDate}
                      incomingValue={incomingValues.startDate}
                      selectedValue={selectedStartDate}
                      onSelect={onSelectStartDate}
                    />
                  )}

                  {/* End Date */}
                  {(incomingValues.endDate || !diffOnly) && (
                    <MetadataFieldRow
                      label="End Date"
                      icon={<Calendar className="size-3.5" />}
                      currentValue={currentValues.endDate}
                      incomingValue={incomingValues.endDate}
                      selectedValue={selectedEndDate}
                      onSelect={onSelectEndDate}
                    />
                  )}

                  {/* Content Rating */}
                  {(incomingValues.contentRating || !diffOnly) && (
                    <MetadataFieldRow
                      label="Content Rating"
                      icon={<ShieldAlert className="size-3.5" />}
                      currentValue={currentValues.contentRating}
                      incomingValue={incomingValues.contentRating}
                      selectedValue={selectedContentRating}
                      onSelect={onSelectContentRating}
                      textTransform="capitalize"
                    />
                  )}

                  {/* Country */}
                  {(incomingValues.country || !diffOnly) && (
                    <MetadataFieldRow
                      label="Country"
                      icon={<Globe className="size-3.5" />}
                      currentValue={currentValues.country}
                      incomingValue={incomingValues.country}
                      selectedValue={selectedCountry}
                      onSelect={onSelectCountry}
                      textTransform="uppercase"
                    />
                  )}

                  {/* Reading Mode */}
                  {(incomingValues.readingMode || !diffOnly) && (
                    <MetadataFieldRow
                      label="Reading Mode"
                      icon={<BookOpen className="size-3.5" />}
                      currentValue={currentValues.readingMode}
                      incomingValue={incomingValues.readingMode}
                      selectedValue={selectedReadingMode}
                      onSelect={onSelectReadingMode}
                      textTransform="uppercase"
                    />
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
          onClick={onBack}
          className="text-xs cursor-pointer"
        >
          {mode === 'merge' ? 'Cancel' : 'Back'}
        </Button>

        <Button
          type="button"
          onClick={onImport}
          disabled={isImporting}
          className="gap-2 text-xs cursor-pointer"
        >
          {isImporting ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Download className="size-3.5" />
          )}
          {mode === 'merge' ? 'Merge' : 'Import & Bind Provider'}
        </Button>
      </div>
    </div>
  );
};
