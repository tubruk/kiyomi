import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PluginLogsModal } from './DiagnosticLogsModal';
import { api } from '../../api/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PluginItem, PluginLogEntry } from '../../types/api';

vi.mock('../../api/client', () => ({
  api: {
    getPluginLogs: vi.fn(),
  },
}));

const mockPlugin: PluginItem = {
  pluginId: 'mangadex-plugin',
  pluginName: 'MangaDex Provider',
  pluginVersion: '1.2.0',
  sdkVersion: '1.0.0',
  sdkCompatible: true,
  executablePath: '/usr/local/bin/mangadex-plugin',
  pid: 12345,
  state: 'running',
  loadedAt: '2026-08-30T08:00:00Z',
  providers: [],
};

const mockLogs: PluginLogEntry[] = [
  {
    timestamp: '2026-08-30T10:00:00.123Z',
    level: 'INFO',
    message: 'Plugin subprocess initialized successfully',
  },
  {
    timestamp: '2026-08-30T10:01:00.456Z',
    level: 'WARN',
    message: 'Rate limit approaching threshold',
    fields: { remaining: 2, limit: 10 },
  },
  {
    timestamp: '2026-08-30T10:02:00.789Z',
    level: 'ERROR',
    message: 'Failed to fetch chapter manifest',
    fields: { error_code: 503 },
  },
  {
    timestamp: '2026-08-30T10:03:00.000Z',
    level: 'DEBUG',
    message: 'Executing raw query',
  },
];

describe('PluginLogsModal', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    vi.mocked(api.getPluginLogs).mockResolvedValue(mockLogs);

    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
  });

  const renderModal = (
    open = true,
    plugin: PluginItem | null = mockPlugin,
    onOpenChange = vi.fn()
  ) => {
    return render(
      <QueryClientProvider client={queryClient}>
        <PluginLogsModal
          open={open}
          onOpenChange={onOpenChange}
          plugin={plugin}
        />
      </QueryClientProvider>
    );
  };

  it('renders null when plugin is null', () => {
    const { container } = renderModal(true, null);
    expect(container.firstChild).toBeNull();
  });

  it('renders modal header, plugin details, and log entries', async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /MangaDex Provider Diagnostics/i })).toBeInTheDocument();
      expect(screen.getByText('Plugin subprocess initialized successfully')).toBeInTheDocument();
    });

    expect(screen.getByText('mangadex-plugin')).toBeInTheDocument();
    expect(screen.getByText('PID: 12345')).toBeInTheDocument();
    expect(screen.getByText('/usr/local/bin/mangadex-plugin')).toBeInTheDocument();
    expect(screen.getByText('SDK v1.0.0')).toBeInTheDocument();

    // Log messages
    expect(screen.getByText('Rate limit approaching threshold')).toBeInTheDocument();
    expect(screen.getByText('Failed to fetch chapter manifest')).toBeInTheDocument();
    expect(screen.getByText('Executing raw query')).toBeInTheDocument();

    // Footer entry count
    expect(screen.getByText(/Showing/)).toBeInTheDocument();
  });

  it('filters logs by log level buttons', async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText('Plugin subprocess initialized successfully')).toBeInTheDocument();
    });

    // Filter by ERROR
    const errorLevelBtn = screen.getByRole('button', { name: 'ERROR' });
    fireEvent.click(errorLevelBtn);

    expect(screen.getByText('Failed to fetch chapter manifest')).toBeInTheDocument();
    expect(screen.queryByText('Plugin subprocess initialized successfully')).not.toBeInTheDocument();
    expect(screen.queryByText('Rate limit approaching threshold')).not.toBeInTheDocument();
    expect(screen.queryByText('Executing raw query')).not.toBeInTheDocument();

    // Switch back to ALL
    const allLevelBtn = screen.getByRole('button', { name: 'ALL' });
    fireEvent.click(allLevelBtn);

    expect(screen.getByText('Plugin subprocess initialized successfully')).toBeInTheDocument();
    expect(screen.getByText('Rate limit approaching threshold')).toBeInTheDocument();
  });

  it('filters logs by text search query', async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText('Plugin subprocess initialized successfully')).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText('Filter logs by message or field...');
    fireEvent.change(searchInput, { target: { value: 'manifest' } });

    expect(screen.getByText('Failed to fetch chapter manifest')).toBeInTheDocument();
    expect(screen.queryByText('Plugin subprocess initialized successfully')).not.toBeInTheDocument();
  });

  it('renders empty message when no logs match filter', async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText('Plugin subprocess initialized successfully')).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText('Filter logs by message or field...');
    fireEvent.change(searchInput, { target: { value: 'nonexistent-log-entry' } });

    expect(
      screen.getByText('No logs match your current search/level filter.')
    ).toBeInTheDocument();
  });

  it('renders initial empty state when no diagnostic logs captured', async () => {
    vi.mocked(api.getPluginLogs).mockResolvedValue([]);

    renderModal();

    await waitFor(() => {
      expect(
        screen.getByText('No diagnostic logs captured for this plugin subprocess yet.')
      ).toBeInTheDocument();
    });
  });

  it('toggles auto-refresh state on button click', async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText('Auto-refresh: ON')).toBeInTheDocument();
    });

    const autoRefreshBtn = screen.getByRole('button', { name: /auto-refresh: on/i });
    fireEvent.click(autoRefreshBtn);

    expect(screen.getByText('Auto-refresh: OFF')).toBeInTheDocument();
  });

  it('manually refetches logs on refresh button click', async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText('Plugin subprocess initialized successfully')).toBeInTheDocument();
    });

    const refreshBtn = screen.getByRole('button', { name: 'Refresh logs' });
    fireEvent.click(refreshBtn);

    await waitFor(() => {
      expect(api.getPluginLogs).toHaveBeenCalledTimes(2);
    });
  });

  it('copies all logs to clipboard when Copy All Logs button is clicked', async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText('Plugin subprocess initialized successfully')).toBeInTheDocument();
    });

    const copyBtn = screen.getByRole('button', { name: /copy all logs/i });
    fireEvent.click(copyBtn);

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledTimes(1);
      expect(screen.getByText('Copied!')).toBeInTheDocument();
    });
  });

  it('calls onOpenChange(false) when Close button is clicked', async () => {
    const onOpenChange = vi.fn();
    renderModal(true, mockPlugin, onOpenChange);

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider Diagnostics')).toBeInTheDocument();
    });

    const closeBtns = screen.getAllByRole('button', { name: /close/i });
    fireEvent.click(closeBtns[0]);

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
