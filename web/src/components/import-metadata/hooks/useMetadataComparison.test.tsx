import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useMetadataComparison } from './useMetadataComparison';
import { api } from '../../../api/client';
import { Manga, Source, ProviderRef } from '../../../types/api';
import * as toastContext from '../../../context/ToastContext';

vi.mock('../../../context/ToastContext', () => ({
  useToast: vi.fn(),
}));

const mockSources: Source[] = [
  { id: 'mangadex', name: 'MangaDex' },
  { id: 'anilist', name: 'AniList' },
];

const baseKeepManga: Manga = {
  id: 'm-keep',
  title: 'Keep Title',
  authors: ['Author A'],
  tags: ['Fantasy'],
  metadata: {
    title: 'Keep Title',
    aliases: [],
    description: '',
    authors: ['Author A'],
    artists: [],
    tags: ['Fantasy'],
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
    providers: [
      {
        provider_id: 'mangadex',
        provider_manga_id: 'md-1',
        manga_title: 'Keep Title',
      } as ProviderRef,
    ],
  },
};

const baseIncomingManga: Manga = {
  id: 'm-source',
  title: 'Incoming Title',
  authors: ['Author B'],
  tags: ['Adventure'],
  metadata: {
    title: 'Incoming Title',
    aliases: [],
    description: '',
    authors: ['Author B'],
    artists: [],
    tags: ['Adventure'],
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
    providers: [
      {
        provider_id: 'anilist',
        provider_manga_id: 'al-1',
        manga_title: 'Incoming Title',
      } as ProviderRef,
    ],
  },
};

const renderUseMetadataComparison = (overrides: Partial<Parameters<typeof useMetadataComparison>[0]> = {}) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  return renderHook(
    () =>
      useMetadataComparison({
        manga: baseKeepManga,
        sources: mockSources,
        selectedProviderId: 'mangadex',
        onClose: vi.fn(),
        ...overrides,
      }),
    { wrapper }
  );
};

describe('useMetadataComparison - providers field', () => {
  const showToast = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(toastContext.useToast).mockReturnValue({ showToast } as any);
  });

  it('exposes providers from manga.bindings.providers in currentValues (import mode)', async () => {
    const { result } = renderUseMetadataComparison();

    await waitFor(() => {
      expect(result.current.currentValues.providers).toEqual([
        expect.objectContaining({ provider_id: 'mangadex', provider_manga_id: 'md-1' }),
      ]);
    });
  });

  it('exposes providers from incomingManga.bindings.providers in incomingValues when mode="merge"', async () => {
    const { result } = renderUseMetadataComparison({
      mode: 'merge',
      incomingManga: baseIncomingManga,
      sourceMangaIds: [baseIncomingManga.id],
    });

    await waitFor(() => {
      expect(result.current.incomingValues.providers).toEqual([
        expect.objectContaining({ provider_id: 'anilist', provider_manga_id: 'al-1' }),
      ]);
    });
  });

  it('hides providers in incomingValues when mode="import"', async () => {
    const { result } = renderUseMetadataComparison();

    // No remote has been chosen yet — incomingValues.providers should be empty.
    expect(result.current.incomingValues.providers).toEqual([]);
  });

  it('defaults selectedProvidersMode to "merged" in merge mode', async () => {
    const { result } = renderUseMetadataComparison({
      mode: 'merge',
      incomingManga: baseIncomingManga,
      sourceMangaIds: [baseIncomingManga.id],
    });

    expect(result.current.selectedProvidersMode).toBe('merged');
  });
});

describe('useMetadataComparison - merge mutation', () => {
  const showToast = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(toastContext.useToast).mockReturnValue({ showToast } as any);
    vi.spyOn(api, 'mergeLibraryManga').mockResolvedValue({} as any);
    vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    vi.spyOn(api, 'addProvider').mockResolvedValue({} as any);
  });

  it('calls api.mergeLibraryManga (not patchLibraryManga + addProvider) when mode="merge"', async () => {
    const { result } = renderUseMetadataComparison({
      mode: 'merge',
      incomingManga: baseIncomingManga,
      sourceMangaIds: [baseIncomingManga.id],
    });

    await waitFor(() => {
      // Wait for the useEffect that runs initComparisonState to populate state.
      expect(result.current.incomingValues.providers.length).toBeGreaterThan(0);
    });

    await act(async () => {
      await result.current.importMutation.mutateAsync();
    });

    expect(api.mergeLibraryManga).toHaveBeenCalledTimes(1);
    expect(api.patchLibraryManga).not.toHaveBeenCalled();
    expect(api.addProvider).not.toHaveBeenCalled();
  });

  it('builds a merge payload with providers: "merge" and all required metadata fields', async () => {
    const { result } = renderUseMetadataComparison({
      mode: 'merge',
      incomingManga: baseIncomingManga,
      sourceMangaIds: [baseIncomingManga.id],
    });

    await waitFor(() => {
      expect(result.current.incomingValues.providers.length).toBeGreaterThan(0);
    });

    await act(async () => {
      await result.current.importMutation.mutateAsync();
    });

    expect(api.mergeLibraryManga).toHaveBeenCalledWith(
      expect.objectContaining({
        keep_manga_id: 'm-keep',
        source_manga_ids: ['m-source'],
        metadata: expect.objectContaining({
          aliases: 'merge',
          tags: 'merge',
          authors: 'merge',
          artists: 'merge',
          publishers: 'merge',
          providers: 'merge',
          title: expect.stringMatching(/^(keep|source:m-source)$/),
          description: expect.stringMatching(/^(keep|source:m-source)$/),
          release_year: expect.stringMatching(/^(keep|source:m-source)$/),
          cover_url: expect.stringMatching(/^(keep|source:m-source)$/),
        }),
      })
    );
  });

  it('defaults source_manga_ids to [incomingManga.id] when sourceMangaIds is not provided', async () => {
    const { result } = renderUseMetadataComparison({
      mode: 'merge',
      incomingManga: baseIncomingManga,
    });

    await waitFor(() => {
      expect(result.current.incomingValues.providers.length).toBeGreaterThan(0);
    });

    await act(async () => {
      await result.current.importMutation.mutateAsync();
    });

    expect(api.mergeLibraryManga).toHaveBeenCalledWith(
      expect.objectContaining({
        source_manga_ids: ['m-source'],
      })
    );
  });
});