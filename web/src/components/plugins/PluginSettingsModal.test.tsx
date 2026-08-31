import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PluginSettingsModal } from './ScopedSettingsModal';
import { api } from '../../api/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { PluginItem } from '../../types/api';

const mockToast = vi.fn();
vi.mock('../../context/ToastContext', () => ({
  useToast: () => ({
    showToast: mockToast,
  }),
  ToastProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock('../../api/client', () => ({
  api: {
    updatePluginConfig: vi.fn(),
  },
}));

const mockPluginWithSettings: PluginItem = {
  pluginId: 'mangadex-plugin',
  pluginName: 'MangaDex Provider',
  pluginVersion: '1.2.0',
  sdkVersion: '1.0.0',
  sdkCompatible: true,
  executablePath: '/usr/local/bin/mangadex-plugin',
  pid: 12345,
  state: 'running',
  loadedAt: '2026-08-30T08:00:00Z',
  pluginSettingsSchema: [
    {
      key: 'request_timeout',
      label: 'Request Timeout',
      type: 'number',
      description: 'HTTP timeout in seconds',
      defaultValue: '30',
    },
    {
      key: 'enable_debug',
      label: 'Debug Mode',
      type: 'boolean',
      description: 'Enable verbose diagnostics',
      defaultValue: 'false',
    },
  ],
  globalConfig: {
    request_timeout: '30',
    enable_debug: 'false',
  },
  providers: [
    {
      id: 'mangadex',
      name: 'MangaDex',
      description: 'Official MangaDex v5 client',
      capabilities: ['content', 'metadata'],
      settingsSchema: [
        {
          key: 'api_token',
          label: 'API Token',
          type: 'secret',
          description: 'Personal bearer token',
        },
        {
          key: 'image_quality',
          label: 'Image Quality',
          type: 'select',
          options: ['data', 'data-saver'],
          defaultValue: 'data',
        },
      ],
    },
  ],
  providerConfigs: {
    mangadex: {
      api_token: 'secret-token-123',
      image_quality: 'data',
    },
  },
};

const mockPluginWithoutSettings: PluginItem = {
  pluginId: 'simple-plugin',
  pluginName: 'Simple Plugin',
  pluginVersion: '1.0.0',
  sdkVersion: '1.0.0',
  sdkCompatible: true,
  executablePath: '/usr/local/bin/simple-plugin',
  pid: 23456,
  state: 'running',
  loadedAt: '2026-08-30T08:00:00Z',
  providers: [
    {
      id: 'simple-source',
      name: 'Simple Source',
      capabilities: ['content'],
    },
  ],
};

describe('PluginSettingsModal', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    vi.mocked(api.updatePluginConfig).mockResolvedValue(undefined as any);
  });

  const renderModal = (
    open = true,
    plugin: PluginItem | null = mockPluginWithSettings,
    onOpenChange = vi.fn()
  ) => {
    return render(
      <QueryClientProvider client={queryClient}>
        <PluginSettingsModal
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

  it('renders empty state when plugin has no configurable settings', () => {
    renderModal(true, mockPluginWithoutSettings);

    expect(screen.getByText('Configure Simple Plugin')).toBeInTheDocument();
    expect(screen.getByText('No Configurable Settings')).toBeInTheDocument();
    expect(
      screen.getByText(/This plugin does not declare any custom setting specifications or credentials/i)
    ).toBeInTheDocument();
  });

  it('renders global and provider settings with various field types', () => {
    renderModal();

    expect(screen.getByText('Configure MangaDex Provider')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /global/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /mangadex/i })).toBeInTheDocument();

    // Number field in active global tab
    expect(screen.getByPlaceholderText('30')).toHaveValue(30);
    expect(screen.getByText('request_timeout')).toBeInTheDocument();
    expect(screen.getByText('enable_debug')).toBeInTheDocument();
  });

  it('switches tabs and toggles secret visibility in provider settings', async () => {
    renderModal();

    const providerTab = screen.getByRole('tab', { name: /mangadex/i });
    fireEvent.click(providerTab);

    expect(screen.getByText('api_token')).toBeInTheDocument();

    const tokenInput = screen.getByDisplayValue('secret-token-123');
    expect(tokenInput).toHaveAttribute('type', 'password');

    const toggleSecretBtn = screen.getByRole('button', { name: /show secret/i });
    fireEvent.click(toggleSecretBtn);

    expect(tokenInput).toHaveAttribute('type', 'text');

    const hideSecretBtn = screen.getByRole('button', { name: /hide secret/i });
    fireEvent.click(hideSecretBtn);

    expect(tokenInput).toHaveAttribute('type', 'password');
  });

  it('updates form fields and saves configuration successfully', async () => {
    const onOpenChange = vi.fn();
    renderModal(true, mockPluginWithSettings, onOpenChange);

    const timeoutInput = screen.getByPlaceholderText('30');
    fireEvent.change(timeoutInput, { target: { value: '45' } });

    const saveBtn = screen.getByRole('button', { name: /save settings/i });
    fireEvent.click(saveBtn);

    await waitFor(() => {
      expect(api.updatePluginConfig).toHaveBeenCalledWith('mangadex-plugin', {
        globalConfig: {
          request_timeout: '45',
          enable_debug: 'false',
        },
        providerConfigs: {
          mangadex: {
            api_token: 'secret-token-123',
            image_quality: 'data',
          },
        },
      });
      expect(mockToast).toHaveBeenCalledWith('Saved settings for MangaDex Provider', 'success');
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it('handles save error and opens ErrorDetailsModal', async () => {
    vi.mocked(api.updatePluginConfig).mockRejectedValue(new Error('Validation failed for timeout'));

    renderModal();

    const saveBtn = screen.getByRole('button', { name: /save settings/i });
    fireEvent.click(saveBtn);

    await waitFor(() => {
      expect(mockToast).toHaveBeenCalledWith('Validation failed for timeout', 'error', expect.anything());
    });
  });

  it('calls onOpenChange(false) when Cancel is clicked', () => {
    const onOpenChange = vi.fn();
    renderModal(true, mockPluginWithSettings, onOpenChange);

    const cancelBtn = screen.getByRole('button', { name: 'Cancel' });
    fireEvent.click(cancelBtn);

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
