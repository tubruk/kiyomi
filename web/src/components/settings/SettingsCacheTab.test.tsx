import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SettingsCacheTab } from './SettingsCacheTab';
import { api } from '../../api/client';
import { clearQueryPersistence } from '../../lib/persister';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const mockToast = vi.fn();
vi.mock('../../context/ToastContext', () => ({
  useToast: () => ({
    showToast: mockToast,
  }),
}));

vi.mock('../../api/client', () => ({
  api: {
    getCacheStats: vi.fn(),
    clearCache: vi.fn(),
  },
}));

vi.mock('../../lib/persister', () => ({
  clearQueryPersistence: vi.fn(),
}));

describe('SettingsCacheTab', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    vi.mocked(api.getCacheStats).mockResolvedValue({
      size_bytes: 10485760, // 10 MB
      item_count: 250,
    });
  });

  const renderComponent = () => {
    return render(
      <QueryClientProvider client={queryClient}>
        <SettingsCacheTab />
      </QueryClientProvider>
    );
  };

  it('renders cache storage statistics correctly', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('10 MB')).toBeInTheDocument();
      expect(screen.getByText('250')).toBeInTheDocument();
    });

    expect(screen.getByText('Image Cache')).toBeInTheDocument();
    expect(screen.getByText('Offline Query Cache')).toBeInTheDocument();
  });

  it('refetches cache stats when Refresh button is clicked', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('10 MB')).toBeInTheDocument();
    });

    const refreshBtn = screen.getByRole('button', { name: /refresh/i });
    fireEvent.click(refreshBtn);

    await waitFor(() => {
      expect(api.getCacheStats).toHaveBeenCalledTimes(2);
    });
  });

  it('opens confirmation dialog and clears image cache on confirm', async () => {
    vi.mocked(api.clearCache).mockResolvedValue(undefined as any);
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('10 MB')).toBeInTheDocument();
    });

    const clearBtn = screen.getByRole('button', { name: /clear image cache/i });
    fireEvent.click(clearBtn);

    // Dialog title
    expect(screen.getAllByText('Clear Image Cache').length).toBeGreaterThanOrEqual(2);
    expect(
      screen.getByText(/Are you sure you want to clear the local image cache/i)
    ).toBeInTheDocument();

    // Confirm button inside dialog
    const confirmBtn = screen.getByRole('button', { name: 'Clear Cache' });
    fireEvent.click(confirmBtn);

    await waitFor(() => {
      expect(api.clearCache).toHaveBeenCalledTimes(1);
      expect(mockToast).toHaveBeenCalledWith('Image cache cleared successfully', 'success');
    });
  });

  it('opens confirmation dialog and clears offline query cache on confirm', async () => {
    vi.mocked(clearQueryPersistence).mockResolvedValue(undefined);
    renderComponent();

    const clearOfflineBtn = screen.getByRole('button', { name: /clear offline cache/i });
    fireEvent.click(clearOfflineBtn);

    expect(screen.getAllByText('Clear Offline Query Cache').length).toBeGreaterThanOrEqual(1);
    expect(
      screen.getByText(/Are you sure you want to clear the offline query cache/i)
    ).toBeInTheDocument();

    const confirmBtn = screen.getByRole('button', { name: 'Clear Offline Cache' });
    fireEvent.click(confirmBtn);

    await waitFor(() => {
      expect(clearQueryPersistence).toHaveBeenCalledWith(queryClient);
      expect(mockToast).toHaveBeenCalledWith('Offline query cache cleared successfully', 'success');
    });
  });
});
