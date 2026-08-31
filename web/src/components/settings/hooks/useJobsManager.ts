import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { jobsQueryOptions } from '../../../lib/queryOptions';
import { queryKeys } from '../../../lib/queryKeys';
import { api } from '../../../api/client';
import { useToast } from '../../../context/ToastContext';
import type { Job } from '../../../types/api';

export interface UseJobsManagerOptions {
  enabled?: boolean;
}

export interface JobsStats {
  active: number;
  pending: number;
  completed: number;
  failed: number;
  activeCount: number;
  pendingCount: number;
  completedCount: number;
  failedCount: number;
}

export function useJobsManager(options?: UseJobsManagerOptions) {
  const { enabled = true } = options ?? {};
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const [viewMode, setViewMode] = useState<'hierarchical' | 'flat'>('hierarchical');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [typeFilter, setTypeFilter] = useState<string>('');

  const jobFilter = useMemo(() => {
    if (viewMode === 'flat') {
      return { all: true };
    }
    return undefined;
  }, [viewMode]);

  const {
    data: allJobs = [],
    isLoading: isLoadingJobs,
    refetch,
  } = useQuery({
    ...jobsQueryOptions(jobFilter),
    enabled,
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

  const filteredJobs = useMemo(() => {
    return (allJobs as Job[])
      .filter((job) => {
        const matchStatus = statusFilter === 'all' || job.status === statusFilter;
        const matchType =
          typeFilter.trim() === '' || job.type.toLowerCase().includes(typeFilter.toLowerCase().trim());
        return matchStatus && matchType;
      })
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }, [allJobs, statusFilter, typeFilter]);

  const stats: JobsStats = useMemo(() => {
    let active = 0;
    let pending = 0;
    let completed = 0;
    let failed = 0;
    (allJobs as Job[]).forEach((job) => {
      if (job.status === 'running') active++;
      else if (job.status === 'pending') pending++;
      else if (job.status === 'completed') completed++;
      else if (job.status === 'failed') failed++;
    });
    return {
      active,
      pending,
      completed,
      failed,
      activeCount: active,
      pendingCount: pending,
      completedCount: completed,
      failedCount: failed,
    };
  }, [allJobs]);

  return {
    allJobs: allJobs as Job[],
    filteredJobs,
    isLoading: isLoadingJobs,
    isLoadingJobs,
    refetch,
    stats,
    activeCount: stats.active,
    pendingCount: stats.pending,
    completedCount: stats.completed,
    failedCount: stats.failed,
    viewMode,
    setViewMode,
    jobViewMode: viewMode,
    setJobViewMode: setViewMode,
    statusFilter,
    setStatusFilter,
    jobStatusFilter: statusFilter,
    setJobStatusFilter: setStatusFilter,
    typeFilter,
    setTypeFilter,
    jobTypeFilter: typeFilter,
    setJobTypeFilter: setTypeFilter,
    cancelJobMutation,
    deleteJobMutation,
    cleanupJobsMutation,
    cleanJobsMutation: cleanupJobsMutation,
    cancelJob: (id: string) => cancelJobMutation.mutate(id),
    deleteJob: (id: string) => deleteJobMutation.mutate(id),
    cleanupJobs: (status: string = 'completed') => cleanupJobsMutation.mutate(status),
    cleanJobs: (status: string = 'completed') => cleanupJobsMutation.mutate(status),
  };
}
