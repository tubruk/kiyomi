import React, { useState, useMemo } from 'react';
import {
  Terminal,
  RefreshCw,
  Copy,
  Check,
  Search,
  Activity,
  Cpu,
  ShieldCheck,
  ShieldAlert,
} from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Input } from '../ui/input';
import { PluginItem } from '../../types/api';
import { useQuery } from '@tanstack/react-query';
import { pluginLogsQueryOptions } from '../../lib/queryOptions';

interface DiagnosticLogsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  plugin: PluginItem | null;
}

export const DiagnosticLogsModal: React.FC<DiagnosticLogsModalProps> = ({
  open,
  onOpenChange,
  plugin,
}) => {
  const [filterQuery, setFilterQuery] = useState('');
  const [selectedLevel, setSelectedLevel] = useState<string>('ALL');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [copied, setCopied] = useState(false);

  const pluginId = plugin?.pluginId || '';

  const { data: logs = [], refetch, isFetching } = useQuery({
    ...pluginLogsQueryOptions(pluginId),
    enabled: open && Boolean(pluginId),
    refetchInterval: open && autoRefresh ? 2000 : false,
  });

  const filteredLogs = useMemo(() => {
    return logs.filter((entry) => {
      if (selectedLevel !== 'ALL') {
        const lvl = (entry.level || 'INFO').toUpperCase();
        if (lvl !== selectedLevel) return false;
      }

      if (filterQuery.trim()) {
        const q = filterQuery.toLowerCase();
        const msg = (entry.message || '').toLowerCase();
        const raw = (entry.raw || '').toLowerCase();
        const fieldsStr = entry.fields ? JSON.stringify(entry.fields).toLowerCase() : '';
        if (!msg.includes(q) && !raw.includes(q) && !fieldsStr.includes(q)) {
          return false;
        }
      }

      return true;
    });
  }, [logs, selectedLevel, filterQuery]);

  const handleCopyLogs = async () => {
    if (logs.length === 0) return;
    const text = logs
      .map((l) => {
        const ts = l.timestamp ? new Date(l.timestamp).toISOString() : '';
        const fields = l.fields && Object.keys(l.fields).length > 0 ? ` | ${JSON.stringify(l.fields)}` : '';
        return `[${ts}] [${l.level || 'INFO'}] ${l.message || l.raw || ''}${fields}`;
      })
      .join('\n');

    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy logs:', err);
    }
  };

  if (!plugin) return null;

  const isRunning = plugin.state === 'running';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl sm:max-w-4xl max-h-[90vh] flex flex-col p-0 overflow-hidden">
        {/* Header */}
        <DialogHeader className="px-6 pt-5 pb-4 border-b border-border/80 bg-card/60">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-lg bg-zinc-900 text-zinc-100 dark:bg-zinc-800">
                <Terminal className="size-5 text-emerald-400" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <DialogTitle className="text-base font-bold tracking-tight text-foreground">
                    {plugin.pluginName} Diagnostics
                  </DialogTitle>
                  <Badge
                    variant="outline"
                    className={`text-[10px] font-semibold capitalize ${
                      isRunning
                        ? 'border-emerald-500/30 text-emerald-600 dark:text-emerald-400 bg-emerald-500/10'
                        : 'border-destructive/30 text-destructive bg-destructive/10'
                    }`}
                  >
                    {plugin.state || 'stopped'}
                  </Badge>
                  {plugin.sdkCompatible ? (
                    <Badge variant="outline" className="text-[10px] border-emerald-500/30 text-emerald-600 dark:text-emerald-400 gap-1 hidden sm:inline-flex">
                      <ShieldCheck className="size-3" />
                      SDK v{plugin.sdkVersion}
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-[10px] border-destructive/30 text-destructive gap-1 hidden sm:inline-flex">
                      <ShieldAlert className="size-3" />
                      Incompatible SDK
                    </Badge>
                  )}
                </div>
                <DialogDescription className="text-xs text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1">
                  <span>Plugin ID: <code className="text-foreground font-semibold">{plugin.pluginId}</code></span>
                  {plugin.pid > 0 && (
                    <span className="flex items-center gap-1">
                      <Cpu className="size-3 text-muted-foreground" /> PID: {plugin.pid}
                    </span>
                  )}
                  <span className="truncate max-w-xs font-mono text-[11px] text-muted-foreground">
                    {plugin.executablePath}
                  </span>
                </DialogDescription>
              </div>
            </div>

            {/* Quick Actions */}
            <div className="flex items-center gap-2 self-end sm:self-auto">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setAutoRefresh((prev) => !prev)}
                className={`text-xs gap-1.5 cursor-pointer ${
                  autoRefresh ? 'bg-primary/10 text-primary border-primary/30' : ''
                }`}
              >
                <Activity className={`size-3.5 ${autoRefresh ? 'animate-pulse text-emerald-500' : ''}`} />
                <span>Auto-refresh: {autoRefresh ? 'ON' : 'OFF'}</span>
              </Button>
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                onClick={() => refetch()}
                disabled={isFetching}
                className="cursor-pointer"
                aria-label="Refresh logs"
              >
                <RefreshCw className={`size-3.5 ${isFetching ? 'animate-spin text-primary' : ''}`} />
              </Button>
            </div>
          </div>

          {/* Filter Bar */}
          <div className="mt-3 flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-border/40">
            <div className="flex items-center gap-1 overflow-x-auto">
              {['ALL', 'INFO', 'WARN', 'ERROR', 'DEBUG'].map((level) => (
                <button
                  key={level}
                  type="button"
                  onClick={() => setSelectedLevel(level)}
                  className={`rounded-md px-2.5 py-1 text-xs font-semibold transition-colors cursor-pointer ${
                    selectedLevel === level
                      ? 'bg-primary text-primary-foreground shadow-xs'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                >
                  {level}
                </button>
              ))}
            </div>

            <div className="relative flex-1 min-w-[200px] max-w-xs">
              <Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                type="text"
                placeholder="Filter logs by message or field..."
                value={filterQuery}
                onChange={(e) => setFilterQuery(e.target.value)}
                className="h-8 pl-8 text-xs bg-background/80"
              />
            </div>
          </div>
        </DialogHeader>

        {/* Terminal Log Viewport */}
        <div className="flex-1 min-h-[350px] max-h-[500px] overflow-y-auto bg-zinc-950 p-4 font-mono text-xs text-zinc-300 select-text border-y border-zinc-800">
          {filteredLogs.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full py-16 text-zinc-500">
              <Terminal className="size-8 mb-2 opacity-30" />
              <p className="text-xs">
                {logs.length === 0
                  ? 'No diagnostic logs captured for this plugin subprocess yet.'
                  : 'No logs match your current search/level filter.'}
              </p>
            </div>
          ) : (
            <div className="space-y-1">
              {filteredLogs.map((entry, idx) => {
                const ts = entry.timestamp
                  ? new Date(entry.timestamp).toISOString().substring(11, 23)
                  : '--:--:--.---';

                const level = (entry.level || 'INFO').toUpperCase();
                let levelColor = 'text-emerald-400 bg-emerald-950/60 border-emerald-800/60';
                if (level === 'WARN') {
                  levelColor = 'text-amber-400 bg-amber-950/60 border-amber-800/60';
                } else if (level === 'ERROR') {
                  levelColor = 'text-rose-400 bg-rose-950/60 border-rose-800/60';
                } else if (level === 'DEBUG') {
                  levelColor = 'text-sky-400 bg-sky-950/60 border-sky-800/60';
                }

                const hasFields = entry.fields && Object.keys(entry.fields).length > 0;

                return (
                  <div
                    key={idx}
                    className="flex flex-wrap items-baseline gap-2 py-0.5 rounded px-1.5 hover:bg-zinc-900/80 transition-colors leading-relaxed group"
                  >
                    <span className="text-zinc-500 shrink-0 select-none text-[11px]">{ts}</span>
                    <span
                      className={`inline-flex items-center px-1.5 py-0.2 rounded text-[10px] font-bold uppercase border shrink-0 ${levelColor}`}
                    >
                      {level}
                    </span>
                    <span className="text-zinc-200 flex-1 break-all">
                      {entry.message || entry.raw}
                    </span>
                    {hasFields && (
                      <span className="text-zinc-400 text-[11px] bg-zinc-900/90 rounded px-1.5 py-0.5 border border-zinc-800/80 break-all">
                        {JSON.stringify(entry.fields)}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Footer */}
        <DialogFooter className="px-6 py-3 border-t border-border/80 bg-muted/40 flex items-center justify-between gap-2">
          <div className="text-xs text-muted-foreground">
            Showing <span className="font-semibold text-foreground">{filteredLogs.length}</span> of{' '}
            <span className="font-semibold text-foreground">{logs.length}</span> buffered entries (ring buffer capacity: 200)
          </div>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleCopyLogs}
              disabled={logs.length === 0}
              className="text-xs gap-1.5 cursor-pointer"
            >
              {copied ? (
                <>
                  <Check className="size-3.5 text-emerald-500" />
                  <span>Copied!</span>
                </>
              ) : (
                <>
                  <Copy className="size-3.5" />
                  <span>Copy All Logs</span>
                </>
              )}
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={() => onOpenChange(false)}
              className="text-xs cursor-pointer"
            >
              Close
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
