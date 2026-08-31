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
  Download,
  Smartphone,
  Database,
  CheckCircle2,
  Sparkles,
  Loader2,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { settingsTabRoute } from '../router';
import {
  pluginsQueryOptions,
  collisionsQueryOptions,
  infoQueryOptions,
  cacheStatsQueryOptions,
  jobsQueryOptions,
} from '../lib/queryOptions';
import { queryKeys } from '../lib/queryKeys';
import { api } from '../api/client';
import { useToast } from '../context/ToastContext';
import { usePWAInstall } from '../hooks/usePWAInstall';
import { clearQueryPersistence } from '../lib/persister';
import { PluginItem, Job } from '../types/api';
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
import { Input } from '../components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { formatBytes } from '../lib/utils';

const JobRow: React.FC<{
  job: Job;
  onCancel: (id: string) => void;
  isCancelling: boolean;
  onDelete: (id: string) => void;
  isDeleting: boolean;
  depth?: number;
}> = ({ job, onCancel, isCancelling, onDelete, isDeleting, depth = 0 }) => {
  const [expanded, setExpanded] = useState(false);
  const [showError, setShowError] = useState(false);

  const { data: childJobs = [], isLoading: isLoadingChildren } = useQuery({
    ...jobsQueryOptions({ parent_id: job.id }),
    enabled: expanded && Boolean(job.child_count && job.child_count > 0),
  });

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'running':
        return (
          <Badge className="bg-blue-500/10 text-blue-500 hover:bg-blue-500/10 border-blue-500/20 font-medium flex items-center gap-1.5 w-fit">
            <Loader2 className="size-3 animate-spin" />
            Running
          </Badge>
        );
      case 'pending':
        return (
          <Badge variant="outline" className="bg-amber-500/10 text-amber-500 hover:bg-amber-500/10 border-amber-500/20 font-medium w-fit">
            Pending
          </Badge>
        );
      case 'completed':
        return (
          <Badge variant="outline" className="bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500/10 border-emerald-500/20 font-medium w-fit">
            Completed
          </Badge>
        );
      case 'failed':
        return (
          <Badge variant="outline" className="bg-rose-500/10 text-rose-500 hover:bg-rose-500/10 border-rose-500/20 font-medium w-fit">
            Failed
          </Badge>
        );
      default:
        return <Badge variant="outline" className="w-fit">{status}</Badge>;
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '—';
    try {
      const d = new Date(dateStr);
      return d.toLocaleString(undefined, {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <>
      <tr className="hover:bg-muted/10 align-top transition-colors">
        <td className="px-4 py-3 font-mono text-[10px] break-all max-w-[120px]">
          {job.id}
        </td>
        <td className="px-4 py-3 font-semibold break-all max-w-[180px]">
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-1.5 flex-wrap">
              <span>{job.type}</span>
              {job.child_count && job.child_count > 0 ? (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setExpanded(!expanded)}
                  className="h-6 px-1.5 text-[10px] gap-1 text-primary hover:text-primary hover:bg-primary/10 border border-primary/20"
                >
                  {expanded ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
                  <span>Sub-jobs ({job.child_count})</span>
                </Button>
              ) : null}
            </div>
          </div>
        </td>
        <td className="px-4 py-3 text-muted-foreground font-mono text-[10px]">
          {job.concurrency_group || '—'}
        </td>
        <td className="px-4 py-3">
          <div className="flex flex-wrap gap-1 max-w-[200px]">
            {job.metadata && Object.keys(job.metadata).length > 0 ? (
              Object.entries(job.metadata).map(([k, v]) => (
                <Badge
                  key={k}
                  variant="outline"
                  className="px-1.5 py-0.5 text-[9px] font-mono leading-none bg-muted/40 text-muted-foreground hover:bg-muted/40"
                >
                  {k}={String(v)}
                </Badge>
              ))
            ) : (
              <span className="text-muted-foreground">—</span>
            )}
          </div>
        </td>
        <td className="px-4 py-3">
          {getStatusBadge(job.status)}
        </td>
        <td className="px-4 py-3 text-muted-foreground">
          {job.retries} / {job.max_retries}
        </td>
        <td className="px-4 py-3 space-y-0.5 text-[10px] text-muted-foreground font-mono leading-tight">
          <div><span className="text-muted-foreground/60">Created:</span> {formatDate(job.created_at)}</div>
          {job.started_at && <div><span className="text-muted-foreground/60">Started:</span> {formatDate(job.started_at)}</div>}
          {job.completed_at && <div><span className="text-muted-foreground/60">Ended:</span> {formatDate(job.completed_at)}</div>}
        </td>
        <td className="px-4 py-3 text-right">
          <div className="flex items-center justify-end gap-1.5">
            {job.status === 'failed' && job.error && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-[11px] font-medium cursor-pointer"
                onClick={() => setShowError(!showError)}
              >
                {showError ? 'Hide Error' : 'Show Error'}
              </Button>
            )}
            {(job.status === 'pending' || job.status === 'running') && (
              <Button
                variant="destructive"
                size="sm"
                className="h-7 px-2 text-[11px] font-medium cursor-pointer"
                disabled={isCancelling}
                onClick={() => onCancel(job.id)}
              >
                {isCancelling ? 'Cancelling...' : 'Cancel'}
              </Button>
            )}
            {(job.status === 'completed' || job.status === 'failed') && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-[11px] font-medium cursor-pointer text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                disabled={isDeleting}
                onClick={() => onDelete(job.id)}
                title="Remove job from history"
              >
                <Trash2 className="size-3 mr-1" />
                <span>{isDeleting ? 'Removing...' : 'Remove'}</span>
              </Button>
            )}
          </div>
        </td>
      </tr>
      {showError && job.status === 'failed' && job.error && (
        <tr className="bg-rose-500/[0.02] border-t-0">
          <td colSpan={8} className="px-4 pb-3 pt-0">
            <div className="rounded-md border border-rose-500/10 bg-rose-500/[0.03] p-3 text-[11px] font-mono text-rose-600 dark:text-rose-400 break-all whitespace-pre-wrap max-h-48 overflow-y-auto">
              <strong>Error:</strong> {job.error}
            </div>
          </td>
        </tr>
      )}
      {expanded && Boolean(job.child_count && job.child_count > 0) && (
        <tr className="bg-muted/5 border-t-0">
          <td colSpan={8} className="p-0">
            <div className="border-l-2 border-primary/30 pl-3 my-1.5 mr-2 ml-4">
              <div className="rounded-lg border border-border/60 bg-background/90 overflow-hidden my-2 shadow-xs">
                <div className="bg-muted/30 px-3 py-2 text-[11px] font-medium text-muted-foreground flex items-center justify-between border-b border-border/40">
                  <span className="font-mono">↳ Sub-jobs of {job.type} ({job.id.slice(0, 8)})</span>
                  <span className="text-[10px] text-muted-foreground font-mono">{job.child_count} sub-job{job.child_count === 1 ? '' : 's'}</span>
                </div>
                {isLoadingChildren ? (
                  <div className="p-4 text-center text-xs text-muted-foreground flex items-center justify-center gap-2">
                    <Loader2 className="size-3 animate-spin text-primary" />
                    <span>Loading sub-jobs...</span>
                  </div>
                ) : childJobs.length === 0 ? (
                  <div className="p-4 text-center text-xs text-muted-foreground">
                    No sub-jobs found.
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse min-w-[700px]">
                      <thead>
                        <tr className="bg-muted/20 text-[10px] text-muted-foreground font-semibold border-b border-border/40">
                          <th className="px-4 py-2">Job ID</th>
                          <th className="px-4 py-2">Type</th>
                          <th className="px-4 py-2">Concurrency Group</th>
                          <th className="px-4 py-2">Metadata</th>
                          <th className="px-4 py-2">Status</th>
                          <th className="px-4 py-2">Retries</th>
                          <th className="px-4 py-2">Timestamps</th>
                          <th className="px-4 py-2 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border/40 text-xs text-foreground">
                        {childJobs.map((childJob) => (
                          <JobRow
                            key={childJob.id}
                            job={childJob}
                            onCancel={onCancel}
                            isCancelling={isCancelling}
                            onDelete={onDelete}
                            isDeleting={isDeleting}
                            depth={depth + 1}
                          />
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  );
};

export const SettingsPage: React.FC = () => {
  const navigate = useNavigate();
  const { tab } = settingsTabRoute.useParams();
  const currentTab = tab && ['plugins', 'cache', 'jobs', 'about'].includes(tab) ? tab : 'plugins';
  const handleTabChange = (newTab: string) => navigate({ to: '/settings/$tab', params: { tab: newTab } });

  const { showToast } = useToast();
  const queryClient = useQueryClient();
  const { isInstallable, isInstalled, install } = usePWAInstall();

  const [selectedSettingsPlugin, setSelectedSettingsPlugin] = useState<PluginItem | null>(null);
  const [selectedLogsPlugin, setSelectedLogsPlugin] = useState<PluginItem | null>(null);

  // Clear cache confirmation modals
  const [confirmClearCacheOpen, setConfirmClearCacheOpen] = useState(false);
  const [confirmClearOfflineCacheOpen, setConfirmClearOfflineCacheOpen] = useState(false);
  const [isClearingOfflineCache, setIsClearingOfflineCache] = useState(false);

  // Error modal state
  const [errorModalOpen, setErrorModalOpen] = useState(false);
  const [errorTitle, setErrorTitle] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [errorDetails, setErrorDetails] = useState('');

  // Job filters state
  const [jobViewMode, setJobViewMode] = useState<'hierarchical' | 'flat'>('hierarchical');
  const [jobStatusFilter, setJobStatusFilter] = useState<string>('all');
  const [jobTypeFilter, setJobTypeFilter] = useState<string>('');

  // Queries
  const { data: plugins = [], isLoading: isLoadingPlugins } = useQuery({
    ...pluginsQueryOptions(),
    enabled: currentTab === 'plugins',
  });
  const { data: collisions = [] } = useQuery({
    ...collisionsQueryOptions(),
    enabled: currentTab === 'plugins',
  });
  const { data: info } = useQuery({
    ...infoQueryOptions(),
    enabled: currentTab === 'about',
  });

  const jobFilter = React.useMemo(() => {
    if (jobViewMode === 'flat') {
      return { all: true };
    }
    return undefined;
  }, [jobViewMode]);

  const { data: allJobs = [], isLoading: isLoadingJobs } = useQuery({
    ...jobsQueryOptions(jobFilter),
    enabled: currentTab === 'jobs',
  });

  const cancelJobMutation = useMutation({
    mutationFn: (id: string) => api.cancelJob(id),
    onSuccess: () => {
      showToast('Job cancelled successfully', 'success');
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
    onError: (err: any) => {
      console.error('Failed to cancel job:', err);
      const msg = err.message || 'Failed to cancel job';
      const details = err.details
        ? typeof err.details === 'object'
          ? JSON.stringify(err.details, null, 2)
          : String(err.details)
        : err.stack || '';
      showToast(msg, 'error', details);
    },
  });

  const deleteJobMutation = useMutation({
    mutationFn: (id: string) => api.deleteJob(id),
    onSuccess: () => {
      showToast('Job removed successfully', 'success');
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
    onError: (err: any) => {
      console.error('Failed to remove job:', err);
      const msg = err.message || 'Failed to remove job';
      const details = err.details
        ? typeof err.details === 'object'
          ? JSON.stringify(err.details, null, 2)
          : String(err.details)
        : err.stack || '';
      showToast(msg, 'error', details);
    },
  });

  const cleanupJobsMutation = useMutation({
    mutationFn: (status: string) => api.cleanupJobs(status),
    onSuccess: (data) => {
      showToast(`Removed ${data.deleted} completed job${data.deleted === 1 ? '' : 's'}`, 'success');
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all });
    },
    onError: (err: any) => {
      console.error('Failed to clean up jobs:', err);
      const msg = err.message || 'Failed to clean up jobs';
      const details = err.details
        ? typeof err.details === 'object'
          ? JSON.stringify(err.details, null, 2)
          : String(err.details)
        : err.stack || '';
      showToast(msg, 'error', details);
    },
  });

  const filteredJobs = React.useMemo(() => {
    return allJobs
      .filter((job) => {
        const matchStatus = jobStatusFilter === 'all' || job.status === jobStatusFilter;
        const matchType = jobTypeFilter.trim() === '' || job.type.toLowerCase().includes(jobTypeFilter.toLowerCase().trim());
        return matchStatus && matchType;
      })
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }, [allJobs, jobStatusFilter, jobTypeFilter]);

  const stats = React.useMemo(() => {
    let active = 0;
    let pending = 0;
    let completed = 0;
    let failed = 0;
    allJobs.forEach((job) => {
      if (job.status === 'running') active++;
      else if (job.status === 'pending') pending++;
      else if (job.status === 'completed') completed++;
      else if (job.status === 'failed') failed++;
    });
    return { active, pending, completed, failed };
  }, [allJobs]);
  const {
    data: cacheStats,
    isLoading: isLoadingCacheStats,
    refetch: refetchCacheStats,
    isRefetching: isRefetchingCacheStats,
  } = useQuery({
    ...cacheStatsQueryOptions(),
    enabled: currentTab === 'cache',
  });

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
      <Tabs value={currentTab} onValueChange={handleTabChange}>
        <TabsList className="mb-2">
          <TabsTrigger value="plugins">Plugins</TabsTrigger>
          <TabsTrigger value="cache">Cache</TabsTrigger>
          <TabsTrigger value="jobs">Jobs</TabsTrigger>
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
        </TabsContent>

        {/* ── Jobs Tab ── */}
        <TabsContent value="jobs" className="space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-base font-bold text-foreground flex items-center gap-2">
                <Clock className="size-4 text-primary" />
                Background Jobs
              </h2>
              <p className="text-xs text-muted-foreground">
                Monitor and manage background jobs.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => cleanupJobsMutation.mutate('completed')}
                disabled={stats.completed === 0 || cleanupJobsMutation.isPending}
                className="text-xs font-semibold cursor-pointer gap-1.5 text-muted-foreground hover:text-destructive"
              >
                <Trash2 className={`size-3.5 ${cleanupJobsMutation.isPending ? 'animate-spin' : ''}`} />
                <span>Clear Completed</span>
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all })}
                disabled={isLoadingJobs}
                className="text-xs font-semibold cursor-pointer gap-2"
              >
                <RefreshCw className={`size-3.5 ${isLoadingJobs ? 'animate-spin' : ''}`} />
                <span>Refresh</span>
              </Button>
            </div>
          </div>

          {/* Stats Row */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card className="border border-border/60 bg-muted/15 p-4 flex flex-col justify-between">
              <span className="text-xs font-medium text-muted-foreground">Active</span>
              <span className="text-2xl font-bold font-mono mt-1 text-blue-500">{stats.active}</span>
            </Card>
            <Card className="border border-border/60 bg-muted/15 p-4 flex flex-col justify-between">
              <span className="text-xs font-medium text-muted-foreground">Pending</span>
              <span className="text-2xl font-bold font-mono mt-1 text-amber-500">{stats.pending}</span>
            </Card>
            <Card className="border border-border/60 bg-muted/15 p-4 flex flex-col justify-between">
              <span className="text-xs font-medium text-muted-foreground">Completed</span>
              <span className="text-2xl font-bold font-mono mt-1 text-emerald-500">{stats.completed}</span>
            </Card>
            <Card className="border border-border/60 bg-muted/15 p-4 flex flex-col justify-between">
              <span className="text-xs font-medium text-muted-foreground">Failed</span>
              <span className="text-2xl font-bold font-mono mt-1 text-rose-500">{stats.failed}</span>
            </Card>
          </div>

          {/* Filters */}
          <div className="flex flex-col sm:flex-row gap-3">
            <div className="w-full sm:w-52">
              <Select value={jobViewMode} onValueChange={(val) => setJobViewMode(val as 'hierarchical' | 'flat')}>
                <SelectTrigger>
                  <SelectValue placeholder="View mode">
                    {jobViewMode === 'flat' ? 'Flat (All jobs)' : 'Hierarchical (Default)'}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="hierarchical">Hierarchical (Default)</SelectItem>
                  <SelectItem value="flat">Flat (All jobs)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="w-full sm:w-48">
              <Select value={jobStatusFilter} onValueChange={(val) => setJobStatusFilter(val || 'all')}>
                <SelectTrigger>
                  <SelectValue placeholder="Filter by status">
                    {jobStatusFilter === 'pending'
                      ? 'Pending'
                      : jobStatusFilter === 'running'
                      ? 'Running'
                      : jobStatusFilter === 'completed'
                      ? 'Completed'
                      : jobStatusFilter === 'failed'
                      ? 'Failed'
                      : 'All Statuses'}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Statuses</SelectItem>
                  <SelectItem value="pending">Pending</SelectItem>
                  <SelectItem value="running">Running</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex-1">
              <Input
                placeholder="Filter by job type..."
                value={jobTypeFilter}
                onChange={(e) => setJobTypeFilter(e.target.value)}
              />
            </div>
          </div>

          {/* Job List/Table */}
          <Card className="border border-border/80 bg-card overflow-hidden">
            {isLoadingJobs ? (
              <div className="p-8 text-center space-y-4">
                <Loader2 className="size-8 animate-spin mx-auto text-primary" />
                <p className="text-xs text-muted-foreground">Loading jobs...</p>
              </div>
            ) : filteredJobs.length === 0 ? (
              <div className="p-8 text-center text-xs text-muted-foreground">
                No jobs found.
              </div>
            ) : (
              <div className="divide-y divide-border/60 overflow-x-auto">
                <table className="w-full text-left border-collapse min-w-[800px]">
                  <thead>
                    <tr className="bg-muted/35 text-xs text-muted-foreground font-semibold border-b border-border/60">
                      <th className="px-4 py-3">Job ID</th>
                      <th className="px-4 py-3">Type</th>
                      <th className="px-4 py-3">Concurrency Group</th>
                      <th className="px-4 py-3">Metadata</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Retries</th>
                      <th className="px-4 py-3">Timestamps</th>
                      <th className="px-4 py-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border/60 text-xs text-foreground">
                    {filteredJobs.map((job) => (
                      <JobRow
                        key={job.id}
                        job={job}
                        onCancel={(id) => cancelJobMutation.mutate(id)}
                        isCancelling={cancelJobMutation.isPending && cancelJobMutation.variables === job.id}
                        onDelete={(id) => deleteJobMutation.mutate(id)}
                        isDeleting={deleteJobMutation.isPending && deleteJobMutation.variables === job.id}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </TabsContent>

        {/* ── About Tab ── */}
        <TabsContent value="about" className="space-y-6">
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
                <div className="flex items-center gap-2 mt-1">
                  <Badge variant="outline" className="text-xs font-mono font-semibold border-primary/30 text-primary">
                    v{info?.version ?? '—'}
                  </Badge>
                  {isInstalled ? (
                    <Badge variant="secondary" className="text-xs font-medium text-emerald-600 dark:text-emerald-400 gap-1">
                      <CheckCircle2 className="size-3" />
                      Installed App
                    </Badge>
                  ) : isInstallable ? (
                    <Badge variant="secondary" className="text-xs font-medium text-primary gap-1">
                      <Sparkles className="size-3" />
                      Installable PWA
                    </Badge>
                  ) : null}
                </div>
              </div>
            </div>

            {/* App Details & Installation Card */}
            <Card className="border border-border/80 bg-card">
              <CardHeader>
                <CardTitle className="text-base font-semibold text-foreground flex items-center gap-2">
                  <Smartphone className="size-4 text-primary" />
                  App Details & Installation
                </CardTitle>
                <CardDescription className="text-xs text-muted-foreground">
                  Install Kiyomi as a Progressive Web App (PWA) for a native standalone app experience and offline access.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="rounded-lg border border-border/60 bg-muted/20 p-3.5 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-xs font-medium text-foreground block">Application Mode</span>
                    <span className="text-xs text-muted-foreground">
                      {isInstalled
                        ? 'Running as standalone installed application'
                        : isInstallable
                        ? 'Ready to install on this device'
                        : 'Running in web browser'}
                    </span>
                  </div>
                  {isInstalled ? (
                    <Badge variant="outline" className="text-xs text-emerald-500 border-emerald-500/30 gap-1 shrink-0">
                      <CheckCircle2 className="size-3" />
                      Installed
                    </Badge>
                  ) : isInstallable ? (
                    <Button
                      type="button"
                      size="sm"
                      onClick={async () => {
                        const installed = await install();
                        if (installed) {
                          showToast('Kiyomi installed successfully!', 'success');
                        }
                      }}
                      className="text-xs font-semibold gap-1.5 cursor-pointer shrink-0"
                    >
                      <Download className="size-3.5" />
                      <span>Install App</span>
                    </Button>
                  ) : (
                    <Badge variant="outline" className="text-xs text-muted-foreground shrink-0">
                      Web Browser
                    </Badge>
                  )}
                </div>

                <div className="rounded-lg border border-border/60 bg-muted/20 p-3.5 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-xs font-medium text-foreground block">Offline Query Cache</span>
                    <span className="text-xs text-muted-foreground">
                      Persisted library and chapter state in browser storage
                    </span>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setConfirmClearOfflineCacheOpen(true)}
                    disabled={isClearingOfflineCache}
                    className="text-xs font-semibold gap-1.5 cursor-pointer shrink-0"
                  >
                    <Trash2 className="size-3.5 text-muted-foreground" />
                    <span>Clear Cache</span>
                  </Button>
                </div>
              </CardContent>
            </Card>

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
