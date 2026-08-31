import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SettingsAboutTab } from './SettingsAboutTab';
import { api } from '../../api/client';
import { usePWAInstall } from '../../hooks/usePWAInstall';
import { clearQueryPersistence } from '../../lib/persister';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { AppInfo } from '../../types/api';

const mockToast = vi.fn();
vi.mock('../../context/ToastContext', () => ({
  useToast: () => ({
    showToast: mockToast,
  }),
}));

vi.mock('../../api/client', () => ({
  api: {
    getInfo: vi.fn(),
  },
}));

vi.mock('../../hooks/usePWAInstall', () => ({
  usePWAInstall: vi.fn(),
}));

vi.mock('../../lib/persister', () => ({
  clearQueryPersistence: vi.fn(),
}));

const mockAppInfo: AppInfo = {
  app: 'Kiyomi Reader',
  version: '1.5.0',
  commit: 'f7a8b9c',
  build_time: '2026-08-30 14:22:00',
  go_version: 'go1.23.1',
};

describe('SettingsAboutTab', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    vi.mocked(api.getInfo).mockResolvedValue(mockAppInfo);
    vi.mocked(usePWAInstall).mockReturnValue({
      isInstallable: false,
      isInstalled: false,
      install: vi.fn().mockResolvedValue(true),
    });
    vi.mocked(clearQueryPersistence).mockResolvedValue(undefined);
  });

  const renderComponent = () => {
    return render(
      <QueryClientProvider client={queryClient}>
        <SettingsAboutTab />
      </QueryClientProvider>
    );
  };

  it('renders app metadata correctly from infoQuery', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Kiyomi Reader')).toBeInTheDocument();
      expect(screen.getByText('v1.5.0')).toBeInTheDocument();
    });

    expect(screen.getByText('1.5.0')).toBeInTheDocument();
    expect(screen.getByText('f7a8b9c')).toBeInTheDocument();
    expect(screen.getByText('2026-08-30 14:22:00')).toBeInTheDocument();
    expect(screen.getByText('go1.23.1')).toBeInTheDocument();
    expect(screen.getByText('Version')).toBeInTheDocument();
    expect(screen.getByText('Commit')).toBeInTheDocument();
    expect(screen.getByText('Build time')).toBeInTheDocument();
    expect(screen.getByText('Go version')).toBeInTheDocument();
  });

  it('renders fallback placeholders when infoQuery returns null or empty fields', async () => {
    vi.mocked(api.getInfo).mockResolvedValue({} as any);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Kiyomi')).toBeInTheDocument();
    });

    const dashes = screen.getAllByText('—');
    expect(dashes.length).toBeGreaterThanOrEqual(4);
  });

  it('renders PWA install prompt when app is installable and triggers install', async () => {
    const mockInstall = vi.fn().mockResolvedValue(true);
    vi.mocked(usePWAInstall).mockReturnValue({
      isInstallable: true,
      isInstalled: false,
      install: mockInstall,
    });

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Installable PWA')).toBeInTheDocument();
      expect(screen.getByText('Ready to install on this device')).toBeInTheDocument();
    });

    const installBtn = screen.getByRole('button', { name: /install app/i });
    expect(installBtn).toBeInTheDocument();

    fireEvent.click(installBtn);

    await waitFor(() => {
      expect(mockInstall).toHaveBeenCalledTimes(1);
      expect(mockToast).toHaveBeenCalledWith('Kiyomi installed successfully!', 'success');
    });
  });

  it('renders installed status when app is running as installed PWA', async () => {
    vi.mocked(usePWAInstall).mockReturnValue({
      isInstallable: false,
      isInstalled: true,
      install: vi.fn(),
    });

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Installed App')).toBeInTheDocument();
      expect(
        screen.getByText('Running as standalone installed application')
      ).toBeInTheDocument();
    });

    expect(screen.getAllByText('Installed').length).toBeGreaterThanOrEqual(1);
  });

  it('renders web browser status when not installable and not installed', async () => {
    vi.mocked(usePWAInstall).mockReturnValue({
      isInstallable: false,
      isInstalled: false,
      install: vi.fn(),
    });

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Running in web browser')).toBeInTheDocument();
      expect(screen.getByText('Web Browser')).toBeInTheDocument();
    });
  });

  it('opens confirmation dialog and clears offline query cache on confirm', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Offline Query Cache')).toBeInTheDocument();
    });

    const clearBtn = screen.getByRole('button', { name: /clear cache/i });
    fireEvent.click(clearBtn);

    expect(
      screen.getAllByText('Clear Offline Query Cache').length
    ).toBeGreaterThanOrEqual(1);
    expect(
      screen.getByText(/Are you sure you want to clear the offline query cache/i)
    ).toBeInTheDocument();

    const confirmBtn = screen.getByRole('button', { name: 'Clear Offline Cache' });
    fireEvent.click(confirmBtn);

    await waitFor(() => {
      expect(clearQueryPersistence).toHaveBeenCalledWith(queryClient);
      expect(mockToast).toHaveBeenCalledWith(
        'Offline query cache cleared successfully',
        'success'
      );
    });
  });

  it('handles error when clearing offline query cache fails', async () => {
    vi.mocked(clearQueryPersistence).mockRejectedValue(new Error('IndexedDB error'));

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Offline Query Cache')).toBeInTheDocument();
    });

    const clearBtn = screen.getByRole('button', { name: /clear cache/i });
    fireEvent.click(clearBtn);

    const confirmBtn = screen.getByRole('button', { name: 'Clear Offline Cache' });
    fireEvent.click(confirmBtn);

    await waitFor(() => {
      expect(mockToast).toHaveBeenCalledWith(
        'Failed to clear offline cache',
        'error',
        'IndexedDB error'
      );
    });
  });

  it('closes dialog without clearing cache when Cancel is clicked', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Offline Query Cache')).toBeInTheDocument();
    });

    const clearBtn = screen.getByRole('button', { name: /clear cache/i });
    fireEvent.click(clearBtn);

    const cancelBtn = screen.getByRole('button', { name: 'Cancel' });
    fireEvent.click(cancelBtn);

    expect(clearQueryPersistence).not.toHaveBeenCalled();
  });
});
