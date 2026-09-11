import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { AddProviderDialog } from './AddProviderDialog';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { api } from '../api/client';
import { Source } from '../types/api';
import { ToastProvider } from '../context/ToastContext';

const mockSources: Source[] = [
  {
    id: 'mangadex',
    name: 'MangaDex',
    capabilities: ['content', 'metadata'],
  },
];

describe('AddProviderDialog', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.restoreAllMocks();
  });

  const renderDialog = (props: Partial<React.ComponentProps<typeof AddProviderDialog>> = {}) => {
    return render(
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <AddProviderDialog
            mangaId="local-1"
            sources={mockSources}
            mangaTitle="Frieren"
            open={true}
            onOpenChange={() => {}}
            {...props}
          />
        </ToastProvider>
      </QueryClientProvider>
    );
  };

  it('proxies search result cover image and hides image on error', async () => {
    vi.spyOn(api, 'searchManga').mockResolvedValue({
      mangas: [
        {
          id: 'remote-1',
          title: 'Frieren at the Funeral',
          coverUrl: 'https://example.com/cover.jpg',
          url: 'https://mangadex.org/title/remote-1',
          metadata: { title: 'Frieren at the Funeral', aliases: [], description: '', authors: [], artists: [], tags: [], collections: [], publishers: [] },
          user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
          bindings: { providers: [] },
        },
      ],
      hasNext: false,
      page: 1,
    });

    renderDialog();

    await waitFor(() => {
      expect(screen.getByText('Frieren at the Funeral')).toBeInTheDocument();
    }, { timeout: 4000 });

    const img = document.querySelector('img[src*="/api/v1/proxy/image"]') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toContain(encodeURIComponent('https://example.com/cover.jpg'));
    expect(img.src).toContain(encodeURIComponent('https://mangadex.org/title/remote-1'));

    // Trigger onError
    fireEvent.error(img);
    expect(img.style.display).toBe('none');
  });

  it('proxies confirmation step cover image and hides image on error', async () => {
    vi.spyOn(api, 'searchManga').mockResolvedValue({
      mangas: [
        {
          id: 'remote-1',
          title: 'Frieren at the Funeral',
          coverUrl: 'https://example.com/cover.jpg',
          url: 'https://mangadex.org/title/remote-1',
          metadata: { title: 'Frieren at the Funeral', aliases: [], description: '', authors: [], artists: [], tags: [], collections: [], publishers: [] },
          user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
          bindings: { providers: [] },
        },
      ],
      hasNext: false,
      page: 1,
    });

    renderDialog();

    await waitFor(() => {
      expect(screen.getByText('Frieren at the Funeral')).toBeInTheDocument();
    }, { timeout: 4000 });

    fireEvent.click(screen.getByText('Frieren at the Funeral'));

    await waitFor(() => {
      expect(screen.getByText('Confirm Add Provider')).toBeInTheDocument();
    });

    const img = document.querySelector('img[src*="/api/v1/proxy/image"]') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toContain(encodeURIComponent('https://example.com/cover.jpg'));

    // Trigger onError
    fireEvent.error(img);
    expect(img.style.display).toBe('none');
  });
});
