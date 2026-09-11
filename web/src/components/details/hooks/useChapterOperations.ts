import { useState, useEffect, useMemo, useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../../api/client';
import {
  useUpdateLibraryMangaMutation,
  useDeleteLibraryMangaMutation,
  useBatchUpdateChapterProgressMutation,
  useBatchPullChaptersMutation,
  useBatchRefreshChaptersMutation,
  useBatchDeleteChapterFilesMutation,
  useBatchDeleteChaptersMutation,
} from '../../../api/hooks';
import { useToast } from '../../../context/ToastContext';
import { queryKeys } from '../../../lib/queryKeys';
import { Manga, Chapter, Job, ProviderRef } from '../../../types/api';

export interface UseChapterOperationsOptions {
  targetMangaId: string;
  activeContentProviderId?: string;
  providerIdParam?: string;
  remoteIdParam?: string;
  manga?: Manga;
  chapters?: Chapter[];
  mangaJobs?: Job[];
  activeJobChapterIds?: string[];
}

export function useChapterOperations({
  targetMangaId,
  activeContentProviderId,
  providerIdParam = '',
  remoteIdParam = '',
  manga,
  chapters = [],
  mangaJobs = [],
  activeJobChapterIds = [],
}: UseChapterOperationsOptions) {
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const [optimisticPullIds, setOptimisticPullIds] = useState<Set<string>>(new Set());

  // Cleanup optimistic pull IDs when chapter is downloaded or background job finishes
  useEffect(() => {
    setOptimisticPullIds((prev) => {
      if (prev.size === 0) return prev;
      let changed = false;
      const next = new Set(prev);
      for (const id of prev) {
        const ch = chapters.find((c) => c.id === id);
        const isDownloaded = Boolean(
          ch?.is_downloaded ?? (ch as any)?.isDownloaded ?? ch?.meta?.is_downloaded
        );
        const job = mangaJobs.find(
          (j) => j.type === 'pull_chapter' && j.metadata?.chapter_id === id
        );
        const isJobFinished = job && (job.status === 'completed' || job.status === 'failed');
        if (isDownloaded || isJobFinished) {
          next.delete(id);
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [chapters, mangaJobs]);

  // Mutations
  const addToLibraryMutation = useMutation({
    mutationFn: async () => {
      if (!manga) return null;
      const created = await api.importProviderManga(
        providerIdParam || manga.sourceId || 'mangafox',
        remoteIdParam || manga.contentRemoteId || manga.id,
        'plan_to_read'
      );
      return created;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.all });
      showToast('Series added to library', 'success');
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Failed to add to library: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const deleteLibraryMutation = useDeleteLibraryMangaMutation();
  const updateLibraryMangaMutation = useUpdateLibraryMangaMutation();

  const refreshChaptersMutation = useMutation({
    mutationFn: () => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.refreshLibraryManga(targetMangaId);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(targetMangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(targetMangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.refresh(targetMangaId) });
      if (data.added === 0) {
        showToast('Up to date', 'success');
      } else {
        showToast(`Refresh complete: ${data.added} chapter(s)`, 'success');
      }
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Refresh failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const removeChapterMutation = useMutation({
    mutationFn: ({ chapterId, providerId }: { chapterId: string; providerId: string }) => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.deleteChapter(targetMangaId, chapterId, providerId);
    },
    onSuccess: (_, { providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(targetMangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(targetMangaId, providerId),
        });
      }
      showToast('Chapter removed from library', 'success');
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Remove failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const deleteChapterFilesMutation = useMutation({
    mutationFn: ({ chapterId, providerId }: { chapterId: string; providerId: string }) => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.deleteChapterFiles(targetMangaId, providerId, chapterId);
    },
    onSuccess: (_, { providerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(targetMangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(targetMangaId, providerId),
        });
      }
      showToast('Files deleted (chapter entry preserved)', 'success');
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Delete files failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const pullChapterMutation = useMutation({
    mutationFn: ({ chapterId, providerId }: { chapterId: string; providerId: string }) => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.pullChapter(targetMangaId, providerId, chapterId);
    },
    onSuccess: (_, { chapterId, providerId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.chapters.pages(chapterId, targetMangaId, providerId),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(targetMangaId) });
      if (providerId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(targetMangaId, providerId),
        });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
      showToast('Chapter pulled from provider', 'success');
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Pull failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const handlePullChapter = useCallback(
    (chapterId: string, providerId: string) => {
      setOptimisticPullIds((prev) => {
        const next = new Set(prev);
        next.add(chapterId);
        return next;
      });
      pullChapterMutation.mutate({ chapterId, providerId });
    },
    [pullChapterMutation]
  );

  const removeProviderMutation = useMutation({
    mutationFn: (provider: ProviderRef) => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.removeBinding(targetMangaId, provider.provider_id, provider.provider_manga_id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.bindings(targetMangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(targetMangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      showToast('Provider removed', 'success');
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Remove failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const switchToMutation = useMutation({
    mutationFn: ({ provider }: { provider: ProviderRef }) => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.setActiveContentSource(targetMangaId, {
        provider_id: provider.provider_id,
        provider_manga_id: provider.provider_manga_id,
      });
    },
    onSuccess: (_, { provider }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.bindings(targetMangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(targetMangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(targetMangaId) });
      if (provider?.provider_id) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.chapters.providerList(targetMangaId, provider.provider_id),
        });
      }
      showToast('Switched content provider', 'success');
    },
    onError: (err: any) => {
      const detail = err.details
        ? typeof err.details === 'string'
          ? err.details
          : JSON.stringify(err.details, null, 2)
        : err.stack || String(err);
      showToast(`Switch failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  // Batch Mutations
  const batchUpdateProgressMutation = useBatchUpdateChapterProgressMutation();
  const batchPullMutation = useBatchPullChaptersMutation();
  const batchRefreshChaptersMutation = useBatchRefreshChaptersMutation();
  const batchDeleteFilesMutation = useBatchDeleteChapterFilesMutation();
  const batchDeleteChaptersMutation = useBatchDeleteChaptersMutation();

  const handleBatchUpdateProgress = useCallback(
    (chapterIds: string[], progress: { is_read?: boolean; last_read_page?: number }) => {
      if (!targetMangaId || !activeContentProviderId) return;
      batchUpdateProgressMutation.mutate(
        {
          mangaId: targetMangaId,
          providerId: activeContentProviderId,
          chapterIds,
          progress,
        },
        {
          onSuccess: (data) => {
            const count = data?.updated ?? chapterIds.length;
            showToast(
              progress.is_read
                ? `Marked ${count} chapter(s) as read`
                : `Marked ${count} chapter(s) as unread`,
              'success'
            );
          },
          onError: (err: any) => {
            const detail = err.details
              ? typeof err.details === 'string'
                ? err.details
                : JSON.stringify(err.details, null, 2)
              : err.stack || String(err);
            showToast(
              `Failed to update chapter progress: ${err.message || 'An error occurred'}`,
              'error',
              detail
            );
          },
        }
      );
    },
    [targetMangaId, activeContentProviderId, batchUpdateProgressMutation, showToast]
  );

  const handleBatchPull = useCallback(
    (chapterIds: string[]) => {
      if (!targetMangaId || !activeContentProviderId) return;
      setOptimisticPullIds((prev) => {
        const next = new Set(prev);
        chapterIds.forEach((id) => next.add(id));
        return next;
      });
      batchPullMutation.mutate(
        {
          mangaId: targetMangaId,
          providerId: activeContentProviderId,
          chapterIds,
        },
        {
          onSuccess: (data) => {
            const count = data?.chapter_count ?? chapterIds.length;
            queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
            showToast(`Pull enqueued for ${count} chapter(s)`, 'success');
          },
          onError: (err: any) => {
            const detail = err.details
              ? typeof err.details === 'string'
                ? err.details
                : JSON.stringify(err.details, null, 2)
              : err.stack || String(err);
            showToast(`Failed to pull chapters: ${err.message || 'An error occurred'}`, 'error', detail);
          },
        }
      );
    },
    [targetMangaId, activeContentProviderId, batchPullMutation, queryClient, showToast]
  );

  const handleBatchRefresh = useCallback(
    (chapterIds: string[]) => {
      if (!targetMangaId || !activeContentProviderId) return;
      batchRefreshChaptersMutation.mutate(
        {
          mangaId: targetMangaId,
          providerId: activeContentProviderId,
          chapterIds,
        },
        {
          onSuccess: (data) => {
            const count = data?.refreshed || chapterIds.length;
            showToast(`Refreshed metadata for ${count} chapter(s)`, 'success');
          },
          onError: (err: any) => {
            const detail = err.details
              ? typeof err.details === 'string'
                ? err.details
                : JSON.stringify(err.details, null, 2)
              : err.stack || String(err);
            showToast(
              `Failed to refresh chapter metadata: ${err.message || 'An error occurred'}`,
              'error',
              detail
            );
          },
        }
      );
    },
    [targetMangaId, activeContentProviderId, batchRefreshChaptersMutation, showToast]
  );

  const handleBatchDeleteFiles = useCallback(
    (chapterIds: string[]) => {
      if (!targetMangaId || !activeContentProviderId) return;
      batchDeleteFilesMutation.mutate(
        {
          mangaId: targetMangaId,
          providerId: activeContentProviderId,
          chapterIds,
        },
        {
          onSuccess: (data) => {
            const count = data?.deleted ?? chapterIds.length;
            showToast(`Deleted files for ${count} chapter(s)`, 'success');
          },
          onError: (err: any) => {
            const detail = err.details
              ? typeof err.details === 'string'
                ? err.details
                : JSON.stringify(err.details, null, 2)
              : err.stack || String(err);
            showToast(`Failed to delete files: ${err.message || 'An error occurred'}`, 'error', detail);
          },
        }
      );
    },
    [targetMangaId, activeContentProviderId, batchDeleteFilesMutation, showToast]
  );

  const handleBatchRemove = useCallback(
    (chapterIds: string[]) => {
      if (!targetMangaId || !activeContentProviderId) return;
      batchDeleteChaptersMutation.mutate(
        {
          mangaId: targetMangaId,
          providerId: activeContentProviderId,
          chapterIds,
        },
        {
          onSuccess: (data) => {
            const count = data?.removed ?? chapterIds.length;
            showToast(`Removed ${count} chapter(s) from library`, 'success');
          },
          onError: (err: any) => {
            const detail = err.details
              ? typeof err.details === 'string'
                ? err.details
                : JSON.stringify(err.details, null, 2)
              : err.stack || String(err);
            showToast(`Failed to remove chapters: ${err.message || 'An error occurred'}`, 'error', detail);
          },
        }
      );
    },
    [targetMangaId, activeContentProviderId, batchDeleteChaptersMutation, showToast]
  );

  const handleUserMetadataChange = useCallback(
    (updatedFields: Partial<Manga>) => {
      if (!targetMangaId) return;
      updateLibraryMangaMutation.mutate(
        { mangaId: targetMangaId, fields: updatedFields },
        {
          onError: (err: any) => {
            const detail = err.details
              ? typeof err.details === 'string'
                ? err.details
                : JSON.stringify(err.details, null, 2)
              : err.stack || String(err);
            showToast(`Failed to update metadata: ${err.message || 'An error occurred'}`, 'error', detail);
          },
        }
      );
    },
    [targetMangaId, updateLibraryMangaMutation, showToast]
  );

  const pullingChapterIds = useMemo(() => {
    const set = new Set<string>(activeJobChapterIds);
    optimisticPullIds.forEach((id) => {
      const ch = chapters.find((c) => c.id === id);
      const isDownloaded = Boolean(
        ch?.is_downloaded ?? (ch as any)?.isDownloaded ?? ch?.meta?.is_downloaded
      );
      if (!isDownloaded) {
        set.add(id);
      }
    });
    if (pullChapterMutation.isPending && pullChapterMutation.variables?.chapterId) {
      set.add(pullChapterMutation.variables.chapterId);
    }
    if (batchPullMutation.isPending && batchPullMutation.variables?.chapterIds) {
      batchPullMutation.variables.chapterIds.forEach((id) => set.add(id));
    }
    return Array.from(set);
  }, [
    activeJobChapterIds,
    optimisticPullIds,
    chapters,
    pullChapterMutation.isPending,
    pullChapterMutation.variables,
    batchPullMutation.isPending,
    batchPullMutation.variables,
  ]);

  const deletingFilesChapterIds = useMemo(() => {
    const ids: string[] = [];
    if (deleteChapterFilesMutation.isPending && deleteChapterFilesMutation.variables?.chapterId) {
      ids.push(deleteChapterFilesMutation.variables.chapterId);
    }
    if (batchDeleteFilesMutation.isPending && batchDeleteFilesMutation.variables?.chapterIds) {
      ids.push(...batchDeleteFilesMutation.variables.chapterIds);
    }
    return ids;
  }, [
    deleteChapterFilesMutation.isPending,
    deleteChapterFilesMutation.variables,
    batchDeleteFilesMutation.isPending,
    batchDeleteFilesMutation.variables,
  ]);

  const removingChapterIds = useMemo(() => {
    const ids: string[] = [];
    if (removeChapterMutation.isPending && removeChapterMutation.variables?.chapterId) {
      ids.push(removeChapterMutation.variables.chapterId);
    }
    if (batchDeleteChaptersMutation.isPending && batchDeleteChaptersMutation.variables?.chapterIds) {
      ids.push(...batchDeleteChaptersMutation.variables.chapterIds);
    }
    return ids;
  }, [
    removeChapterMutation.isPending,
    removeChapterMutation.variables,
    batchDeleteChaptersMutation.isPending,
    batchDeleteChaptersMutation.variables,
  ]);

  return {
    optimisticPullIds,
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
  };
}
