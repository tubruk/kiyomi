import React, { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
} from '../ui/dialog';
import { cn } from '../../lib/utils';
import { ImportMetadataDialogProps, DialogStep } from './types';
import { useMetadataSearch } from './hooks/useMetadataSearch';
import { useMetadataComparison } from './hooks/useMetadataComparison';
import { MetadataSearchStep } from './steps/MetadataSearchStep';
import { MetadataCompareStep } from './steps/MetadataCompareStep';

export const ImportMetadataDialog: React.FC<ImportMetadataDialogProps> = ({
  manga,
  sources,
  open,
  onOpenChange,
  initialProviderId,
  initialRemoteId,
  onSuccess,
  mode = 'import',
  incomingManga,
  sourceMangaIds,
}) => {
  const [step, setStep] = useState<DialogStep>(mode === 'merge' ? 'compare' : 'search');

  const handleClose = () => {
    setStep(mode === 'merge' ? 'compare' : 'search');
    searchState.resetSearchState();
    comparisonState.resetComparisonState();
    onOpenChange(false);
  };

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      handleClose();
    } else {
      onOpenChange(isOpen);
    }
  };

  const searchState = useMetadataSearch({
    manga,
    sources,
    open: mode === 'merge' ? false : open,
    initialProviderId,
    initialRemoteId,
    onSelectRemoteManga: (remote) => {
      comparisonState.initComparisonState(remote);
      setStep('compare');
    },
    onErrorFallbackToSearch: () => {
      setStep('search');
    },
  });

  const comparisonState = useMetadataComparison({
    manga,
    sources,
    selectedProviderId: searchState.selectedProviderId,
    onSuccess,
    onClose: handleClose,
    mode,
    incomingManga,
    sourceMangaIds,
  });

  useEffect(() => {
    if (open) {
      if (mode === 'merge') {
        setStep('compare');
      } else if (!initialProviderId || !initialRemoteId) {
        setStep('search');
      }
    }
  }, [open, initialProviderId, initialRemoteId, mode]);

  // In merge mode we render only the comparison step (no search step, no back action).
  if (mode === 'merge') {
    return (
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent
          className={cn(
            'sm:max-w-4xl md:max-w-4xl lg:max-w-5xl w-[95vw] max-h-[90vh] h-[85vh] p-0 overflow-hidden flex flex-col'
          )}
        >
          <MetadataCompareStep
            manga={manga}
            selectedRemoteManga={comparisonState.selectedRemoteManga}
            selectedProvider={searchState.selectedProvider}
            diffOnly={comparisonState.diffOnly}
            onSetDiffOnly={comparisonState.setDiffOnly}
            onKeepCurrent={comparisonState.handleKeepCurrent}
            onAcceptIncoming={comparisonState.handleAcceptIncoming}
            onBack={handleClose}
            onImport={() => comparisonState.importMutation.mutate()}
            isImporting={comparisonState.importMutation.isPending}
            currentValues={comparisonState.currentValues}
            incomingValues={comparisonState.incomingValues}
            diffs={comparisonState.diffs}
            mergedExternalLinks={comparisonState.mergedExternalLinks}
            selectedTitle={comparisonState.selectedTitle}
            onSelectTitle={comparisonState.setSelectedTitle}
            selectedCover={comparisonState.selectedCover}
            onSelectCover={comparisonState.setSelectedCover}
            selectedDescription={comparisonState.selectedDescription}
            onSelectDescription={comparisonState.setSelectedDescription}
            selectedAuthorsMode={comparisonState.selectedAuthorsMode}
            onSelectAuthorsMode={comparisonState.setSelectedAuthorsMode}
            selectedArtistsMode={comparisonState.selectedArtistsMode}
            onSelectArtistsMode={comparisonState.setSelectedArtistsMode}
            selectedPublishersMode={comparisonState.selectedPublishersMode}
            onSelectPublishersMode={comparisonState.setSelectedPublishersMode}
            selectedExternalLinksMode={comparisonState.selectedExternalLinksMode}
            onSelectExternalLinksMode={comparisonState.setSelectedExternalLinksMode}
            selectedProvidersMode={comparisonState.selectedProvidersMode}
            onSelectProvidersMode={comparisonState.setSelectedProvidersMode}
            selectedTags={comparisonState.selectedTags}
            unselectedTags={comparisonState.unselectedTags}
            tagInput={comparisonState.tagInput}
            onTagInputChange={comparisonState.handleTagInputChange}
            onTagInputKeyDown={comparisonState.handleTagInputKeyDown}
            onAddTag={comparisonState.addTag}
            onRemoveTag={comparisonState.removeTag}
            onSetSelectedTags={comparisonState.setSelectedTags}
            selectedAliases={comparisonState.selectedAliases}
            unselectedAliases={comparisonState.unselectedAliases}
            aliasInput={comparisonState.aliasInput}
            onAliasInputChange={comparisonState.handleAliasInputChange}
            onAliasInputKeyDown={comparisonState.handleAliasInputKeyDown}
            onAddAlias={comparisonState.addAlias}
            onRemoveAlias={comparisonState.removeAlias}
            selectedReleaseYear={comparisonState.selectedReleaseYear}
            onSelectReleaseYear={comparisonState.setSelectedReleaseYear}
            selectedStartDate={comparisonState.selectedStartDate}
            onSelectStartDate={comparisonState.setSelectedStartDate}
            selectedEndDate={comparisonState.selectedEndDate}
            onSelectEndDate={comparisonState.setSelectedEndDate}
            selectedContentRating={comparisonState.selectedContentRating}
            onSelectContentRating={comparisonState.setSelectedContentRating}
            selectedCountry={comparisonState.selectedCountry}
            onSelectCountry={comparisonState.setSelectedCountry}
            selectedReadingMode={comparisonState.selectedReadingMode}
            onSelectReadingMode={comparisonState.setSelectedReadingMode}
            mode={mode}
          />
        </DialogContent>
      </Dialog>
    );
  }

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
        {searchState.isLoadingDetails && !comparisonState.selectedRemoteManga ? (
          <div className="flex flex-col items-center justify-center py-20 space-y-3">
            <Loader2 className="size-8 animate-spin text-primary" />
            <p className="text-xs font-medium text-muted-foreground">
              Fetching metadata from {searchState.selectedProvider?.name || searchState.selectedProviderId}...
            </p>
          </div>
        ) : step === 'search' ? (
          <MetadataSearchStep
            manga={manga}
            sources={sources}
            boundProviders={searchState.boundProviders}
            selectedProviderId={searchState.selectedProviderId}
            onSelectProviderId={searchState.setSelectedProviderId}
            selectedProvider={searchState.selectedProvider}
            searchMode={searchState.searchMode}
            onSelectSearchMode={searchState.setSearchMode}
            searchQuery={searchState.searchQuery}
            onSearchQueryChange={searchState.setSearchQuery}
            directIdOrUrl={searchState.directIdOrUrl}
            onDirectIdOrUrlChange={searchState.setDirectIdOrUrl}
            isSearching={searchState.searchMutation.isPending}
            isLookingUp={searchState.directLookupMutation.isPending}
            isLoadingDetails={searchState.isLoadingDetails}
            searchResults={searchState.searchResults}
            searchError={searchState.searchError}
            onSearch={searchState.handleSearch}
            onDirectLookup={searchState.handleDirectLookup}
            onSelectSearchResult={searchState.handleSelectSearchResult}
            onCancel={handleClose}
          />
        ) : (
          <MetadataCompareStep
            manga={manga}
            selectedRemoteManga={comparisonState.selectedRemoteManga}
            selectedProvider={searchState.selectedProvider}
            diffOnly={comparisonState.diffOnly}
            onSetDiffOnly={comparisonState.setDiffOnly}
            onKeepCurrent={comparisonState.handleKeepCurrent}
            onAcceptIncoming={comparisonState.handleAcceptIncoming}
            onBack={() => setStep('search')}
            onImport={() => comparisonState.importMutation.mutate()}
            isImporting={comparisonState.importMutation.isPending}
            currentValues={comparisonState.currentValues}
            incomingValues={comparisonState.incomingValues}
            diffs={comparisonState.diffs}
            mergedExternalLinks={comparisonState.mergedExternalLinks}
            selectedTitle={comparisonState.selectedTitle}
            onSelectTitle={comparisonState.setSelectedTitle}
            selectedCover={comparisonState.selectedCover}
            onSelectCover={comparisonState.setSelectedCover}
            selectedDescription={comparisonState.selectedDescription}
            onSelectDescription={comparisonState.setSelectedDescription}
            selectedAuthorsMode={comparisonState.selectedAuthorsMode}
            onSelectAuthorsMode={comparisonState.setSelectedAuthorsMode}
            selectedArtistsMode={comparisonState.selectedArtistsMode}
            onSelectArtistsMode={comparisonState.setSelectedArtistsMode}
            selectedPublishersMode={comparisonState.selectedPublishersMode}
            onSelectPublishersMode={comparisonState.setSelectedPublishersMode}
            selectedExternalLinksMode={comparisonState.selectedExternalLinksMode}
            onSelectExternalLinksMode={comparisonState.setSelectedExternalLinksMode}
            selectedProvidersMode={comparisonState.selectedProvidersMode}
            onSelectProvidersMode={comparisonState.setSelectedProvidersMode}
            selectedTags={comparisonState.selectedTags}
            unselectedTags={comparisonState.unselectedTags}
            tagInput={comparisonState.tagInput}
            onTagInputChange={comparisonState.handleTagInputChange}
            onTagInputKeyDown={comparisonState.handleTagInputKeyDown}
            onAddTag={comparisonState.addTag}
            onRemoveTag={comparisonState.removeTag}
            onSetSelectedTags={comparisonState.setSelectedTags}
            selectedAliases={comparisonState.selectedAliases}
            unselectedAliases={comparisonState.unselectedAliases}
            aliasInput={comparisonState.aliasInput}
            onAliasInputChange={comparisonState.handleAliasInputChange}
            onAliasInputKeyDown={comparisonState.handleAliasInputKeyDown}
            onAddAlias={comparisonState.addAlias}
            onRemoveAlias={comparisonState.removeAlias}
            selectedReleaseYear={comparisonState.selectedReleaseYear}
            onSelectReleaseYear={comparisonState.setSelectedReleaseYear}
            selectedStartDate={comparisonState.selectedStartDate}
            onSelectStartDate={comparisonState.setSelectedStartDate}
            selectedEndDate={comparisonState.selectedEndDate}
            onSelectEndDate={comparisonState.setSelectedEndDate}
            selectedContentRating={comparisonState.selectedContentRating}
            onSelectContentRating={comparisonState.setSelectedContentRating}
            selectedCountry={comparisonState.selectedCountry}
            onSelectCountry={comparisonState.setSelectedCountry}
            selectedReadingMode={comparisonState.selectedReadingMode}
            onSelectReadingMode={comparisonState.setSelectedReadingMode}
            mode={mode}
          />
        )}
      </DialogContent>
    </Dialog>
  );
};
