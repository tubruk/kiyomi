import React, { useState } from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';
import { ProviderCollision } from '../../types/api';
import { Badge } from '../ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { api } from '../../api/client';
import { useToast } from '../../context/ToastContext';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../lib/queryKeys';

interface CollisionResolutionAlertProps {
  collisions: ProviderCollision[];
}

export const CollisionResolutionAlert: React.FC<CollisionResolutionAlertProps> = ({
  collisions,
}) => {
  const { showToast } = useToast();
  const queryClient = useQueryClient();
  const [updatingId, setUpdatingId] = useState<string | null>(null);

  if (!collisions || collisions.length === 0) return null;

  const handleSelectPreference = async (providerId: string, preference: string) => {
    setUpdatingId(providerId);
    try {
      await api.setPluginPreference(providerId, preference);
      showToast(
        `Switched active ${providerId} provider to ${preference === 'builtin' ? 'Built-in' : preference}`,
        'success'
      );
      await queryClient.invalidateQueries({ queryKey: queryKeys.collisions.all });
      await queryClient.invalidateQueries({ queryKey: queryKeys.plugins.all });
      await queryClient.invalidateQueries({ queryKey: queryKeys.sources.all });
    } catch (err: any) {
      console.error('Failed to update provider preference:', err);
      showToast(
        err.message || 'Failed to update provider preference',
        'error',
        err.stack
      );
    } finally {
      setUpdatingId(null);
    }
  };

  return (
    <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 sm:p-5 text-amber-950 dark:text-amber-200">
      <div className="flex items-start gap-3">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-amber-500/20 text-amber-600 dark:text-amber-400">
          <AlertTriangle className="size-5" />
        </div>
        <div className="flex-1 space-y-1">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-bold text-amber-900 dark:text-amber-100">
              Provider ID Collision Detected
            </h3>
            <Badge variant="outline" className="border-amber-500/40 text-amber-700 dark:text-amber-300 text-[10px] font-semibold">
              {collisions.length} {collisions.length === 1 ? 'conflict' : 'conflicts'}
            </Badge>
          </div>
          <p className="text-xs text-amber-800/90 dark:text-amber-300/90 leading-relaxed">
            Multiple installed packages register implementations for the same provider identifier. Choose which instance serves incoming library, catalog, and chapter requests.
          </p>

          {/* Collision items list */}
          <div className="mt-4 space-y-3">
            {collisions.map((item) => {
              const isBusy = updatingId === item.providerId;

              return (
                <div
                  key={item.providerId}
                  className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-lg border border-amber-500/20 bg-background/80 p-3 shadow-xs"
                >
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold text-foreground">Provider:</span>
                    <Badge variant="secondary" className="font-mono text-xs font-bold text-primary">
                      {item.providerId}
                    </Badge>
                    <span className="text-xs text-muted-foreground hidden md:inline">
                      ({item.candidates.length} available implementations)
                    </span>
                  </div>

                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-muted-foreground shrink-0">
                      Active Provider:
                    </span>
                    <Select
                      value={item.selected}
                      onValueChange={(val) => { if (val) handleSelectPreference(item.providerId, val); }}
                      disabled={isBusy}
                    >
                      <SelectTrigger className="w-56 text-xs h-8 bg-card border-border">
                        <SelectValue placeholder="Select active provider" />
                      </SelectTrigger>
                      <SelectContent>
                        {item.candidates.map((cand) => {
                          const valueKey = cand.isBuiltIn ? 'builtin' : cand.pluginId;
                          const label = cand.isBuiltIn
                            ? `Built-in (in-process direct v${cand.version})`
                            : `Plugin: ${cand.pluginId} (v${cand.version})`;

                          return (
                            <SelectItem key={valueKey} value={valueKey} className="text-xs">
                              {label}
                            </SelectItem>
                          );
                        })}
                      </SelectContent>
                    </Select>

                    {isBusy && <RefreshCw className="size-4 animate-spin text-primary shrink-0" />}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
};
