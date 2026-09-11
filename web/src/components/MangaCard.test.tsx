import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/react';
import { MangaCard } from './MangaCard';
import { LibraryMangaCard } from './LibraryMangaCard';
import { Manga } from '../types/api';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Mock Link from @tanstack/react-router
vi.mock('@tanstack/react-router', () => ({
  Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
}));

describe('MangaCard and LibraryMangaCard cover image fallbacks', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.restoreAllMocks();
  });

  it('MangaCard uses proxied cover url and hides the img (revealing the icon placeholder) on error', () => {
    const manga: Manga = {
      id: 'local-1',
      title: 'Manga Test',
      coverUrl: 'https://example.com/cover.jpg',
      url: 'https://mangadex.org/title/manga-test',
      metadata: { title: 'Manga Test', aliases: [], description: '', authors: [], artists: [], tags: [], collections: [], publishers: [] },
      user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
      bindings: { providers: [] },
    };

    const { container } = render(<MangaCard manga={manga} />);
    const img = container.querySelector('img') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toContain('/api/v1/proxy/image');
    expect(img.src).toContain(encodeURIComponent('https://example.com/cover.jpg'));

    // Trigger error on proxied image — img gets hidden, icon placeholder remains.
    fireEvent.error(img);
    expect(img.style.display).toBe('none');
    // Ensure it NEVER points to raw remote URL
    expect(img.src).not.toBe('https://example.com/cover.jpg');
  });

  it('MangaCard falls back to proxied url if coverAssetUrl fails, then hides the img on second failure', () => {
    const manga: Manga = {
      id: 'local-1',
      title: 'Manga Test',
      coverAssetUrl: '/api/v1/library/manga/local-1/cover',
      coverUrl: 'https://example.com/cover.jpg',
      url: 'https://mangadex.org/title/manga-test',
      metadata: { title: 'Manga Test', aliases: [], description: '', authors: [], artists: [], tags: [], collections: [], publishers: [] },
      user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
      bindings: { providers: [] },
    };

    const { container } = render(<MangaCard manga={manga} />);
    const initialImg = container.querySelector('img') as HTMLImageElement;
    expect(initialImg.src).toContain('/api/v1/library/manga/local-1/cover');

    // First error on coverAssetUrl -> CoverImage swaps src to proxied URL and remounts the img.
    fireEvent.error(initialImg);
    const fallbackImg = container.querySelector('img') as HTMLImageElement;
    expect(fallbackImg.src).toContain('/api/v1/proxy/image');
    expect(fallbackImg.src).not.toBe('https://example.com/cover.jpg');

    // Second error on proxied URL -> img hidden, icon placeholder remains.
    fireEvent.error(fallbackImg);
    expect(fallbackImg.style.display).toBe('none');
    expect(fallbackImg.src).not.toBe('https://example.com/cover.jpg');
  });

  it('LibraryMangaCard uses proxied cover url and hides the img on error', () => {
    const manga: Manga = {
      id: 'local-1',
      title: 'Manga Test',
      coverUrl: 'https://example.com/cover.jpg',
      url: 'https://mangadex.org/title/manga-test',
      metadata: { title: 'Manga Test', aliases: [], description: '', authors: [], artists: [], tags: [], collections: [], publishers: [] },
      user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
      bindings: { providers: [] },
    };

    const { container } = render(
      <QueryClientProvider client={queryClient}>
        <LibraryMangaCard manga={manga} />
      </QueryClientProvider>
    );
    const img = container.querySelector('img') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toContain('/api/v1/proxy/image');

    // Trigger error on proxied image
    fireEvent.error(img);
    expect(img.style.display).toBe('none');
    expect(img.src).not.toBe('https://example.com/cover.jpg');
  });
});
