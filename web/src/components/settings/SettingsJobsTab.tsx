import React from 'react';
import { Clock, RefreshCw, Trash2, Loader2 } from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../lib/queryKeys';
import { useJobsManager } from './hooks/useJobsManager';
import { JobRow } from './JobRow';
import { Button } from '../ui/button';
import { Card } from '../ui/card';
import { Input } from '../ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';

export const SettingsJobsTab: React.FC = () => {
  const queryClient = useQueryClient();
  const {
    filteredJobs,
    isLoadingJobs,
    stats,
    viewMode,
    setViewMode,
    statusFilter,
    setStatusFilter,
    typeFilter,
    setTypeFilter,
    cancelJobMutation,
    deleteJobMutation,
    cleanupJobsMutation,
  } = useJobsManager();

  return (
    <div className="space-y-6">
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
          <Select value={viewMode} onValueChange={(val) => setViewMode(val as 'hierarchical' | 'flat')}>
            <SelectTrigger>
              <SelectValue placeholder="View mode">
                {viewMode === 'flat' ? 'Flat (All jobs)' : 'Hierarchical (Default)'}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="hierarchical">Hierarchical (Default)</SelectItem>
              <SelectItem value="flat">Flat (All jobs)</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="w-full sm:w-48">
          <Select value={statusFilter} onValueChange={(val) => setStatusFilter(val || 'all')}>
            <SelectTrigger>
              <SelectValue placeholder="Filter by status">
                {statusFilter === 'pending'
                  ? 'Pending'
                  : statusFilter === 'running'
                  ? 'Running'
                  : statusFilter === 'completed'
                  ? 'Completed'
                  : statusFilter === 'failed'
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
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
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
    </div>
  );
};
