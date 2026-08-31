import React, { useState } from 'react';
import { Puzzle, RefreshCw, FolderOpen } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { pluginsQueryOptions, collisionsQueryOptions } from '../../lib/queryOptions';
import { queryKeys } from '../../lib/queryKeys';
import { api } from '../../api/client';
import { useToast } from '../../context/ToastContext';
import { PluginItem } from '../../types/api';
import { PluginCard } from '../plugins/PluginCard';
import { ScopedSettingsModal } from '../plugins/ScopedSettingsModal';
import { DiagnosticLogsModal } from '../plugins/DiagnosticLogsModal';
import { CollisionResolutionAlert } from '../plugins/CollisionResolutionAlert';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Card, CardTitle, CardDescription } from '../ui/card';
import { ErrorDetailsModal } from '../ErrorDetailsModal';

export const SettingsPluginsTab: React.FC = () => {
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
  const { data: plugins = [], isLoading: isLoadingPlugins } = useQuery({
    ...pluginsQueryOptions(),
  });
  const { data: collisions = [] } = useQuery({
    ...collisionsQueryOptions(),
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

  return (
    <div className="space-y-6">
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

      {/* Modals */}
      <ScopedSettingsModal
        open={Boolean(selectedSettingsPlugin)}
        onOpenChange={(open) => {
          if (!open) setSelectedSettingsPlugin(null);
        }}
        plugin={selectedSettingsPlugin}
      />
      <DiagnosticLogsModal
        open={Boolean(selectedLogsPlugin)}
        onOpenChange={(open) => {
          if (!open) setSelectedLogsPlugin(null);
        }}
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
