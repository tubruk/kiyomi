import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SettingsPluginsTab } from './SettingsPluginsTab';
import { api } from '../../api/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PluginItem, ProviderCollision } from '../../types/api';

const mockToast = vi.fn();
vi.mock('../../context/ToastContext', () => ({
  useToast: () => ({
    showToast: mockToast,
  }),
}));

vi.mock('../../api/client', () => ({
  api: {
    getPlugins: vi.fn(),
    getCollisions: vi.fn(),
    reloadPlugins: vi.fn(),
    getPluginLogs: vi.fn(),
    updatePluginConfig: vi.fn(),
  },
}));

const mockPlugins: PluginItem[] = [
  {
    pluginId: 'mangadex-plugin',
    pluginName: 'MangaDex Provider',
    pluginVersion: '1.2.0',
    sdkVersion: '1.0.0',
    sdkCompatible: true,
    executablePath: '/usr/local/bin/mangadex-plugin',
    pid: 12345,
    state: 'running',
    loadedAt: '2026-08-30T08:00:00Z',
    providers: [
      {
        id: 'mangadex',
        name: 'MangaDex',
        description: 'Official MangaDex v5 API client',
        capabilities: ['content', 'metadata'],
        settingsSchema: [
          {
            key: 'api_token',
            label: 'API Token',
            type: 'secret',
            description: 'Personal access token for MangaDex',
          },
        ],
      },
    ],
    pluginSettingsSchema: [
      {
        key: 'request_timeout',
        label: 'Request Timeout (s)',
        type: 'number',
        defaultValue: '30',
      },
    ],
    globalConfig: { request_timeout: '30' },
  },
  {
    pluginId: 'comic-extra-plugin',
    pluginName: 'ComicExtra Provider',
    pluginVersion: '0.9.1',
    sdkVersion: '1.0.0',
    sdkCompatible: false,
    executablePath: '/usr/local/bin/comicextra-plugin',
    pid: 0,
    state: 'error',
    errorMessage: 'Incompatible SDK version',
    loadedAt: '2026-08-30T08:05:00Z',
    providers: [
      {
        id: 'comicextra',
        name: 'ComicExtra',
        capabilities: ['content'],
      },
    ],
  },
];

const mockCollisions: ProviderCollision[] = [
  {
    providerId: 'mangadex',
    selected: 'mangadex-plugin',
    candidates: [
      {
        pluginId: 'mangadex-plugin',
        version: '1.2.0',
        isBuiltIn: false,
        selected: true,
      },
      {
        pluginId: 'builtin',
        version: '1.0.0',
        isBuiltIn: true,
        selected: false,
      },
    ],
  },
];

describe('SettingsPluginsTab', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    vi.mocked(api.getPlugins).mockResolvedValue(mockPlugins);
    vi.mocked(api.getCollisions).mockResolvedValue([]);
    vi.mocked(api.getPluginLogs).mockResolvedValue([]);
    vi.mocked(api.reloadPlugins).mockResolvedValue({
      status: 'ok',
      message: 'Reloaded 2 plugin(s)',
      reloadedPlugins: ['mangadex-plugin', 'comic-extra-plugin'],
      activeProviders: 2,
    });
  });

  const renderComponent = () => {
    return render(
      <QueryClientProvider client={queryClient}>
        <SettingsPluginsTab />
      </QueryClientProvider>
    );
  };

  it('renders plugin cards grid with fetched plugins and active count badge', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider')).toBeInTheDocument();
      expect(screen.getByText('ComicExtra Provider')).toBeInTheDocument();
    });

    expect(screen.getByText('2 active')).toBeInTheDocument();
    expect(screen.getByText('v1.2.0')).toBeInTheDocument();
    expect(screen.getByText('v0.9.1')).toBeInTheDocument();
    expect(screen.getByText('SDK v1.0.0')).toBeInTheDocument();
    expect(screen.getByText('Incompatible SDK')).toBeInTheDocument();
  });

  it('renders empty state card when no plugins are installed', async () => {
    vi.mocked(api.getPlugins).mockResolvedValue([]);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('No Plugins Installed')).toBeInTheDocument();
    });

    expect(screen.getByText('0 active')).toBeInTheDocument();
    expect(screen.getByText(/Place Kiyomi provider plugins inside your configured plugin directory/i)).toBeInTheDocument();
  });

  it('triggers reload mutation and displays success toast when Reload button is clicked', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider')).toBeInTheDocument();
    });

    const reloadBtn = screen.getByRole('button', { name: /reload/i });
    fireEvent.click(reloadBtn);

    await waitFor(() => {
      expect(api.reloadPlugins).toHaveBeenCalledTimes(1);
      expect(mockToast).toHaveBeenCalledWith('Reloaded 2 plugin(s)', 'success');
    });
  });

  it('handles reload error and opens ErrorDetailsModal', async () => {
    const error = new Error('Plugin reload failed') as any;
    error.details = 'Subprocess exited with status 1';
    vi.mocked(api.reloadPlugins).mockRejectedValue(error);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider')).toBeInTheDocument();
    });

    const reloadBtn = screen.getByRole('button', { name: /reload/i });
    fireEvent.click(reloadBtn);

    await waitFor(() => {
      expect(mockToast).toHaveBeenCalledWith(
        'Plugin reload failed',
        'error',
        'Subprocess exited with status 1'
      );
      expect(screen.getByText('Failed to reload plugins')).toBeInTheDocument();
    });
  });

  it('renders CollisionResolutionAlert when collisions exist', async () => {
    vi.mocked(api.getCollisions).mockResolvedValue(mockCollisions);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Provider ID Collision Detected')).toBeInTheDocument();
      expect(screen.getByText('1 conflict')).toBeInTheDocument();
    });
  });

  it('does not render CollisionResolutionAlert when collisions array is empty', async () => {
    vi.mocked(api.getCollisions).mockResolvedValue([]);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider')).toBeInTheDocument();
    });

    expect(screen.queryByText('Provider ID Collision Detected')).not.toBeInTheDocument();
  });

  it('opens ScopedSettingsModal when Settings button is clicked on a plugin card', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider')).toBeInTheDocument();
    });

    const settingsButtons = screen.getAllByRole('button', { name: /settings/i });
    fireEvent.click(settingsButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('Configure MangaDex Provider')).toBeInTheDocument();
      expect(screen.getByText('Plugin Global')).toBeInTheDocument();
      expect(screen.getByText('MangaDex')).toBeInTheDocument();
    });
  });

  it('opens DiagnosticLogsModal when Logs button is clicked on a plugin card', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider')).toBeInTheDocument();
    });

    const logsButtons = screen.getAllByRole('button', { name: /logs/i });
    fireEvent.click(logsButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('MangaDex Provider Diagnostics')).toBeInTheDocument();
    });
  });
});
