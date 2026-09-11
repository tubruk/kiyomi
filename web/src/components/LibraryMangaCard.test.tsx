import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LibraryMangaCard } from './LibraryMangaCard';
import { Manga, ChapterListResponse } from '../types/api';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { api } from '../api/client';

// Mock Link from @tanstack/react-router
vi.mock('@tanstack/react-router', () => ({
  Link: ({ children, to, params, ...props }: any) => (
    <a href={to ? to.replace('$mangaId', params?.mangaId || '') : '#'} {...props}>
      {children}
    </a>
  ),
}));

const mockManga: Manga = {
  id: 'manga-1',
  title: 'Frieren at the Funeral',
  coverAssetUrl: '/api/v1/library/manga/manga-1/cover',
  coverUrl: 'https://example.com/cover.jpg',
  url: 'https://mangadex.org/title/manga-1',
  authors: ['Kanehito Yamada', 'Tsukasa Abe'],
  sourceId: 'mangadex',
  userStatus: 'reading',
  userRating: 9,
  userFavorite: true,
  shelves: ['Top Favorites', 'Reading Later'],
  metadata: {
    title: 'Frieren at the Funeral',
    aliases: [],
    description: '',
    authors: ['Kanehito Yamada', 'Tsukasa Abe'],
    artists: [],
    tags: [],
    collections: [],
    publishers: [],
  },
  user_state: { status: 'reading', rating: 9, favorite: true, notes: '' },
  bindings: { providers: [] },
};

