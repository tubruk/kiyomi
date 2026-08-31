import React, { useState } from 'react';
import { Loader2, ChevronDown, ChevronRight, Trash2 } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { jobsQueryOptions } from '../../lib/queryOptions';
import { Job } from '../../types/api';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';

export interface JobRowProps {
  job: Job;
  onCancel: (id: string) => void;
  isCancelling: boolean;
  onDelete: (id: string) => void;
  isDeleting: boolean;
  depth?: number;
}

export const JobRow: React.FC<JobRowProps> = ({
  job,
  onCancel,
  isCancelling,
  onDelete,
  isDeleting,
  depth = 0,
}) => {
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
