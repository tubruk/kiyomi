import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CollisionResolutionAlert } from './CollisionResolutionAlert';
import { ToastProvider } from '../../context/ToastContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ProviderCollision } from '../../types/api';

const queryClient = new QueryClient();

const renderComponent = (collisions: ProviderCollision[]) =>
  render(
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <CollisionResolutionAlert collisions={collisions} />
      </ToastProvider>
    </QueryClientProvider>
  );

describe('CollisionResolutionAlert', () => {
  it('returns null when collisions is empty', () => {
    const { container } = renderComponent([]);
    expect(container.firstChild).toBeNull();
  });

  it('renders collision alert with count and provider information', () => {
    const mockCollisions: ProviderCollision[] = [
      {
        providerId: 'mangadex',
        selected: 'builtin',
        candidates: [
          {
            pluginId: 'builtin',
            version: '1.0.0',
            isBuiltIn: true,
            selected: true,
          },
          {
            pluginId: 'mangadex-ext',
            version: '2.0.0',
            isBuiltIn: false,
            selected: false,
          },
        ],
      },
    ];

    renderComponent(mockCollisions);

    expect(screen.getByText('Provider ID Collision Detected')).toBeInTheDocument();
    expect(screen.getByText('1 conflict')).toBeInTheDocument();
    expect(screen.getByText('mangadex')).toBeInTheDocument();
    expect(screen.getByText('Built-in (v1.0.0)')).toBeInTheDocument();
  });
});
