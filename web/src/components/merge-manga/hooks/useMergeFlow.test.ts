import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useMergeFlow } from './useMergeFlow';
import { Manga } from '../../../types/api';

const keepManga: Manga = {
  id: 'm-keep',
  title: 'Frieren at the Funeral',
  coverUrl: 'https://example.com/keep-cover.jpg',
  content: {
    provider_id: 'mangadex',
    provider_manga_id: 'keep-1',
  },
  metadata: {
    title: 'Frieren at the Funeral',
    aliases: ['Frieren'],
    description: 'Keep description',
    authors: [],
    artists: [],
    tags: [],
    collections: [],
    publishers: [],
    release_year: 2020,
    cover_url: 'https://example.com/keep-cover.jpg',
  },
  user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
  bindings: {
    content: {
      provider_id: 'mangadex',
      provider_manga_id: 'keep-1',
    },
    providers: [
      { provider_id: 'mangadex', provider_manga_id: 'keep-1' },
    ],
  },
};

const sourceManga1: Manga = {
  id: 'm-src-1',
  title: 'Frieren (MAL)',
  coverUrl: 'https://example.com/src1-cover.jpg',
  content: {
    provider_id: 'mal',
    provider_manga_id: 'src-1',
  },
  metadata: {
    title: 'Frieren',
    aliases: ['Frieren Beyond'],
    description: 'Source 1 description',
    authors: [],
    artists: [],
    tags: [],
    collections: [],
    publishers: [],
    release_year: 2021,
    cover_url: 'https://example.com/src1-cover.jpg',
  },
  user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
  bindings: {
    content: {
      provider_id: 'mal',
      provider_manga_id: 'src-1',
    },
    providers: [
      { provider_id: 'mal', provider_manga_id: 'src-1' },
    ],
  },
};

const sourceManga2: Manga = {
  id: 'm-src-2',
  title: 'Frieren (MangaFox)',
  coverUrl: 'https://example.com/src2-cover.jpg',
  content: {
    provider_id: 'mangafox',
    provider_manga_id: 'src-2',
  },
  metadata: {
    title: 'Frieren',
    aliases: [],
    description: 'Source 2 description',
    authors: [],
    artists: [],
    tags: [],
    collections: [],
    publishers: [],
    release_year: 2022,
    cover_url: 'https://example.com/src2-cover.jpg',
  },
  user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
  bindings: {
    content: {
      provider_id: 'mangafox',
      provider_manga_id: 'src-2',
    },
    providers: [
      { provider_id: 'mangafox', provider_manga_id: 'src-2' },
    ],
  },
};

const libraryMangas: Manga[] = [keepManga, sourceManga1, sourceManga2];

describe('useMergeFlow', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    // Pre-seed the library manga query cache so the hook can find keep + sources.
    queryClient.setQueryData(['library', 'manga'], libraryMangas);
  });

  const createWrapper = () => {
    return ({ children }: { children: React.ReactNode }) =>
      React.createElement(QueryClientProvider, { client: queryClient }, children);
  };

  it('initializes with no source selected', () => {
    const { result } = renderHook(
      () => useMergeFlow({ keepMangaId: 'm-keep' }),
      { wrapper: createWrapper() }
    );

    expect(result.current.sourceMangaId).toBeNull();
    expect(result.current.sourceManga).toBeUndefined();
    expect(result.current.availableSources.map((m) => m.id)).toEqual(['m-src-1', 'm-src-2']);
  });

  it('availableSources excludes the keep manga', () => {
    const { result } = renderHook(
      () => useMergeFlow({ keepMangaId: 'm-keep' }),
      { wrapper: createWrapper() }
    );

    const ids = result.current.availableSources.map((m) => m.id);
    expect(ids).not.toContain('m-keep');
    expect(ids).toEqual(['m-src-1', 'm-src-2']);
  });

  it('setSourceMangaId updates sourceMangaId and sourceManga', () => {
    const { result } = renderHook(
      () => useMergeFlow({ keepMangaId: 'm-keep' }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setSourceMangaId('m-src-1');
    });

    expect(result.current.sourceMangaId).toBe('m-src-1');
    expect(result.current.sourceManga?.id).toBe('m-src-1');
    expect(result.current.sourceManga?.title).toBe('Frieren (MAL)');
  });

  it('setSourceMangaId(null) clears the selection', () => {
    const { result } = renderHook(
      () => useMergeFlow({ keepMangaId: 'm-keep' }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setSourceMangaId('m-src-2');
    });
    expect(result.current.sourceMangaId).toBe('m-src-2');

    act(() => {
      result.current.setSourceMangaId(null);
    });
    expect(result.current.sourceMangaId).toBeNull();
    expect(result.current.sourceManga).toBeUndefined();
  });

  it('sourceManga is undefined when an unknown id is set', () => {
    const { result } = renderHook(
      () => useMergeFlow({ keepMangaId: 'm-keep' }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setSourceMangaId('does-not-exist');
    });

    expect(result.current.sourceMangaId).toBe('does-not-exist');
    expect(result.current.sourceManga).toBeUndefined();
  });
});