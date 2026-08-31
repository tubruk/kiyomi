import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PluginCard } from './PluginCard';
import type { PluginItem } from '../../types/api';

const singleProviderPlugin: PluginItem = {
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
      description: 'Official MangaDex API client',
      capabilities: ['content', 'tracking'],
      defaultRateLimit: {
        requestsPerSecond: 5,
        maxConcurrentRequests: 10,
      },
    },
  ],
};

const multiProviderPlugin: PluginItem = {
  pluginId: 'multi-source-plugin',
  pluginName: 'Multi-Source Provider',
  pluginVersion: '2.0.0',
  sdkVersion: '1.1.0',
  sdkCompatible: true,
  executablePath: '/usr/local/bin/multi-plugin',
  pid: 54321,
  state: 'stopped',
  loadedAt: '2026-08-30T08:00:00Z',
  providers: [
    {
      id: 'source-a',
      name: 'Source Alpha',
      description: 'First source provider',
      capabilities: ['content'],
      defaultRateLimit: {
        requestsPerSecond: 3,
        maxConcurrentRequests: 5,
      },
    },
    {
      id: 'source-b',
      name: 'Source Beta',
      description: 'Second source provider',
      capabilities: ['metadata'],
    },
  ],
};

const errorPlugin: PluginItem = {
  pluginId: 'broken-plugin',
  pluginName: 'Broken Provider',
  pluginVersion: '0.1.0',
  sdkVersion: '0.9.0',
  sdkCompatible: false,
  executablePath: '/usr/local/bin/broken-plugin',
  pid: 0,
  state: 'error',
  loadedAt: '2026-08-30T08:00:00Z',
  providers: [],
};

describe('PluginCard', () => {
  it('renders plugin name, version, ID, and running status badge', () => {
    render(
      <PluginCard
        plugin={singleProviderPlugin}
        onOpenSettings={vi.fn()}
        onOpenLogs={vi.fn()}
      />
    );

    expect(screen.getByText('MangaDex Provider')).toBeInTheDocument();
    expect(screen.getByText('v1.2.0')).toBeInTheDocument();
    expect(screen.getByText('mangadex-plugin')).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
    expect(screen.getByText('SDK v1.0.0')).toBeInTheDocument();
  });

  it('renders single provider details, rate limit, and capability badges', () => {
    render(
      <PluginCard
        plugin={singleProviderPlugin}
        onOpenSettings={vi.fn()}
        onOpenLogs={vi.fn()}
      />
    );

    expect(screen.getByText('Official MangaDex API client')).toBeInTheDocument();
    expect(screen.getByText('5 req/s')).toBeInTheDocument();
    expect(screen.getByText('content')).toBeInTheDocument();
    expect(screen.getByText('tracking')).toBeInTheDocument();
  });

  it('renders multiple providers list with count', () => {
    render(
      <PluginCard
        plugin={multiProviderPlugin}
        onOpenSettings={vi.fn()}
        onOpenLogs={vi.fn()}
      />
    );

    expect(screen.getByText('Providers (2)')).toBeInTheDocument();
    expect(screen.getByText('Source Alpha')).toBeInTheDocument();
    expect(screen.getByText('(source-a)')).toBeInTheDocument();
    expect(screen.getByText('First source provider')).toBeInTheDocument();
    expect(screen.getByText('3 req/s')).toBeInTheDocument();
    expect(screen.getByText('Source Beta')).toBeInTheDocument();
    expect(screen.getByText('(source-b)')).toBeInTheDocument();
    expect(screen.getByText('Second source provider')).toBeInTheDocument();
    expect(screen.getByText('stopped')).toBeInTheDocument();
  });

  it('renders empty providers notice when plugin has no providers', () => {
    render(
      <PluginCard
        plugin={errorPlugin}
        onOpenSettings={vi.fn()}
        onOpenLogs={vi.fn()}
      />
    );

    expect(screen.getByText('No providers registered.')).toBeInTheDocument();
    expect(screen.getByText('error')).toBeInTheDocument();
    expect(screen.getByText('Incompatible SDK')).toBeInTheDocument();
  });

  it('toggles collapsible diagnostic details (PID and executable path)', () => {
    render(
      <PluginCard
        plugin={singleProviderPlugin}
        onOpenSettings={vi.fn()}
        onOpenLogs={vi.fn()}
      />
    );

    expect(screen.queryByText('PID: 12345')).not.toBeInTheDocument();

    const toggleBtn = screen.getByRole('button', { name: /show details/i });
    fireEvent.click(toggleBtn);

    expect(screen.getByText('PID: 12345')).toBeInTheDocument();
    expect(screen.getByText('/usr/local/bin/mangadex-plugin')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /hide details/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /hide details/i }));
    expect(screen.queryByText('PID: 12345')).not.toBeInTheDocument();
  });

  it('triggers onOpenLogs callback when Logs button is clicked', () => {
    const onOpenLogs = vi.fn();
    render(
      <PluginCard
        plugin={singleProviderPlugin}
        onOpenSettings={vi.fn()}
        onOpenLogs={onOpenLogs}
      />
    );

    const logsBtn = screen.getByRole('button', { name: /logs/i });
    fireEvent.click(logsBtn);

    expect(onOpenLogs).toHaveBeenCalledWith(singleProviderPlugin);
  });

  it('triggers onOpenSettings callback when Settings button is clicked', () => {
    const onOpenSettings = vi.fn();
    render(
      <PluginCard
        plugin={singleProviderPlugin}
        onOpenSettings={onOpenSettings}
        onOpenLogs={vi.fn()}
      />
    );

    const settingsBtn = screen.getByRole('button', { name: /settings/i });
    fireEvent.click(settingsBtn);

    expect(onOpenSettings).toHaveBeenCalledWith(singleProviderPlugin);
  });
});
