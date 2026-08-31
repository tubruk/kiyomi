import React, { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { ChevronDown, ChevronUp, Plus } from 'lucide-react';
import { Button } from '../components/ui/button';
import { ChapterList } from '../components/ChapterList';
import { ProviderList } from '../components/ProviderList';
import { EditMetadataDialog } from '../components/EditMetadataDialog';
import { AddProviderDialog } from '../components/AddProviderDialog';
import { ImportMetadataDialog } from '../components/ImportMetadataDialog';
import {
  useDetailsManga,
  useChapterOperations,
  DetailsHeroCard,
  DetailsUserMetadata,
  DetailsActionBar,
} from '../components/details';

export const DetailsPage: React.FC = () => {
  const navigate = useNavigate();

  const [sortBy, setSortBy] = useState('source');
  const [order, setOrder] = useState<'asc' | 'desc'>('desc');

  const [isEditMetadataOpen, setIsEditMetadataOpen] = useState(false);
  const [isAddProviderOpen, setIsAddProviderOpen] = useState(false);
  const [isImportMetadataOpen, setIsImportMetadataOpen] = useState(false);
  const [importMetadataProviderId, setImportMetadataProviderId] = useState<string | undefined>(undefined);
  const [importMetadataRemoteId, setImportMetadataRemoteId] = useState<string | undefined>(undefined);
  const [isProvidersCollapsed, setIsProvidersCollapsed] = useState(true);

  // Manga details data & route resolution
  const {
    isRemoteRoute,
    providerIdParam,
    remoteIdParam,
    targetMangaId,
    isInLibrary,
    manga,
    isMangaLoading,
    sources,
    activeContentProviderId,
    activeProvider,
    contentProviderName,
    chapters,
    isChaptersLoading,
    isChaptersError,
    mangaJobs,
    activeJobChapterIds,
    readingCta,
    isUnavailable,
    hasZeroChapters,
  } = useDetailsManga();

  // Chapter & Library operations
  const {
    pullingChapterIds,
    deletingFilesChapterIds,
    removingChapterIds,
    addToLibraryMutation,
    deleteLibraryMutation,
    updateLibraryMangaMutation,
    refreshChaptersMutation,
    removeChapterMutation,
    deleteChapterFilesMutation,
    pullChapterMutation,
    removeProviderMutation,
    switchToMutation,
    batchUpdateProgressMutation,
    batchPullMutation,
    batchRefreshChaptersMutation,
    batchDeleteFilesMutation,
    batchDeleteChaptersMutation,
    handlePullChapter,
    handleUserMetadataChange,
    handleBatchUpdateProgress,
    handleBatchPull,
    handleBatchRefresh,
    handleBatchDeleteFiles,
    handleBatchRemove,
  } = useChapterOperations({
    targetMangaId,
    activeContentProviderId,
    providerIdParam,
    remoteIdParam,
    manga,
    chapters,
    mangaJobs,
    activeJobChapterIds,
  });

  const handlePrimaryCta = () => {
    if (!readingCta) return;
    if (isRemoteRoute && !isInLibrary) {
      navigate({
        to: '/providers/$providerId/manga/$remoteId/chapter/$chapterId',
        params: {
          providerId: providerIdParam || manga?.sourceId || 'mangafox',
          remoteId: remoteIdParam || manga?.contentRemoteId || manga?.id || '',
          chapterId: readingCta.chapterId,
        },
        search: readingCta.page > 1 ? { page: readingCta.page } : {},
      });
    } else {
      navigate({
        to: '/manga/$mangaId/chapter/$chapterId',
        params: {
          mangaId: targetMangaId || manga?.id || '',
          chapterId: readingCta.chapterId,
        },
        search: readingCta.page > 1 ? { page: readingCta.page } : {},
      });
    }
  };

  const handleConfirmRemoveFromLibrary = () => {
    if (confirm(`Are you sure you want to remove "${manga?.title || 'this series'}" from your library?`)) {
      if (isInLibrary && targetMangaId) {
        deleteLibraryMutation.mutate(targetMangaId, {
          onSuccess: () => {
            navigate({ to: '/' });
          },
        });
      }
    }
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Top Action Bar */}
      <DetailsActionBar
        isRemoteRoute={isRemoteRoute}
        providerIdParam={providerIdParam}
        isInLibrary={isInLibrary}
        targetMangaId={targetMangaId}
        manga={manga}
        hasZeroChapters={hasZeroChapters}
        isUnavailable={isUnavailable}
        hasChapters={chapters.length > 0}
        readingCta={readingCta}
        isAddingToLibrary={addToLibraryMutation.isPending}
        onAddToLibrary={() => addToLibraryMutation.mutate()}
        onPrimaryCta={handlePrimaryCta}
        onOpenImportMetadata={() => {
          setImportMetadataProviderId(undefined);
          setImportMetadataRemoteId(undefined);
          setIsImportMetadataOpen(true);
        }}
        onOpenEditMetadata={() => setIsEditMetadataOpen(true)}
        onRemoveFromLibrary={handleConfirmRemoveFromLibrary}
      />

      {/* Hero Card */}
      <DetailsHeroCard
        manga={manga}
        isMangaLoading={isMangaLoading}
        contentProviderName={contentProviderName}
        userMetadataSlot={
          !isRemoteRoute && isInLibrary && manga ? (
            <DetailsUserMetadata
              manga={manga}
              isUpdating={updateLibraryMangaMutation.isPending}
              onUpdate={handleUserMetadataChange}
            />
          ) : null
        }
      />

      {/* Provider Bindings Section */}
      {!isRemoteRoute && isInLibrary && manga && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <button
              onClick={() => setIsProvidersCollapsed(!isProvidersCollapsed)}
              className="flex items-center gap-1.5 hover:opacity-80 cursor-pointer"
            >
              <h2 className="text-sm font-semibold text-foreground">Providers</h2>
              {isProvidersCollapsed ? (
                <ChevronDown className="size-4 text-muted-foreground" />
              ) : (
                <ChevronUp className="size-4 text-muted-foreground" />
              )}
            </button>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsAddProviderOpen(true)}
                className="gap-1.5 h-8 text-xs cursor-pointer"
              >
                <Plus className="size-3" />
                Add provider
              </Button>
            </div>
          </div>

          {!isProvidersCollapsed && (
            <ProviderList
              providers={manga.meta?.providers || []}
              contentProviderId={manga.contentProviderId || manga.sourceId || manga.meta?.content?.provider_id}
              contentProviderMangaId={manga.contentRemoteId || manga.meta?.content?.provider_manga_id}
              sources={sources}
              onImportMetadata={(provider) => {
                setImportMetadataProviderId(provider.provider_id);
                setImportMetadataRemoteId(provider.provider_manga_id);
                setIsImportMetadataOpen(true);
              }}
              onRemove={(provider) => {
                if (confirm(`Remove "${provider.manga_title || provider.provider_id}" from this manga?`)) {
                  removeProviderMutation.mutate(provider);
                }
              }}
              onSwitchTo={(provider) => {
                if (
                  confirm(
                    `Switch the default content provider to "${provider.manga_title || provider.provider_id}"? Chapters from the previous provider remain accessible — nothing is deleted.`
                  )
                ) {
                  switchToMutation.mutate({ provider });
                }
              }}
              isRemoving={removeProviderMutation.isPending}
              canRemoveProvider={(provider) => {
                const hasContentCapability = sources.find((s) => s.id === provider.provider_id)?.capabilities?.includes('content');
                if (!hasContentCapability) return true;
                const contentProviders = (manga.meta?.providers || []).filter((p) =>
                  sources.find((s) => s.id === p.provider_id)?.capabilities?.includes('content')
                );
                return contentProviders.length > 1;
              }}
              isContentUnavailable={isUnavailable}
            />
          )}
        </div>
      )}

      {/* Chapters Section using ChapterList */}
      {manga && (
        <ChapterList
          chapters={chapters}
          mangaId={manga.id}
          providerId={isRemoteRoute ? providerIdParam : undefined}
          remoteId={isRemoteRoute ? remoteIdParam : undefined}
          sortBy={sortBy}
          order={order}
          onSortByChange={setSortBy}
          onOrderToggle={() => setOrder((prev) => (prev === 'desc' ? 'asc' : 'desc'))}
          isLoading={isChaptersLoading}
          isError={isChaptersError}
          contentProviderName={contentProviderName}
          providerName={activeProvider?.name || activeContentProviderId}
          isUnavailable={isUnavailable}
          isInLibrary={isInLibrary}
          onRefreshChapters={isInLibrary ? () => refreshChaptersMutation.mutate() : undefined}
          isRefreshing={refreshChaptersMutation.isPending}
          onRemoveChapter={
            isInLibrary
              ? (chId, provId) => provId && removeChapterMutation.mutate({ chapterId: chId, providerId: provId })
              : undefined
          }
          onDeleteFiles={
            isInLibrary
              ? (chId, provId) => deleteChapterFilesMutation.mutate({ chapterId: chId, providerId: provId })
              : undefined
          }
          onPullChapter={isInLibrary ? handlePullChapter : undefined}
          isDeletingFiles={deleteChapterFilesMutation.isPending}
          isPullingChapter={pullChapterMutation.isPending}
          isRemovingChapter={removeChapterMutation.isPending}
          pullingChapterIds={pullingChapterIds}
          deletingFilesChapterIds={deletingFilesChapterIds}
          removingChapterIds={removingChapterIds}
          onBatchUpdateProgress={isInLibrary && activeContentProviderId ? handleBatchUpdateProgress : undefined}
          isBatchUpdatingProgress={batchUpdateProgressMutation.isPending}
          onBatchPull={isInLibrary && activeContentProviderId ? handleBatchPull : undefined}
          isBatchPulling={batchPullMutation.isPending}
          onBatchRefresh={isInLibrary && activeContentProviderId ? handleBatchRefresh : undefined}
          isBatchRefreshing={batchRefreshChaptersMutation.isPending}
          onBatchDeleteFiles={isInLibrary && activeContentProviderId ? handleBatchDeleteFiles : undefined}
          isBatchDeletingFiles={batchDeleteFilesMutation.isPending}
          onBatchRemove={isInLibrary && activeContentProviderId ? handleBatchRemove : undefined}
          isBatchRemoving={batchDeleteChaptersMutation.isPending}
        />
      )}

      {/* Dialogs */}
      {manga && (
        <EditMetadataDialog
          manga={manga}
          open={isEditMetadataOpen}
          onOpenChange={setIsEditMetadataOpen}
        />
      )}

      {manga && (
        <AddProviderDialog
          mangaId={manga.id}
          sources={sources}
          existingProviders={manga.meta?.providers || []}
          mangaTitle={manga.title}
          mangaAliases={manga.aliases || manga.meta?.aliases || []}
          open={isAddProviderOpen}
          onOpenChange={setIsAddProviderOpen}
        />
      )}

      {manga && (
        <ImportMetadataDialog
          manga={manga}
          sources={sources}
          open={isImportMetadataOpen}
          onOpenChange={(open) => {
            setIsImportMetadataOpen(open);
            if (!open) {
              setImportMetadataProviderId(undefined);
              setImportMetadataRemoteId(undefined);
            }
          }}
          initialProviderId={importMetadataProviderId}
          initialRemoteId={importMetadataRemoteId}
        />
      )}
    </div>
  );
};
