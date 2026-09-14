import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useEditMetadataForm } from './useEditMetadataForm';
import * as hooks from '../../api/hooks';
import { Manga } from '../../types/api';

vi.mock('../../api/hooks', () => ({
  useUpdateLibraryMangaMutation: vi.fn(),
}));

const mockManga: Manga = {
  id: 'm-1',
  title: 'Bleach',
  aliases: ['Kurosaki'],
  authors: ['Tite Kubo'],
  artists: ['Tite Kubo'],
  description: 'Soul reaper story',
  readingMode: 'rtl',
  contentRating: 'safe',
  publisher: 'Shueisha',
  publishers: ['Shueisha', 'VIZ Media'],
  releaseYear: 2001,
  startDate: '2001-08-07',
  endDate: '2016-08-22',
  country: 'JP',
  tags: ['Action', 'Supernatural'],
  shelves: ['Shonen'],
  externalLinks: [{ provider: 'anilist', label: 'AniList', url: 'https://anilist.co' }],
  metadata: {
    title: 'Bleach',
    aliases: ['Kurosaki'],
    description: 'Soul reaper story',
    authors: ['Tite Kubo'],
    artists: ['Tite Kubo'],
    tags: ['Action', 'Supernatural'],
    collections: ['Shonen'],
    publishers: ['Shueisha', 'VIZ Media'],
  },
  user_state: {
    status: 'reading',
    rating: 0,
    favorite: false,
    notes: '',
  },
  bindings: {
    providers: [],
  },
};

