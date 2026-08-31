import React, { useState } from 'react';
import {
  HardDrive,
  RefreshCw,
  Image as ImageIcon,
  Info,
  Database,
  Trash2,
} from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { cacheStatsQueryOptions } from '../../lib/queryOptions';
import { queryKeys } from '../../lib/queryKeys';
import { api } from '../../api/client';
import { useToast } from '../../context/ToastContext';
import { clearQueryPersistence } from '../../lib/persister';
import { formatBytes } from '../../lib/utils';
import { Button } from '../ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from '../ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../ui/dialog';
import { ErrorDetailsModal } from '../ErrorDetailsModal';

export const SettingsCacheTab: React.FC = () => {
  const { showToast } = useToast();
  const queryClient = useQueryClient();

  // Clear cache confirmation modals
  const [confirmClearCacheOpen, setConfirmClearCacheOpen] = useState(false);
  const [confirmClearOfflineCacheOpen, setConfirmClearOfflineCacheOpen] = useState(false);
  const [isClearingOfflineCache, setIsClearingOfflineCache] = useState(false);

  // Error modal state
  const [errorModalOpen, setErrorModalOpen] = useState(false);
  const [errorTitle, setErrorTitle] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [errorDetails, setErrorDetails] = useState('');

  const {
    data: cacheStats,
    isLoading: isLoadingCacheStats,
    refetch: refetchCacheStats,
    isRefetching: isRefetchingCacheStats,
  } = useQuery({
    ...cacheStatsQueryOptions(),
  });

  // Clear cache mutation
  const clearCacheMutation = useMutation({
    mutationFn: () => api.clearCache(),
    onSuccess: () => {
      showToast('Image cache cleared successfully', 'success');
      queryClient.invalidateQueries({ queryKey: queryKeys.system.cache });
    },
    onError: (err: any) => {
      console.error('Failed to clear cache:', err);
      const msg = err.message || 'Failed to clear cache';
      const details = err.details
        ? typeof err.details === 'object'
          ? JSON.stringify(err.details, null, 2)
          : String(err.details)
        : err.stack || '';

      showToast(msg, 'error', details);
      setErrorTitle('Failed to clear image cache');
      setErrorMessage(msg);
      setErrorDetails(details);
      setErrorModalOpen(true);
    },
  });

  return (
    <div className="space-y-6">
      {/* Tab action row */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-bold text-foreground flex items-center gap-2">
            <HardDrive className="size-4 text-primary" />
            Image Cache
          </h2>
          <p className="text-xs text-muted-foreground">
            Local disk cache for manga covers and chapter page images.
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => refetchCacheStats()}
          disabled={isLoadingCacheStats || isRefetchingCacheStats}
          className="text-xs font-semibold cursor-pointer gap-2"
        >
          <RefreshCw className={`size-3.5 ${isRefetchingCacheStats ? 'animate-spin' : ''}`} />
          <span>Refresh</span>
        </Button>
      </div>

      {/* Cache Storage Card */}
      <div className="max-w-2xl">
        <Card className="border border-border/80 bg-card">
          <CardHeader>
            <CardTitle className="text-base font-semibold text-foreground">
              Storage Usage
            </CardTitle>
            <CardDescription className="text-xs text-muted-foreground">
              Disk space consumed by locally cached manga covers and reading pages.
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {/* Disk Size */}
              <div className="rounded-lg border border-border/60 bg-muted/20 p-4 flex items-center gap-4">
                <div className="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary shrink-0">
                  <HardDrive className="size-5" />
                </div>
                <div>
                  <span className="text-xs text-muted-foreground font-medium block">Total Cache Size</span>
                  <span className="text-lg font-bold font-mono text-foreground">
                    {isLoadingCacheStats ? '...' : formatBytes(cacheStats?.size_bytes ?? 0)}
                  </span>
                </div>
              </div>

              {/* Total Files */}
              <div className="rounded-lg border border-border/60 bg-muted/20 p-4 flex items-center gap-4">
                <div className="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary shrink-0">
                  <ImageIcon className="size-5" />
                </div>
                <div>
                  <span className="text-xs text-muted-foreground font-medium block">Cached Files</span>
                  <span className="text-lg font-bold font-mono text-foreground">
                    {isLoadingCacheStats ? '...' : (cacheStats?.item_count ?? 0).toLocaleString()}
                  </span>
                </div>
              </div>
            </div>

            <div className="rounded-lg bg-muted/30 p-3.5 border border-border/40 text-xs text-muted-foreground flex items-start gap-2.5">
              <Info className="size-4 text-primary shrink-0 mt-0.5" />
              <p className="leading-relaxed">
                Images are cached locally to provide fast reading response times and minimize outbound requests to providers.
                Clearing the cache frees up disk storage immediately. Cached images will be downloaded again as you browse and read.
              </p>
            </div>
          </CardContent>

          <CardFooter className="flex items-center justify-end border-t border-border/60 bg-muted/20 px-6 py-4">
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => setConfirmClearCacheOpen(true)}
              disabled={
                isLoadingCacheStats ||
                clearCacheMutation.isPending ||
                (cacheStats?.item_count === 0 && cacheStats?.size_bytes === 0)
              }
              className="text-xs font-semibold cursor-pointer gap-2"
            >
              {clearCacheMutation.isPending ? (
                <>
                  <RefreshCw className="size-4 animate-spin" />
                  <span>Clearing...</span>
                </>
              ) : (
                <>
                  <Trash2 className="size-4" />
                  <span>Clear Image Cache</span>
                </>
              )}
            </Button>
          </CardFooter>
        </Card>
      </div>

      {/* Offline Query Cache Card */}
      <div className="max-w-2xl">
        <Card className="border border-border/80 bg-card">
          <CardHeader>
            <CardTitle className="text-base font-semibold text-foreground flex items-center gap-2">
              <Database className="size-4 text-primary" />
              Offline Query Cache
            </CardTitle>
            <CardDescription className="text-xs text-muted-foreground">
              Browser IndexedDB storage used to persist library catalog, metadata, and chapter listings for offline browsing.
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4">
            <div className="rounded-lg bg-muted/30 p-3.5 border border-border/40 text-xs text-muted-foreground flex items-start gap-2.5">
              <Info className="size-4 text-primary shrink-0 mt-0.5" />
              <p className="leading-relaxed">
                Kiyomi caches queries locally in your browser so you can access your saved manga library and chapter details even when offline. Clearing this cache will reset the offline data store and refetch fresh data from the server.
              </p>
            </div>
          </CardContent>

          <CardFooter className="flex items-center justify-end border-t border-border/60 bg-muted/20 px-6 py-4">
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => setConfirmClearOfflineCacheOpen(true)}
              disabled={isClearingOfflineCache}
              className="text-xs font-semibold cursor-pointer gap-2"
            >
              <Trash2 className="size-4" />
              <span>Clear Offline Cache</span>
            </Button>
          </CardFooter>
        </Card>
      </div>

      {/* Confirmation Dialog for Clearing Image Cache */}
      <Dialog open={confirmClearCacheOpen} onOpenChange={setConfirmClearCacheOpen}>
        <DialogContent className="max-w-md sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-foreground">
              Clear Image Cache
            </DialogTitle>
            <DialogDescription className="text-xs text-muted-foreground pt-1">
              Are you sure you want to clear the local image cache? All cached manga cover images and chapter pages will be removed from disk. Cached items will be re-downloaded on demand when viewed.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex items-center justify-end gap-2 pt-4">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setConfirmClearCacheOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              disabled={clearCacheMutation.isPending}
              onClick={() => {
                setConfirmClearCacheOpen(false);
                clearCacheMutation.mutate();
              }}
              className="gap-2"
            >
              <Trash2 className="size-4" />
              <span>Clear Cache</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Confirmation Dialog for Clearing Offline Query Cache */}
      <Dialog open={confirmClearOfflineCacheOpen} onOpenChange={setConfirmClearOfflineCacheOpen}>
        <DialogContent className="max-w-md sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-foreground">
              Clear Offline Query Cache
            </DialogTitle>
            <DialogDescription className="text-xs text-muted-foreground pt-1">
              Are you sure you want to clear the offline query cache? Persisted library metadata, catalog results, and chapter lists in browser storage will be removed. Fresh data will be loaded from the server on demand.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex items-center justify-end gap-2 pt-4">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setConfirmClearOfflineCacheOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              disabled={isClearingOfflineCache}
              onClick={async () => {
                setIsClearingOfflineCache(true);
                try {
                  await clearQueryPersistence(queryClient);
                  showToast('Offline query cache cleared successfully', 'success');
                } catch (err: any) {
                  showToast('Failed to clear offline cache', 'error', err?.message);
                } finally {
                  setIsClearingOfflineCache(false);
                  setConfirmClearOfflineCacheOpen(false);
                }
              }}
              className="gap-2"
            >
              <Trash2 className="size-4" />
              <span>Clear Offline Cache</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Error Details Modal */}
      <ErrorDetailsModal
        open={errorModalOpen}
        onOpenChange={setErrorModalOpen}
        title={errorTitle}
        message={errorMessage}
        details={errorDetails}
      />
    </div>
  );
};
