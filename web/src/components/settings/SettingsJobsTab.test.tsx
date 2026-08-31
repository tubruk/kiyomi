import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SettingsJobsTab } from './SettingsJobsTab';
import { api } from '../../api/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { Job } from '../../types/api';

const mockToast = vi.fn();
vi.mock('../../context/ToastContext', () => ({
  useToast: () => ({
    showToast: mockToast,
  }),
}));

vi.mock('../../api/client', () => ({
  api: {
    getJobs: vi.fn(),
    cancelJob: vi.fn(),
    deleteJob: vi.fn(),
    cleanupJobs: vi.fn(),
  },
}));

const mockJobsList: Job[] = [
  {
    id: 'job-101',
    type: 'chapter_download',
    status: 'running',
    concurrency_group: 'group-downloads',
    metadata: { manga_id: 'manga-abc', chapter_id: 'ch-1' },
    retries: 0,
    max_retries: 3,
    created_at: '2026-08-30T10:00:00Z',
    updated_at: '2026-08-30T10:01:00Z',
    started_at: '2026-08-30T10:00:30Z',
  },
  {
    id: 'job-102',
    type: 'metadata_sync',
    status: 'pending',
    concurrency_group: 'group-sync',
    retries: 0,
    max_retries: 2,
    created_at: '2026-08-30T09:30:00Z',
    updated_at: '2026-08-30T09:30:00Z',
  },
  {
    id: 'job-103',
    type: 'chapter_download',
    status: 'completed',
    retries: 1,
    max_retries: 3,
    created_at: '2026-08-30T09:00:00Z',
    updated_at: '2026-08-30T09:10:00Z',
    completed_at: '2026-08-30T09:10:00Z',
  },
  {
    id: 'job-104',
    type: 'library_scan',
    status: 'failed',
    error: 'Disk I/O error',
    retries: 3,
    max_retries: 3,
    created_at: '2026-08-30T08:00:00Z',
    updated_at: '2026-08-30T08:05:00Z',
  },
];

describe('SettingsJobsTab', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    vi.mocked(api.getJobs).mockResolvedValue(mockJobsList);
    vi.mocked(api.cancelJob).mockResolvedValue({ status: 'cancelled' } as any);
    vi.mocked(api.deleteJob).mockResolvedValue({ status: 'deleted' } as any);
    vi.mocked(api.cleanupJobs).mockResolvedValue({ deleted: 1 } as any);
  });

  const renderComponent = () => {
    return render(
      <QueryClientProvider client={queryClient}>
        <SettingsJobsTab />
      </QueryClientProvider>
    );
  };

  it('renders stats badges with correct counts', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
    });

    // Check stats row
    expect(screen.getByText('Active')).toBeInTheDocument();
    expect(screen.getAllByText('Pending').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Completed').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Failed').length).toBeGreaterThanOrEqual(1);

    // 1 active (running), 1 pending, 1 completed, 1 failed
    const numbers = screen.getAllByText('1');
    expect(numbers.length).toBeGreaterThanOrEqual(4);
  });

  it('renders table headers and job rows for all jobs', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
      expect(screen.getByText('job-102')).toBeInTheDocument();
      expect(screen.getByText('job-103')).toBeInTheDocument();
      expect(screen.getByText('job-104')).toBeInTheDocument();
    });

    expect(screen.getByText('Job ID')).toBeInTheDocument();
    expect(screen.getByText('Type')).toBeInTheDocument();
    expect(screen.getByText('Concurrency Group')).toBeInTheDocument();
    expect(screen.getByText('Metadata')).toBeInTheDocument();
    expect(screen.getByText('Retries')).toBeInTheDocument();
    expect(screen.getByText('Timestamps')).toBeInTheDocument();
  });

  it('filters jobs by Type search input', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
    });

    const typeInput = screen.getByPlaceholderText('Filter by job type...');
    fireEvent.change(typeInput, { target: { value: 'scan' } });

    expect(screen.getByText('job-104')).toBeInTheDocument();
    expect(screen.queryByText('job-101')).not.toBeInTheDocument();
    expect(screen.queryByText('job-102')).not.toBeInTheDocument();
    expect(screen.queryByText('job-103')).not.toBeInTheDocument();
  });

  it('filters jobs by Status selector', async () => {
    const user = userEvent.setup();
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
    });

    const statusTrigger = screen.getByText('All Statuses');
    await user.click(statusTrigger);

    const completedOption = await screen.findByRole('option', { name: 'Completed' });
    await user.click(completedOption);

    await waitFor(() => {
      expect(screen.getByText('job-103')).toBeInTheDocument();
      expect(screen.queryByText('job-101')).not.toBeInTheDocument();
      expect(screen.queryByText('job-102')).not.toBeInTheDocument();
      expect(screen.queryByText('job-104')).not.toBeInTheDocument();
    });
  });

  it('switches view mode between Hierarchical and Flat', async () => {
    const user = userEvent.setup();
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
    });

    const viewModeTrigger = screen.getByText('Hierarchical (Default)');
    await user.click(viewModeTrigger);

    const flatOption = await screen.findByRole('option', { name: 'Flat (All jobs)' });
    await user.click(flatOption);

    await waitFor(() => {
      expect(api.getJobs).toHaveBeenCalledWith({ all: true });
    });
  });

  it('calls cleanupJobsMutation when Clear Completed button is clicked', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
    });

    const clearCompletedBtn = screen.getByRole('button', { name: /clear completed/i });
    expect(clearCompletedBtn).toBeEnabled();

    fireEvent.click(clearCompletedBtn);

    await waitFor(() => {
      expect(api.cleanupJobs).toHaveBeenCalledWith('completed');
      expect(mockToast).toHaveBeenCalledWith('Removed 1 completed job', 'success');
    });
  });

  it('disables Clear Completed button when there are no completed jobs', async () => {
    const noCompletedJobs = mockJobsList.filter((j) => j.status !== 'completed');
    vi.mocked(api.getJobs).mockResolvedValue(noCompletedJobs);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
    });

    const clearCompletedBtn = screen.getByRole('button', { name: /clear completed/i });
    expect(clearCompletedBtn).toBeDisabled();
  });

  it('refetches jobs when Refresh button is clicked', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-101')).toBeInTheDocument();
    });

    const refreshBtn = screen.getByRole('button', { name: /refresh/i });
    fireEvent.click(refreshBtn);

    await waitFor(() => {
      expect(api.getJobs).toHaveBeenCalledTimes(2);
    });
  });

  it('renders empty state message when no jobs match', async () => {
    vi.mocked(api.getJobs).mockResolvedValue([]);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('No jobs found.')).toBeInTheDocument();
    });
  });

  it('cancels a pending job from JobRow action button', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-102')).toBeInTheDocument();
    });

    const row102 = screen.getByText('job-102').closest('tr')!;
    const cancelBtn = within(row102).getByRole('button', { name: 'Cancel' });
    fireEvent.click(cancelBtn);

    await waitFor(() => {
      expect(api.cancelJob).toHaveBeenCalledWith('job-102');
      expect(mockToast).toHaveBeenCalledWith('Job cancelled successfully', 'success');
    });
  });

  it('deletes a completed job from JobRow action button', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('job-103')).toBeInTheDocument();
    });

    const row103 = screen.getByText('job-103').closest('tr')!;
    const removeBtn = within(row103).getByRole('button', { name: /remove/i });
    fireEvent.click(removeBtn);

    await waitFor(() => {
      expect(api.deleteJob).toHaveBeenCalledWith('job-103');
      expect(mockToast).toHaveBeenCalledWith('Job removed successfully', 'success');
    });
  });
});
