import { describe, it, expect, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useDetailsManga } from './useDetailsManga';
import * as hooks from '../../../api/hooks';
import * as router from '@tanstack/react-router';
import * as query from '@tanstack/react-query';

vi.mock('@tanstack/react-router', () => ({
  useLocation: vi.fn(),
  useParams: vi.fn(),
}));

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useQuery: vi.fn(),
  };
});

vi.mock('../../../api/hooks', () => ({
  useLibraryManga: vi.fn(),
  useMangaDetails: vi.fn(),
  useProviderMangaDetails: vi.fn(),
  useSources: vi.fn(),
  useChapterList: vi.fn(),
  useProviderChapterList: vi.fn(),
}));

describe('useDetailsManga', () => {
  it('resolves local library manga details and chapters correctly', () => {
    vi.mocked(router.useLocation).mockReturnValue({ pathname: '/manga/m-1' } as any);
    vi.mocked(router.useParams).mockReturnValue({ mangaId: 'm-1' });
    vi.mocked(query.useQuery).mockReturnValue({ data: [] } as any);

    const mockLibraryEntry = {
      id: 'm-1',
      title: 'One Piece',
      contentProviderId: 'mangafox',
      contentRemoteId: 'one-piece',
    };

    vi.mocked(hooks.useLibraryManga).mockReturnValue({ data: [mockLibraryEntry] } as any);
    vi.mocked(hooks.useMangaDetails).mockReturnValue({
      data: mockLibraryEntry,
      isLoading: false,
    } as any);
    vi.mocked(hooks.useProviderMangaDetails).mockReturnValue({ data: undefined } as any);
    vi.mocked(hooks.useSources).mockReturnValue({
      data: [{ id: 'mangafox', name: 'MangaFox', language: 'en' }],
    } as any);
    vi.mocked(hooks.useChapterList).mockReturnValue({
      data: {
        chapters: [
          { id: 'c-1', number: 1, title: 'Chapter 1' },
          { id: 'c-2', number: 2, title: 'Chapter 2' },
        ],
      },
      isLoading: false,
      isError: false,
    } as any);
    vi.mocked(hooks.useProviderChapterList).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    } as any);

    const { result } = renderHook(() => useDetailsManga());

    expect(result.current.isRemoteRoute).toBe(false);
    expect(result.current.isInLibrary).toBe(true);
    expect(result.current.targetMangaId).toBe('m-1');
    expect(result.current.manga?.title).toBe('One Piece');
    expect(result.current.contentProviderName).toBe('MangaFox (EN)');
    expect(result.current.chapters.length).toBe(2);
    expect(result.current.isUnavailable).toBe(false);
  });

  it('resolves remote explore manga correctly when not yet in library', () => {
    vi.mocked(router.useLocation).mockReturnValue({ pathname: '/providers/mangafox/manga/bleach' } as any);
    vi.mocked(router.useParams).mockReturnValue({ providerId: 'mangafox', remoteId: 'bleach' });
    vi.mocked(query.useQuery).mockReturnValue({ data: [] } as any);

    vi.mocked(hooks.useLibraryManga).mockReturnValue({ data: [] } as any);
    vi.mocked(hooks.useMangaDetails).mockReturnValue({ data: undefined, isLoading: false } as any);
    vi.mocked(hooks.useProviderMangaDetails).mockReturnValue({
      data: { id: 'bleach', title: 'Bleach', sourceId: 'mangafox' },
      isLoading: false,
    } as any);
    vi.mocked(hooks.useSources).mockReturnValue({
      data: [{ id: 'mangafox', name: 'MangaFox' }],
    } as any);
    vi.mocked(hooks.useChapterList).mockReturnValue({ data: undefined, isLoading: false, isError: false } as any);
    vi.mocked(hooks.useProviderChapterList).mockReturnValue({
      data: { chapters: [{ id: 'b-1', number: 1, title: 'Bleach 1' }] },
      isLoading: false,
      isError: false,
    } as any);

    const { result } = renderHook(() => useDetailsManga());

    expect(result.current.isRemoteRoute).toBe(true);
    expect(result.current.isInLibrary).toBe(false);
    expect(result.current.manga?.title).toBe('Bleach');
    expect(result.current.chapters.length).toBe(1);
  });

  it('resolves remote explore manga as in-library when libraryMangaId is present in provider details', () => {
    vi.mocked(router.useLocation).mockReturnValue({ pathname: '/providers/mangadex/manga/md-frieren' } as any);
    vi.mocked(router.useParams).mockReturnValue({ providerId: 'mangadex', remoteId: 'md-frieren' });
    vi.mocked(query.useQuery).mockReturnValue({ data: [] } as any);

    vi.mocked(hooks.useLibraryManga).mockReturnValue({ data: [] } as any);
    vi.mocked(hooks.useMangaDetails).mockReturnValue({ data: undefined, isLoading: false } as any);
    vi.mocked(hooks.useProviderMangaDetails).mockReturnValue({
      data: { id: 'md-frieren', title: 'Frieren', sourceId: 'mangadex', libraryMangaId: 'lib-1' },
      isLoading: false,
    } as any);
    vi.mocked(hooks.useSources).mockReturnValue({
      data: [{ id: 'mangadex', name: 'MangaDex' }],
    } as any);
    vi.mocked(hooks.useChapterList).mockReturnValue({
      data: { chapters: [{ id: 'f-1', number: 1, title: 'Frieren 1' }] },
      isLoading: false,
      isError: false,
    } as any);
    vi.mocked(hooks.useProviderChapterList).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    } as any);

    const { result } = renderHook(() => useDetailsManga());

    expect(result.current.isRemoteRoute).toBe(true);
    expect(result.current.isInLibrary).toBe(true);
    expect(result.current.targetMangaId).toBe('lib-1');
  });
});