describe('LibraryMangaCard', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.restoreAllMocks();
  });

  const renderCard = (props: Partial<React.ComponentProps<typeof LibraryMangaCard>> = {}) => {
    const defaultProps: React.ComponentProps<typeof LibraryMangaCard> = {
      manga: mockManga,
      ...props,
    };

    return render(
      <QueryClientProvider client={queryClient}>
        <LibraryMangaCard {...defaultProps} />
      </QueryClientProvider>
    );
  };

  it('renders manga title, authors, user status, user rating, and shelves badge', () => {
    renderCard();

    // Title
    expect(screen.getByText('Frieren at the Funeral')).toBeInTheDocument();

    // Authors joined display
    expect(screen.getByText('Kanehito Yamada, Tsukasa Abe')).toBeInTheDocument();

    // User status
    expect(screen.getByText('reading')).toBeInTheDocument();

    // Rating
    expect(screen.getByText('9')).toBeInTheDocument();

    // Shelf badge (first shelf)
    expect(screen.getByText('Top Favorites')).toBeInTheDocument();

    // Favorite heart icon
    expect(screen.getByTitle('Favorite')).toBeInTheDocument();
  });

  it('renders provider badge when providerId is present', () => {
    renderCard({
      manga: {
        ...mockManga,
        sourceId: 'mangadex',
      },
    });

    expect(screen.getByText('mangadex')).toBeInTheDocument();
  });

  it('does not render provider badge when providerId is absent', () => {
    const mangaWithoutProvider: Manga = {
      ...mockManga,
      sourceId: undefined,
      contentProviderId: undefined,
      meta: undefined,
    };

    renderCard({ manga: mangaWithoutProvider });

    expect(screen.queryByText('mangadex')).not.toBeInTheDocument();
  });

  it('renders unread count badge when chapters exist with unread items', async () => {
    const mockChaptersResponse: ChapterListResponse = {
      chapters: [
        { id: 'ch-1', name: 'Ch 1', number: 1, is_read: true } as any,
        { id: 'ch-2', name: 'Ch 2', number: 2, is_read: false } as any,
        { id: 'ch-3', name: 'Ch 3', number: 3, meta: { is_read: false } } as any,
      ],
    };

    vi.spyOn(api, 'getMangaChapters').mockResolvedValue(mockChaptersResponse);

    renderCard();

    // Expect unread count to be 2
    const unreadBadge = await screen.findByText('2 unread');
    expect(unreadBadge).toBeInTheDocument();
  });

  it('renders "Completed" badge when userStatus is completed even if unread chapters exist', async () => {
    const mockChaptersResponse: ChapterListResponse = {
      chapters: [
        { id: 'ch-1', name: 'Ch 1', number: 1, is_read: false } as any,
      ],
    };

    vi.spyOn(api, 'getMangaChapters').mockResolvedValue(mockChaptersResponse);

    renderCard({
      manga: {
        ...mockManga,
        userStatus: 'completed',
        user_state: { status: 'completed', rating: 9, favorite: true, notes: '' },
      },
    });

    const completedBadge = await screen.findByText('Completed');
    expect(completedBadge).toBeInTheDocument();
    expect(screen.queryByText('1 unread')).not.toBeInTheDocument();
  });

  it('renders unread count badge directly when unreadCount prop is passed without fetching chapters', () => {
    const getChaptersSpy = vi.spyOn(api, 'getMangaChapters');

    renderCard({
      unreadCount: 5,
    });

    expect(screen.getByText('5 unread')).toBeInTheDocument();
    expect(getChaptersSpy).not.toHaveBeenCalled();
  });

  it('image src uses coverAssetUrl when provided', () => {
    const { container } = renderCard({
      manga: {
        ...mockManga,
        coverAssetUrl: '/api/v1/library/manga/manga-1/cover',
      },
    });

    const img = container.querySelector('img') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toContain('/api/v1/library/manga/manga-1/cover');
  });

  it('image src falls back to proxied cover URL when coverAssetUrl is absent', () => {
    const { container } = renderCard({
      manga: {
        ...mockManga,
        coverAssetUrl: undefined,
        coverUrl: 'https://example.com/cover.jpg',
        url: 'https://mangadex.org/title/manga-1',
      },
    });

    const img = container.querySelector('img') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toContain('/api/v1/proxy/image');
    expect(img.src).toContain(encodeURIComponent('https://example.com/cover.jpg'));
    expect(img.src).toContain(encodeURIComponent('https://mangadex.org/title/manga-1'));
  });

  it('image onError falls back safely from coverAssetUrl to proxied URL, then hides the img without leaking raw upstream URL', () => {
    const { container } = renderCard({
      manga: {
        ...mockManga,
        coverAssetUrl: '/api/v1/library/manga/manga-1/cover',
        coverUrl: 'https://example.com/cover.jpg',
        url: 'https://mangadex.org/title/manga-1',
      },
    });

    const initialImg = container.querySelector('img') as HTMLImageElement;
    expect(initialImg.src).toContain('/api/v1/library/manga/manga-1/cover');

    // 1st error on coverAssetUrl -> CoverImage swaps src to proxied URL and remounts the img.
    fireEvent.error(initialImg);
    const fallbackImg = container.querySelector('img') as HTMLImageElement;
    expect(fallbackImg.src).toContain('/api/v1/proxy/image');
    expect(fallbackImg.src).not.toBe('https://example.com/cover.jpg');

    // 2nd error on proxied URL -> img hidden, icon placeholder remains
    fireEvent.error(fallbackImg);
    expect(fallbackImg.style.display).toBe('none');
    expect(fallbackImg.src).not.toBe('https://example.com/cover.jpg');
  });

  it('image onError hides the img directly when no coverAssetUrl and proxied URL fails', () => {
    const { container } = renderCard({
      manga: {
        ...mockManga,
        coverAssetUrl: undefined,
        coverUrl: 'https://example.com/cover.jpg',
        url: 'https://mangadex.org/title/manga-1',
      },
    });

    const img = container.querySelector('img') as HTMLImageElement;
    expect(img.src).toContain('/api/v1/proxy/image');

    // Error on proxied URL -> img hidden, icon placeholder remains
    fireEvent.error(img);
    expect(img.style.display).toBe('none');
    expect(img.src).not.toBe('https://example.com/cover.jpg');
  });

  it('calls onDelete when delete button is clicked and user confirms', async () => {
    const user = userEvent.setup();
    const handleDelete = vi.fn();
    vi.spyOn(window, 'confirm').mockReturnValue(true);

    renderCard({ onDelete: handleDelete });

    const deleteBtn = screen.getByRole('button', { name: /remove from library/i });
    await user.click(deleteBtn);

    expect(window.confirm).toHaveBeenCalledWith('Remove "Frieren at the Funeral" from library?');
    expect(handleDelete).toHaveBeenCalledTimes(1);
    expect(handleDelete).toHaveBeenCalledWith('manga-1');
  });

  it('does not call onDelete when user cancels the confirm dialog', async () => {
    const user = userEvent.setup();
    const handleDelete = vi.fn();
    vi.spyOn(window, 'confirm').mockReturnValue(false);

    renderCard({ onDelete: handleDelete });

    const deleteBtn = screen.getByRole('button', { name: /remove from library/i });
    await user.click(deleteBtn);

    expect(window.confirm).toHaveBeenCalled();
    expect(handleDelete).not.toHaveBeenCalled();
  });

  it('disables delete button when isDeleting is true', () => {
    renderCard({ onDelete: vi.fn(), isDeleting: true });

    const deleteBtn = screen.getByRole('button', { name: /remove from library/i });
    expect(deleteBtn).toBeDisabled();
  });
});
