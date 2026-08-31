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

describe('MangaCard and LibraryMangaCard image fallbacks', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.restoreAllMocks();
  });

  it('MangaCard uses proxied cover url and falls back safely to placeholder on error', () => {
    const manga: Manga = {
      id: 'local-1',
      title: 'Manga Test',
      coverUrl: 'https://example.com/cover.jpg',
      url: 'https://mangadex.org/title/manga-test',
    };

    const { container } = render(<MangaCard manga={manga} />);
    const img = container.querySelector('img') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toContain('/api/v1/proxy/image');
    expect(img.src).toContain(encodeURIComponent('https://example.com/cover.jpg'));

    // Trigger error on proxied image
    fireEvent.error(img);
    expect(img.src).toContain('/placeholder.jpg');
    // Ensure it NEVER points to raw remote URL
    expect(img.src).not.toBe('https://example.com/cover.jpg');
  });

  it('MangaCard falls back to proxied url if coverAssetUrl fails, then placeholder', () => {
    const manga: Manga = {
      id: 'local-1',
      title: 'Manga Test',
      coverAssetUrl: '/api/v1/library/manga/local-1/cover',
      coverUrl: 'https://example.com/cover.jpg',
      url: 'https://mangadex.org/title/manga-test',
    };

    const { container } = render(<MangaCard manga={manga} />);
    const img = container.querySelector('img') as HTMLImageElement;
    expect(img.src).toContain('/api/v1/library/manga/local-1/cover');

    // First error on coverAssetUrl -> fallback to proxied URL
    fireEvent.error(img);
    expect(img.src).toContain('/api/v1/proxy/image');

    // Second error on proxied URL -> fallback to placeholder
    fireEvent.error(img);
    expect(img.src).toContain('/placeholder.jpg');
    expect(img.src).not.toBe('https://example.com/cover.jpg');
  });

  it('LibraryMangaCard uses proxied cover url and falls back safely to placeholder on error', () => {
    const manga: Manga = {
      id: 'local-1',
      title: 'Manga Test',
      coverUrl: 'https://example.com/cover.jpg',
      url: 'https://mangadex.org/title/manga-test',
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
    expect(img.src).toContain('/placeholder.jpg');
    expect(img.src).not.toBe('https://example.com/cover.jpg');
  });
});