describe('useEditMetadataForm', () => {
  const mutate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(hooks.useUpdateLibraryMangaMutation).mockReturnValue({
      mutate,
      isPending: false,
    } as any);
  });

  it('initializes form state from manga props when opened', () => {
    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: mockManga,
        open: true,
        onOpenChange: vi.fn(),
      })
    );

    expect(result.current.title).toBe('Bleach');
    expect(result.current.aliases).toEqual(['Kurosaki']);
    expect(result.current.authors).toEqual(['Tite Kubo']);
    expect(result.current.artists).toEqual(['Tite Kubo']);
    expect(result.current.publishers).toEqual(['Shueisha', 'VIZ Media']);
    expect(result.current.description).toBe('Soul reaper story');
    expect(result.current.readingMode).toBe('rtl');
    expect(result.current.contentRating).toBe('safe');
    expect(result.current.releaseYear).toBe(2001);
    expect(result.current.startDate).toBe('2001-08-07');
    expect(result.current.endDate).toBe('2016-08-22');
    expect(result.current.country).toBe('JP');
    expect(result.current.tags).toEqual(['Action', 'Supernatural']);
    expect(result.current.shelves).toEqual(['Shonen']);
    expect(result.current.externalLinks.length).toBe(1);
    expect(result.current.isUpdating).toBe(false);
  });

  it('falls back to legacy single string fields and aliases when arrays are missing', () => {
    const legacyManga: Manga = {
      id: 'm-legacy',
      title: 'One Piece',
      author: 'Eiichiro Oda',
      artist: 'Eiichiro Oda',
      publisher: 'Shueisha',
      genres: ['Adventure', 'Pirates'],
      collections: ['Favorites', 'Ongoing'],
      start_date: '1997-07-22',
      end_date: '2024-01-01',
      release_year: 1997,
      country: 'JP',
      metadata: {
        title: 'One Piece',
        aliases: [],
        description: '',
        authors: [],
        artists: [],
        tags: [],
        collections: [],
        publishers: [],
      },
      user_state: {
        status: 'reading',
        rating: 0,
        favorite: false,
        notes: '',
      },
      bindings: {
        providers: [],
      },
    };

    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: legacyManga,
        open: true,
        onOpenChange: vi.fn(),
      })
    );

    expect(result.current.authors).toEqual(['Eiichiro Oda']);
    expect(result.current.artists).toEqual(['Eiichiro Oda']);
    expect(result.current.publishers).toEqual(['Shueisha']);
    expect(result.current.tags).toEqual(['Adventure', 'Pirates']);
    expect(result.current.shelves).toEqual(['Favorites', 'Ongoing']);
    expect(result.current.startDate).toBe('1997-07-22');
    expect(result.current.endDate).toBe('2024-01-01');
    expect(result.current.releaseYear).toBe(1997);
  });

  it('falls back to meta object properties when top-level fields are absent', () => {
    const metaManga: Manga = {
      id: 'm-meta',
      title: 'Meta Manga',
      readingDirection: 'vertical',
      metadata: {
        title: 'Meta Manga',
        aliases: ['Meta Title Alternate'],
        description: '',
        authors: ['Meta Author'],
        artists: ['Meta Artist'],
        publishers: ['Meta Publisher 1', 'Meta Publisher 2'],
        tags: ['Cyberpunk', 'Sci-Fi'],
        collections: ['Reading List'],
        start_date: '2022-03-01',
        startDate: '2022-03-01',
        end_date: '2023-04-01',
        endDate: '2023-04-01',
        release_year: 2022,
        releaseYear: 2022,
        content_rating: 'mature',
        contentRating: 'mature',
        country: 'KR',
      },
      user_state: {
        status: 'reading',
        rating: 0,
        favorite: false,
        notes: '',
      },
      bindings: {
        providers: [],
      },
    };

    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: metaManga,
        open: true,
        onOpenChange: vi.fn(),
      })
    );

    expect(result.current.aliases).toEqual(['Meta Title Alternate']);
    expect(result.current.authors).toEqual(['Meta Author']);
    expect(result.current.artists).toEqual(['Meta Artist']);
    expect(result.current.publishers).toEqual(['Meta Publisher 1', 'Meta Publisher 2']);
    expect(result.current.tags).toEqual(['Cyberpunk', 'Sci-Fi']);
    expect(result.current.shelves).toEqual(['Reading List']);
    expect(result.current.startDate).toBe('2022-03-01');
    expect(result.current.endDate).toBe('2023-04-01');
    expect(result.current.releaseYear).toBe(2022);
    expect(result.current.contentRating).toBe('mature');
    expect(result.current.country).toBe('KR');
    expect(result.current.readingMode).toBe('vertical');
  });

  it('falls back to meta.publisher single string when meta.publishers array is not defined', () => {
    const metaSinglePublisherManga: Manga = {
      id: 'm-single-pub',
      title: 'Solo Manga',
      publisher: 'Square Enix',
      metadata: {
        title: 'Solo Manga',
        aliases: [],
        description: '',
        authors: [],
        artists: [],
        tags: [],
        collections: [],
        publishers: [],
      },
      meta: {
        publisher: 'Square Enix',
      },
      user_state: {
        status: 'reading',
        rating: 0,
        favorite: false,
        notes: '',
      },
      bindings: {
        providers: [],
      },
    };

    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: metaSinglePublisherManga,
        open: true,
        onOpenChange: vi.fn(),
      })
    );

    expect(result.current.publishers).toEqual(['Square Enix']);
  });

  it('handles state updates for all array-based fields, start/end dates, and scalars', () => {
    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: mockManga,
        open: true,
        onOpenChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setPublishers(['Kodansha', 'Vertical Comics']);
      result.current.setAuthors(['Author A', 'Author B']);
      result.current.setArtists(['Artist A']);
      result.current.setAliases(['Alias 1', 'Alias 2']);
      result.current.setTags(['Action', 'Mecha']);
      result.current.setShelves(['To Read', 'Favorites']);
      result.current.setStartDate('2020-01-15');
      result.current.setEndDate('2023-12-31');
      result.current.setReleaseYear(2020);
      result.current.setCountry('KR');
      result.current.setContentRating('mature');
      result.current.setReadingMode('longstrip');
    });

    expect(result.current.publishers).toEqual(['Kodansha', 'Vertical Comics']);
    expect(result.current.authors).toEqual(['Author A', 'Author B']);
    expect(result.current.artists).toEqual(['Artist A']);
    expect(result.current.aliases).toEqual(['Alias 1', 'Alias 2']);
    expect(result.current.tags).toEqual(['Action', 'Mecha']);
    expect(result.current.shelves).toEqual(['To Read', 'Favorites']);
    expect(result.current.startDate).toBe('2020-01-15');
    expect(result.current.endDate).toBe('2023-12-31');
    expect(result.current.releaseYear).toBe(2020);
    expect(result.current.country).toBe('KR');
    expect(result.current.contentRating).toBe('mature');
    expect(result.current.readingMode).toBe('longstrip');
  });

  it('handles adding, updating, and removing external links', () => {
    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: mockManga,
        open: true,
        onOpenChange: vi.fn(),
      })
    );

    act(() => {
      result.current.handleAddLink();
    });

    expect(result.current.externalLinks.length).toBe(2);

    act(() => {
      result.current.handleLinkChange(1, 'url', 'https://myanimelist.net');
    });

    expect(result.current.externalLinks[1].url).toBe('https://myanimelist.net');

    act(() => {
      result.current.handleRemoveLink(0);
    });

    expect(result.current.externalLinks.length).toBe(1);
    expect(result.current.externalLinks[0].url).toBe('https://myanimelist.net');
  });

  it('submits updated array fields, dates, and scalar metadata, closing dialog on success', () => {
    const onOpenChange = vi.fn();
    const onSaved = vi.fn();

    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: mockManga,
        open: true,
        onOpenChange,
        onSaved,
      })
    );

    act(() => {
      result.current.setTitle('Bleach Thousand-Year Blood War');
      result.current.setAliases(['TYBW', 'Bleach Final']);
      result.current.setAuthors(['Tite Kubo']);
      result.current.setArtists(['Tite Kubo', 'Assistant']);
      result.current.setPublishers(['Shueisha', 'VIZ Media', 'Glénat']);
      result.current.setTags(['Action', 'Supernatural', 'Shonen']);
      result.current.setShelves(['Completed', 'Masterpieces']);
      result.current.setStartDate('2001-08-07');
      result.current.setEndDate('2016-08-22');
      result.current.setReleaseYear(2001);
      result.current.setCountry('JP');
      result.current.setContentRating('suggestive');
      result.current.setReadingMode('rtl');
    });

    act(() => {
      result.current.handleSubmit({ preventDefault: vi.fn() } as any);
    });

    expect(mutate).toHaveBeenCalledWith(
      expect.objectContaining({
        mangaId: 'm-1',
        fields: expect.objectContaining({
          metadata: expect.objectContaining({
            title: 'Bleach Thousand-Year Blood War',
            aliases: ['TYBW', 'Bleach Final'],
            authors: ['Tite Kubo'],
            artists: ['Tite Kubo', 'Assistant'],
            publishers: ['Shueisha', 'VIZ Media', 'Glénat'],
            publisher: 'Shueisha',
            tags: ['Action', 'Supernatural', 'Shonen'],
            shelves: ['Completed', 'Masterpieces'],
            collections: ['Completed', 'Masterpieces'],
            startDate: '2001-08-07',
            start_date: '2001-08-07',
            endDate: '2016-08-22',
            end_date: '2016-08-22',
            releaseYear: 2001,
            release_year: 2001,
            country: 'JP',
            contentRating: 'suggestive',
            content_rating: 'suggestive',
          }),
          bindings: expect.objectContaining({
            content: expect.objectContaining({
              reading_mode: 'rtl',
            }),
          }),
        }),
      }),
      expect.objectContaining({
        onSuccess: expect.any(Function),
      })
    );

    // Simulate mutation onSuccess trigger
    const mutationConfig = mutate.mock.calls[0][1];
    act(() => {
      mutationConfig.onSuccess();
    });

    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(onSaved).toHaveBeenCalledTimes(1);
  });

  it('sets publisher fallback to empty string when publishers array is empty on submit', () => {
    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: mockManga,
        open: true,
        onOpenChange: vi.fn(),
      })
    );

    act(() => {
      result.current.setPublishers([]);
    });

    act(() => {
      result.current.handleSubmit({ preventDefault: vi.fn() } as any);
    });

    expect(mutate).toHaveBeenCalledWith(
      expect.objectContaining({
        fields: expect.objectContaining({
          metadata: expect.objectContaining({
            publishers: [],
            publisher: '',
          }),
        }),
      }),
      expect.any(Object)
    );
  });
});
