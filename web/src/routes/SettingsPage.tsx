import React, { useState } from 'react';
import {
  Puzzle,
  RefreshCw,
  FolderOpen,
  GitCommit,
  Clock,
  Code2,
  Tag,
  HardDrive,
  Trash2,
  Image as ImageIcon,
  Info,
} from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  pluginsQueryOptions,
  collisionsQueryOptions,
  infoQueryOptions,
  cacheStatsQueryOptions,
} from '../lib/queryOptions';
import { queryKeys } from '../lib/queryKeys';
import { api } from '../api/client';
import { useToast } from '../context/ToastContext';
import { PluginItem } from '../types/api';
import { PluginCard } from '../components/plugins/PluginCard';
import { ScopedSettingsModal } from '../components/plugins/ScopedSettingsModal';
import { DiagnosticLogsModal } from '../components/plugins/DiagnosticLogsModal';
import { CollisionResolutionAlert } from '../components/plugins/CollisionResolutionAlert';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from '../components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../components/ui/dialog';
import { ErrorDetailsModal } from '../components/ErrorDetailsModal';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs';
import { formatBytes } from '../lib/utils';

export const SettingsPage: React.FC = () => {
  const { showToast } = useToast();
  const queryClient = useQueryClient();

  const [selectedSettingsPlugin, setSelectedSettingsPlugin] = useState<PluginItem | null>(null);
  const [selectedLogsPlugin, setSelectedLogsPlugin] = useState<PluginItem | null>(null);

  // Clear cache confirmation modal
  const [confirmClearCacheOpen, setConfirmClearCacheOpen] = useState(false);

  // Error modal state
  const [errorModalOpen, setErrorModalOpen] = useState(false);
  const [errorTitle, setErrorTitle] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [errorDetails, setErrorDetails] = useState('');

  // Queries
  const { data: plugins = [], isLoading: isLoadingPlugins } = useQuery(pluginsQueryOptions());
  const { data: collisions = [] } = useQuery(collisionsQueryOptions());
  const { data: info } = useQuery(infoQueryOptions());
  const {
    data: cacheStats,
    isLoading: isLoadingCacheStats,
    refetch: refetchCacheStats,
    isRefetching: isRefetchingCacheStats,
  } = useQuery(cacheStatsQueryOptions());

  // Reload mutation
  const reloadMutation = useMutation({
    mutationFn: () => api.reloadPlugins(),
    onSuccess: (data) => {
      showToast(
        data.message || `Reloaded ${data.reloadedPlugins.length} plugin(s)`,
        'success'
      );
      queryClient.invalidateQueries({ queryKey: queryKeys.plugins.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.collisions.all });
    },
    onError: (err: any) => {
      console.error('Failed to reload plugins:', err);
      const msg = err.message || 'Failed to reload plugins';
      const details = err.details
        ? typeof err.details === 'object'
          ? JSON.stringify(err.details, null, 2)
        : String(err.details)
        : err.stack || '';

      showToast(msg, 'error', details);
      setErrorTitle('Failed to reload plugins');
      setErrorMessage(msg);
      setErrorDetails(details);
      setErrorModalOpen(true);
    },
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
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 space-y-6">
      {/* Page Header */}
      <div className="space-y-1 border-b border-border pb-5">
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Settings</h1>
        <p className="text-xs sm:text-sm text-muted-foreground">
          Manage your content providers, system storage, and app information.
        </p>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="plugins">
        <TabsList className="mb-2">
          <TabsTrigger value="plugins">Plugins</TabsTrigger>
          <TabsTrigger value="cache">Cache</TabsTrigger>
          <TabsTrigger value="about">About</TabsTrigger>
        </TabsList>

        {/* ── Plugins Tab ── */}
        <TabsContent value="plugins" className="space-y-6">
          {/* Tab action row */}
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-base font-bold text-foreground flex items-center gap-2">
                <Puzzle className="size-4 text-primary" />
                Installed Plugins
              </h2>
              <p className="text-xs text-muted-foreground">
                Plugins that add new manga sources and capabilities to Kiyomi.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="secondary" className="text-xs font-mono font-bold">
                {plugins.length} active
              </Badge>
              <Button
                type="button"
                onClick={() => reloadMutation.mutate()}
                disabled={reloadMutation.isPending}
                className="text-xs sm:text-sm font-semibold cursor-pointer gap-2"
              >
                <RefreshCw className={`size-4 ${reloadMutation.isPending ? 'animate-spin' : ''}`} />
                <span>{reloadMutation.isPending ? 'Reloading...' : 'Reload'}</span>
              </Button>
            </div>
          </div>

          {/* Collision Alert */}
          {collisions.length > 0 && <CollisionResolutionAlert collisions={collisions} />}

          {/* Plugin Grid */}
          {isLoadingPlugins ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {[1, 2].map((i) => (
                <div
                  key={i}
                  className="h-56 rounded-xl border border-border/60 bg-muted/20 animate-pulse"
                />
              ))}
            </div>
          ) : plugins.length === 0 ? (
            <Card className="border-dashed border-border/80 bg-muted/10 p-8 text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary mb-3">
                <FolderOpen className="size-6 opacity-60" />
              </div>
              <CardTitle className="text-base font-semibold text-foreground">
                No Plugins Installed
              </CardTitle>
              <CardDescription className="text-xs text-muted-foreground max-w-md mx-auto mt-1 leading-relaxed">
                Place Kiyomi provider plugins inside your configured plugin directory and click{' '}
                <strong>Reload</strong>.
              </CardDescription>
            </Card>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {plugins.map((plugin) => (
                <PluginCard
                  key={plugin.pluginId}
                  plugin={plugin}
                  onOpenSettings={(p) => setSelectedSettingsPlugin(p)}
                  onOpenLogs={(p) => setSelectedLogsPlugin(p)}
                />
              ))}
            </div>
          )}
        </TabsContent>

        {/* ── Cache Tab ── */}
        <TabsContent value="cache" className="space-y-6">
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
                  disabled={isLoadingCacheStats || clearCacheMutation.isPending || (cacheStats?.item_count === 0 && cacheStats?.size_bytes === 0)}
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
        </TabsContent>

        {/* ── About Tab ── */}
        <TabsContent value="about">
          <div className="max-w-lg space-y-6">
            {/* App identity */}
            <div className="flex items-center gap-4">
              <div className="flex size-14 items-center justify-center rounded-2xl bg-primary/10 text-primary shrink-0">
                <Puzzle className="size-7" />
              </div>
              <div>
                <p className="text-xl font-bold tracking-tight text-foreground">
                  {info?.app ?? 'Kiyomi'}
                </p>
                <Badge variant="outline" className="text-xs font-mono font-semibold border-primary/30 text-primary mt-1">
                  v{info?.version ?? '—'}
                </Badge>
              </div>
            </div>

            {/* Build metadata */}
            <div className="rounded-xl border border-border bg-muted/20 divide-y divide-border/60">
              <div className="flex items-center gap-3 px-4 py-3">
                <Tag className="size-4 text-muted-foreground shrink-0" />
                <span className="text-xs text-muted-foreground w-24 shrink-0">Version</span>
                <span className="text-xs font-mono text-foreground">{info?.version ?? '—'}</span>
              </div>
              <div className="flex items-center gap-3 px-4 py-3">
                <GitCommit className="size-4 text-muted-foreground shrink-0" />
                <span className="text-xs text-muted-foreground w-24 shrink-0">Commit</span>
                <span className="text-xs font-mono text-foreground">{info?.commit ?? '—'}</span>
              </div>
              <div className="flex items-center gap-3 px-4 py-3">
                <Clock className="size-4 text-muted-foreground shrink-0" />
                <span className="text-xs text-muted-foreground w-24 shrink-0">Build time</span>
                <span className="text-xs font-mono text-foreground">{info?.build_time ?? '—'}</span>
              </div>
              <div className="flex items-center gap-3 px-4 py-3">
                <Code2 className="size-4 text-muted-foreground shrink-0" />
                <span className="text-xs text-muted-foreground w-24 shrink-0">Go version</span>
                <span className="text-xs font-mono text-foreground">{info?.go_version ?? '—'}</span>
              </div>
            </div>
          </div>
        </TabsContent>
      </Tabs>

      {/* Confirmation Dialog for Clearing Cache */}
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

      {/* Modals */}
      <ScopedSettingsModal
        open={Boolean(selectedSettingsPlugin)}
        onOpenChange={(open) => { if (!open) setSelectedSettingsPlugin(null); }}
        plugin={selectedSettingsPlugin}
      />
      <DiagnosticLogsModal
        open={Boolean(selectedLogsPlugin)}
        onOpenChange={(open) => { if (!open) setSelectedLogsPlugin(null); }}
        plugin={selectedLogsPlugin}
      />
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
