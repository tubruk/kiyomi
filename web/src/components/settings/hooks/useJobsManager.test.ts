import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { useJobsManager } from './useJobsManager';
import { api } from '../../../api/client';
import { Job } from '../../../types/api';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

const mockToast = vi.fn();
vi.mock('../../../context/ToastContext', () => ({
  useToast: () => ({
    showToast: mockToast,
  }),
}));

vi.mock('../../../api/client', () => ({
  api: {
    getJobs: vi.fn(),
    cancelJob: vi.fn(),
    deleteJob: vi.fn(),
    cleanupJobs: vi.fn(),
  },
}));

const mockJobs: Job[] = [
  {
    id: 'job-1',
    type: 'chapter_download',
    status: 'running',
    retries: 0,
    max_retries: 3,
    created_at: '2026-08-30T10:00:00Z',
    updated_at: '2026-08-30T10:00:00Z',
  },
  {
    id: 'job-2',
    type: 'chapter_download',
    status: 'pending',
    retries: 0,
    max_retries: 3,
    created_at: '2026-08-30T10:05:00Z',
    updated_at: '2026-08-30T10:05:00Z',
  },
  {
    id: 'job-3',
    type: 'metadata_refresh',
    status: 'completed',
    retries: 0,
    max_retries: 3,
    created_at: '2026-08-30T09:00:00Z',
    updated_at: '2026-08-30T09:00:00Z',
  },
  {
    id: 'job-4',
    type: 'thumbnail_generate',
    status: 'failed',
    error: 'Disk full',
    retries: 3,
    max_retries: 3,
    created_at: '2026-08-30T08:00:00Z',
    updated_at: '2026-08-30T08:00:00Z',
  },
];

describe('useJobsManager', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.mocked(api.getJobs).mockResolvedValue(mockJobs);
  });

  const createWrapper = () => {
    return ({ children }: { children: React.ReactNode }) =>
      React.createElement(QueryClientProvider, { client: queryClient }, children);
  };

  it('fetches jobs and computes statistics accurately', async () => {
    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.allJobs).toHaveLength(4);
    expect(result.current.activeCount).toBe(1);
    expect(result.current.pendingCount).toBe(1);
    expect(result.current.completedCount).toBe(1);
    expect(result.current.failedCount).toBe(1);
    expect(result.current.stats).toEqual({
      active: 1,
      pending: 1,
      completed: 1,
      failed: 1,
      activeCount: 1,
      pendingCount: 1,
      completedCount: 1,
      failedCount: 1,
    });
  });

  it('sorts filtered jobs descending by created_at', async () => {
    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    // job-2 (10:05), job-1 (10:00), job-3 (09:00), job-4 (08:00)
    expect(result.current.filteredJobs.map((j) => j.id)).toEqual([
      'job-2',
      'job-1',
      'job-3',
      'job-4',
    ]);
  });

  it('filters jobs by statusFilter', async () => {
    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    act(() => {
      result.current.setStatusFilter('running');
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].id).toBe('job-1');

    act(() => {
      result.current.setStatusFilter('completed');
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].id).toBe('job-3');
  });

  it('filters jobs by typeFilter case-insensitively', async () => {
    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    act(() => {
      result.current.setTypeFilter('DOWNLOAD');
    });

    expect(result.current.filteredJobs).toHaveLength(2);
    expect(result.current.filteredJobs.map((j) => j.id)).toEqual(['job-2', 'job-1']);

    act(() => {
      result.current.setTypeFilter('metadata');
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].id).toBe('job-3');
  });

  it('switches viewMode between hierarchical and flat', async () => {
    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(api.getJobs).toHaveBeenCalledWith(undefined);

    act(() => {
      result.current.setViewMode('flat');
    });

    await waitFor(() => {
      expect(api.getJobs).toHaveBeenCalledWith({ all: true });
    });
  });

  it('handles cancelJob mutation successfully', async () => {
    vi.mocked(api.cancelJob).mockResolvedValue(undefined as any);
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await act(async () => {
      result.current.cancelJob('job-1');
    });

    await waitFor(() => {
      expect(api.cancelJob).toHaveBeenCalledWith('job-1');
      expect(mockToast).toHaveBeenCalledWith('Job cancelled successfully', 'success');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['jobs'] });
    });
  });

  it('handles deleteJob mutation successfully', async () => {
    vi.mocked(api.deleteJob).mockResolvedValue(undefined as any);
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await act(async () => {
      result.current.deleteJob('job-3');
    });

    await waitFor(() => {
      expect(api.deleteJob).toHaveBeenCalledWith('job-3');
      expect(mockToast).toHaveBeenCalledWith('Job removed successfully', 'success');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['jobs'] });
    });
  });

  it('handles cleanupJobs mutation successfully', async () => {
    vi.mocked(api.cleanupJobs).mockResolvedValue({ deleted: 5 } as any);
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const { result } = renderHook(() => useJobsManager(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await act(async () => {
      result.current.cleanupJobs('completed');
    });

    await waitFor(() => {
      expect(api.cleanupJobs).toHaveBeenCalledWith('completed');
      expect(mockToast).toHaveBeenCalledWith('Removed 5 completed jobs', 'success');
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['jobs'] });
    });
  });
});
