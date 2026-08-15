import React, { useState } from 'react';
import {
  Puzzle,
  Terminal,
  Sliders,
  ShieldCheck,
  ShieldAlert,
  Cpu,
  Folder,
  Layers,
  Gauge,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { PluginItem } from '../../types/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../ui/card';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';

interface PluginCardProps {
  plugin: PluginItem;
  onOpenSettings: (plugin: PluginItem) => void;
  onOpenLogs: (plugin: PluginItem) => void;
}

// Maps capability strings to coloured badge styles
function capabilityStyle(cap: string): string {
  switch (cap) {
    case 'content':
      return 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20';
    case 'tracking':
      return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20';
    default:
      return 'bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/20';
  }
}

export const PluginCard: React.FC<PluginCardProps> = ({ plugin, onOpenSettings, onOpenLogs }) => {
  const [showDetails, setShowDetails] = useState(false);

  const isRunning = plugin.state === 'running';
  const isError = plugin.state === 'error';
  const providers = plugin.providers ?? [];
  const singleProvider = providers.length === 1 ? providers[0] : null;

  return (
    <Card className="flex flex-col justify-between overflow-hidden border-border/80 bg-card/70 hover:border-primary/40 hover:shadow-md transition-all duration-200">
      <CardHeader className="p-5 pb-4">
        {/* Plugin identity row */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary mt-0.5">
              <Puzzle className="size-5" />
            </div>
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <CardTitle className="text-base font-bold text-foreground">
                  {plugin.pluginName || plugin.pluginId}
                </CardTitle>
                <Badge variant="secondary" className="text-xs font-mono font-bold">
                  v{plugin.pluginVersion}
                </Badge>
              </div>
              <CardDescription className="text-xs font-mono text-muted-foreground mt-0.5">
                {plugin.pluginId}
              </CardDescription>
            </div>
          </div>

          {/* Status badges */}
          <div className="flex flex-col items-end gap-1.5 shrink-0">
            <Badge
              variant="outline"
              className={`text-xs font-semibold capitalize flex items-center gap-1.5 ${
                isRunning
                  ? 'border-emerald-500/30 text-emerald-600 dark:text-emerald-400 bg-emerald-500/10'
                  : isError
                  ? 'border-destructive/30 text-destructive bg-destructive/10'
                  : 'border-muted text-muted-foreground bg-muted/40'
              }`}
            >
              <span
                className={`size-1.5 rounded-full ${
                  isRunning
                    ? 'bg-emerald-500 animate-pulse'
                    : isError
                    ? 'bg-destructive'
                    : 'bg-muted-foreground'
                }`}
              />
              {plugin.state || 'stopped'}
            </Badge>

            {plugin.sdkCompatible ? (
              <Badge
                variant="outline"
                className="text-[10px] border-emerald-500/30 text-emerald-600 dark:text-emerald-400 flex items-center gap-1 font-semibold"
              >
                <ShieldCheck className="size-3" />
                SDK v{plugin.sdkVersion}
              </Badge>
            ) : (
              <Badge
                variant="outline"
                className="text-[10px] border-destructive/30 text-destructive flex items-center gap-1 font-semibold"
              >
                <ShieldAlert className="size-3" />
                Incompatible SDK
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="p-5 pt-0 space-y-4">
        {/* Provider section — adaptive for 1 or N */}
        {singleProvider ? (
          // Single provider: show description + capabilities only — name is already in card header
          <div className="rounded-lg border border-border/60 bg-muted/30 p-3 space-y-2">
            {singleProvider.defaultRateLimit?.requestsPerSecond ? (
              <div className="flex items-center justify-end">
                <Badge variant="outline" className="text-[10px] text-muted-foreground gap-1 shrink-0 font-mono">
                  <Gauge className="size-3 text-muted-foreground" />
                  {singleProvider.defaultRateLimit.requestsPerSecond} req/s
                </Badge>
              </div>
            ) : null}
            {singleProvider.description && (
              <p className="text-[11px] text-muted-foreground line-clamp-2">{singleProvider.description}</p>
            )}
            <div className="flex flex-wrap items-center gap-1">
              {(singleProvider.capabilities ?? []).map((cap) => (
                <Badge
                  key={cap}
                  variant="outline"
                  className={`text-[10px] font-medium uppercase tracking-wider ${capabilityStyle(cap)}`}
                >
                  {cap}
                </Badge>
              ))}
            </div>
          </div>
        ) : providers.length > 1 ? (
          // Multiple providers: show grouped list
          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs font-semibold text-foreground">
              <span className="flex items-center gap-1.5">
                <Layers className="size-3.5 text-primary" />
                Providers ({providers.length})
              </span>
            </div>
            <div className="space-y-2">
              {providers.map((prov) => (
                <div
                  key={prov.id}
                  className="rounded-lg border border-border/60 bg-muted/30 p-2.5 transition-colors hover:bg-muted/50"
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-1.5 min-w-0">
                      <span className="text-xs font-bold text-foreground truncate">
                        {prov.name || prov.id}
                      </span>
                      <span className="text-[11px] font-mono text-muted-foreground">
                        ({prov.id})
                      </span>
                    </div>
                    {prov.defaultRateLimit?.requestsPerSecond ? (
                      <Badge variant="outline" className="text-[10px] text-muted-foreground gap-1 shrink-0 font-mono">
                        <Gauge className="size-3 text-muted-foreground" />
                        {prov.defaultRateLimit.requestsPerSecond} req/s
                      </Badge>
                    ) : null}
                  </div>
                  {prov.description && (
                    <p className="text-[11px] text-muted-foreground mt-1 line-clamp-1">{prov.description}</p>
                  )}
                  <div className="mt-2 flex flex-wrap items-center gap-1">
                    {(prov.capabilities ?? []).map((cap) => (
                      <Badge
                        key={cap}
                        variant="outline"
                        className={`text-[10px] font-medium uppercase tracking-wider ${capabilityStyle(cap)}`}
                      >
                        {cap}
                      </Badge>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : (
          // No providers registered
          <p className="text-xs text-muted-foreground italic">No providers registered.</p>
        )}

        {/* Collapsible diagnostic details */}
        {showDetails && (
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-muted-foreground pt-3 border-t border-border/40">
            {plugin.pid > 0 && (
              <div className="flex items-center gap-1 font-mono">
                <Cpu className="size-3.5 text-muted-foreground" />
                <span>PID: {plugin.pid}</span>
              </div>
            )}
            <div className="flex items-center gap-1 font-mono truncate max-w-full" title={plugin.executablePath}>
              <Folder className="size-3.5 shrink-0 text-muted-foreground" />
              <span className="truncate">{plugin.executablePath}</span>
            </div>
          </div>
        )}

        {/* Action row */}
        <div className="flex items-center justify-between gap-2 pt-2 border-t border-border/40">
          {/* Show/hide details toggle */}
          <button
            type="button"
            onClick={() => setShowDetails((v) => !v)}
            className="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            {showDetails ? (
              <ChevronDown className="size-3.5" />
            ) : (
              <ChevronRight className="size-3.5" />
            )}
            {showDetails ? 'Hide details' : 'Show details'}
          </button>

          {/* Action buttons */}
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenLogs(plugin)}
              className="text-xs gap-1.5 cursor-pointer hover:bg-muted"
            >
              <Terminal className="size-3.5 text-primary" />
              <span>Logs</span>
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenSettings(plugin)}
              className="text-xs gap-1.5 cursor-pointer hover:bg-muted"
            >
              <Sliders className="size-3.5 text-primary" />
              <span>Settings</span>
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
