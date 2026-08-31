import React, { useState } from 'react';
import {
  Puzzle,
  Tag,
  GitCommit,
  Clock,
  Code2,
  Smartphone,
  CheckCircle2,
  Sparkles,
  Download,
  Trash2,
} from 'lucide-react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { infoQueryOptions } from '../../lib/queryOptions';
import { useToast } from '../../context/ToastContext';
import { usePWAInstall } from '../../hooks/usePWAInstall';
import { clearQueryPersistence } from '../../lib/persister';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '../ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../ui/dialog';

export const SettingsAboutTab: React.FC = () => {
  const { showToast } = useToast();
  const queryClient = useQueryClient();
  const { isInstallable, isInstalled, install } = usePWAInstall();

  const [confirmClearOfflineCacheOpen, setConfirmClearOfflineCacheOpen] = useState(false);
  const [isClearingOfflineCache, setIsClearingOfflineCache] = useState(false);

  const { data: info } = useQuery({
    ...infoQueryOptions(),
  });

  return (
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
    </div>
  );
};
