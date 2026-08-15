import React, { useState } from 'react';
import {
  Puzzle,
  RefreshCw,
  FolderOpen,
  GitCommit,
  Clock,
  Code2,
  Tag,
} from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  pluginsQueryOptions,
  collisionsQueryOptions,
  infoQueryOptions,
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
import { Card, CardTitle, CardDescription } from '../components/ui/card';
import { ErrorDetailsModal } from '../components/ErrorDetailsModal';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs';

export const SettingsPage: React.FC = () => {
  const { showToast } = useToast();
  const queryClient = useQueryClient();

  const [selectedSettingsPlugin, setSelectedSettingsPlugin] = useState<PluginItem | null>(null);
  const [selectedLogsPlugin, setSelectedLogsPlugin] = useState<PluginItem | null>(null);

  // Error modal state
  const [errorModalOpen, setErrorModalOpen] = useState(false);
  const [errorTitle, setErrorTitle] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [errorDetails, setErrorDetails] = useState('');

  // Queries
  const { data: plugins = [], isLoading: isLoadingPlugins } = useQuery(pluginsQueryOptions());
  const { data: collisions = [] } = useQuery(collisionsQueryOptions());
  const { data: info } = useQuery(infoQueryOptions());

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

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 space-y-6">
      {/* Page Header */}
      <div className="space-y-1 border-b border-border pb-5">
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Settings</h1>
        <p className="text-xs sm:text-sm text-muted-foreground">
          Manage your content providers and app information.
        </p>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="plugins">
        <TabsList className="mb-2">
          <TabsTrigger value="plugins">Plugins</TabsTrigger>
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
