import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { JobRow } from './JobRow';
import { Job } from '../../types/api';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { api } from '../../api/client';

vi.mock('../../api/client', () => ({
  api: {
    getJobs: vi.fn(),
  },
}));

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <table>
        <tbody>{ui}</tbody>
      </table>
    </QueryClientProvider>
  );
}

const mockJob: Job = {
  id: 'job-12345678',
  type: 'chapter_download',
  status: 'running',
  concurrency_group: 'group-1',
  metadata: { manga_id: 'manga-1', priority: 'high' },
  retries: 1,
  max_retries: 3,
  created_at: '2026-08-30T10:00:00Z',
  updated_at: '2026-08-30T10:00:00Z',
  started_at: '2026-08-30T10:01:00Z',
};

describe('JobRow', () => {
  it('renders job details correctly', () => {
    renderWithClient(
      <JobRow
        job={mockJob}
        onCancel={vi.fn()}
        isCancelling={false}
        onDelete={vi.fn()}
        isDeleting={false}
      />
    );

    expect(screen.getByText('job-12345678')).toBeInTheDocument();
    expect(screen.getByText('chapter_download')).toBeInTheDocument();
    expect(screen.getByText('group-1')).toBeInTheDocument();
    expect(screen.getByText('manga_id=manga-1')).toBeInTheDocument();
    expect(screen.getByText('priority=high')).toBeInTheDocument();
    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.getByText('1 / 3')).toBeInTheDocument();
  });

  it('renders pending status badge and cancel button for pending jobs', () => {
    const pendingJob: Job = {
      ...mockJob,
      status: 'pending',
    };
    const handleCancel = vi.fn();

    renderWithClient(
      <JobRow
        job={pendingJob}
        onCancel={handleCancel}
        isCancelling={false}
        onDelete={vi.fn()}
        isDeleting={false}
      />
    );

    expect(screen.getByText('Pending')).toBeInTheDocument();
    const cancelBtn = screen.getByRole('button', { name: 'Cancel' });
    expect(cancelBtn).toBeInTheDocument();

    fireEvent.click(cancelBtn);
    expect(handleCancel).toHaveBeenCalledWith('job-12345678');
  });

  it('disables cancel button and shows "Cancelling..." when isCancelling is true', () => {
    const pendingJob: Job = {
      ...mockJob,
      status: 'pending',
    };

    renderWithClient(
      <JobRow
        job={pendingJob}
        onCancel={vi.fn()}
        isCancelling={true}
        onDelete={vi.fn()}
        isDeleting={false}
      />
    );

    const cancelBtn = screen.getByRole('button', { name: 'Cancelling...' });
    expect(cancelBtn).toBeDisabled();
  });

  it('renders completed status badge and remove button for completed jobs', () => {
    const completedJob: Job = {
      ...mockJob,
      status: 'completed',
      completed_at: '2026-08-30T10:05:00Z',
    };
    const handleDelete = vi.fn();

    renderWithClient(
      <JobRow
        job={completedJob}
        onCancel={vi.fn()}
        isCancelling={false}
        onDelete={handleDelete}
        isDeleting={false}
      />
    );

    expect(screen.getByText('Completed')).toBeInTheDocument();
    const removeBtn = screen.getByRole('button', { name: /remove/i });
    expect(removeBtn).toBeInTheDocument();

    fireEvent.click(removeBtn);
    expect(handleDelete).toHaveBeenCalledWith('job-12345678');
  });

  it('disables remove button and shows "Removing..." when isDeleting is true', () => {
    const completedJob: Job = {
      ...mockJob,
      status: 'completed',
    };

    renderWithClient(
      <JobRow
        job={completedJob}
        onCancel={vi.fn()}
        isCancelling={false}
        onDelete={vi.fn()}
        isDeleting={true}
      />
    );

    const removeBtn = screen.getByRole('button', { name: /removing.../i });
    expect(removeBtn).toBeDisabled();
  });

  it('renders failed status and toggles error details collapsible', () => {
    const failedJob: Job = {
      ...mockJob,
      status: 'failed',
      error: 'Network connection timeout while fetching chapter pages',
    };

    renderWithClient(
      <JobRow
        job={failedJob}
        onCancel={vi.fn()}
        isCancelling={false}
        onDelete={vi.fn()}
        isDeleting={false}
      />
    );

    expect(screen.getByText('Failed')).toBeInTheDocument();
    const showErrorBtn = screen.getByRole('button', { name: 'Show Error' });
    expect(showErrorBtn).toBeInTheDocument();
    expect(screen.queryByText(/Network connection timeout/i)).not.toBeInTheDocument();

    // Open error details
    fireEvent.click(showErrorBtn);
    expect(screen.getByText(/Network connection timeout/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Hide Error' })).toBeInTheDocument();

    // Close error details
    fireEvent.click(screen.getByRole('button', { name: 'Hide Error' }));
    expect(screen.queryByText(/Network connection timeout/i)).not.toBeInTheDocument();
  });

  it('handles expandable sub-jobs accordion and renders child jobs recursively', async () => {
    const parentJob: Job = {
      ...mockJob,
      id: 'parent-job-1',
      type: 'batch_download',
      child_count: 2,
    };

    const mockChildJobs: Job[] = [
      {
        id: 'child-job-1',
        type: 'download_page',
        status: 'completed',
        retries: 0,
        max_retries: 3,
        created_at: '2026-08-30T10:01:00Z',
        updated_at: '2026-08-30T10:01:00Z',
      },
      {
        id: 'child-job-2',
        type: 'download_page',
        status: 'running',
        retries: 0,
        max_retries: 3,
        created_at: '2026-08-30T10:02:00Z',
        updated_at: '2026-08-30T10:02:00Z',
      },
    ];

    vi.mocked(api.getJobs).mockResolvedValue(mockChildJobs);

    renderWithClient(
      <JobRow
        job={parentJob}
        onCancel={vi.fn()}
        isCancelling={false}
        onDelete={vi.fn()}
        isDeleting={false}
      />
    );

    const subJobsBtn = screen.getByRole('button', { name: /sub-jobs \(2\)/i });
    expect(subJobsBtn).toBeInTheDocument();

    // Click to expand sub-jobs
    fireEvent.click(subJobsBtn);

    await waitFor(() => {
      expect(api.getJobs).toHaveBeenCalledWith({ parent_id: 'parent-job-1' });
    });

    await waitFor(() => {
      expect(screen.getByText('child-job-1')).toBeInTheDocument();
      expect(screen.getByText('child-job-2')).toBeInTheDocument();
    });
  });
});
