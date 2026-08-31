import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useAddProviderSearch } from './useAddProviderSearch';
import { api } from '../../api/client';
import * as toastContext from '../../context/ToastContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Source, Manga } from '../../types/api';
import { queryKeys } from '../../lib/queryKeys';

vi.mock('../../context/ToastContext', () => ({
  useToast: vi.fn(),
}));

const mockSources: Source[] = [
  { id: 'mangadex', name: 'MangaDex' },
  { id: 'mangafox', name: 'MangaFox' },
  { id: 'webtoons', name: 'Webtoons' },
];

describe('useAddProviderSearch', () => {
  const showToast = vi.fn();
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.mocked(toastContext.useToast).mockReturnValue({ showToast } as any);
    vi.spyOn(api, 'searchManga').mockResolvedValue({
      mangas: [{ id: 'rem-1', title: 'Solo Leveling' }],
      hasNext: false,
      page: 1,
    });
    vi.spyOn(api, 'addProvider').mockResolvedValue({
      id: 'm-1',
      title: 'Solo Leveling',
    } as any);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const createWrapper = () => {
    return ({ children }: { children: React.ReactNode }) =>
      React.createElement(QueryClientProvider, { client: queryClient }, children);
  };

  describe('Initial provider selection', () => {
    it('initializes step to pick and prioritizes unbound provider over bound ones', () => {
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            existingProviders: [{ provider_id: 'mangadex', provider_manga_id: 'md-1' }],
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      expect(result.current.step).toBe('pick');
      expect(result.current.selectedProviderId).toBe('mangafox');
      expect(result.current.selectedProvider).toEqual(mockSources[1]);
      expect(result.current.searchQuery).toBe('Solo Leveling');
    });

    it('falls back to the first source if all sources are already bound', () => {
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            existingProviders: [
              { provider_id: 'mangadex', provider_manga_id: 'md-1' },
              { provider_id: 'mangafox', provider_manga_id: 'mf-1' },
              { provider_id: 'webtoons', provider_manga_id: 'wt-1' },
            ],
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      expect(result.current.selectedProviderId).toBe('mangadex');
      expect(result.current.selectedProvider).toEqual(mockSources[0]);
    });

    it('falls back to empty string if sources list is empty', () => {
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: [],
            existingProviders: [],
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      expect(result.current.selectedProviderId).toBe('');
      expect(result.current.selectedProvider).toBeNull();
      expect(result.current.searchQuery).toBe('');
    });
  });

  describe('Debounced auto-search and query changes', () => {
    it('debounces auto-search by 300ms when searchQuery is updated', async () => {
      vi.useFakeTimers();

      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: '',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      expect(api.searchManga).not.toHaveBeenCalled();

      act(() => {
        result.current.setSearchQuery('Naruto');
      });

      // Advance by 150ms -> not called yet
      act(() => {
        vi.advanceTimersByTime(150);
      });
      expect(api.searchManga).not.toHaveBeenCalled();

      // Change query again before 300ms
      act(() => {
        result.current.setSearchQuery('Naruto Shippuden');
      });

      // Advance another 150ms -> still not called because timer reset
      act(() => {
        vi.advanceTimersByTime(150);
      });
      expect(api.searchManga).not.toHaveBeenCalled();

      // Advance 150ms more (total 300ms since last change)
      await act(async () => {
        vi.advanceTimersByTime(150);
      });

      expect(api.searchManga).toHaveBeenCalledTimes(1);
      expect(api.searchManga).toHaveBeenCalledWith('mangadex', 'Naruto Shippuden');
    });

    it('clears search results and error when query is empty or only whitespace without calling api', () => {
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Initial',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.setSearchQuery('   ');
      });

      expect(result.current.searchResults).toEqual([]);
      expect(result.current.searchError).toBeNull();
      expect(api.searchManga).not.toHaveBeenCalled();
    });

    it('does not trigger search when open is false', () => {
      vi.useFakeTimers();

      renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: false,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        vi.advanceTimersByTime(500);
      });

      expect(api.searchManga).not.toHaveBeenCalled();
    });

    it('does not trigger search when step is confirm', () => {
      vi.useFakeTimers();

      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: '',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.setStep('confirm');
        result.current.setSearchQuery('One Piece');
      });

      act(() => {
        vi.advanceTimersByTime(500);
      });

      expect(api.searchManga).not.toHaveBeenCalled();
    });
  });

  describe('Search mutation outcomes', () => {
    it('populates searchResults and clears searchError on successful search', async () => {
      const mockResults: Manga[] = [
        { id: 'rem-1', title: 'Solo Leveling' },
        { id: 'rem-2', title: 'Solo Leveling: Ragnarok' },
      ];
      vi.mocked(api.searchManga).mockResolvedValue({
        mangas: mockResults,
        hasNext: false,
        page: 1,
      });

      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.searchResults).toEqual(mockResults);
        expect(result.current.searchError).toBeNull();
      });
    });

    it('sets searchError and resets searchResults on search failure', async () => {
      vi.mocked(api.searchManga).mockRejectedValue(new Error('Rate limited'));

      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.searchError).toBe('Rate limited');
        expect(result.current.searchResults).toEqual([]);
      });
    });

    it('uses fallback error message if error has no message property', async () => {
      vi.mocked(api.searchManga).mockRejectedValue({});

      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.searchError).toBe('Search failed');
        expect(result.current.searchResults).toEqual([]);
      });
    });
  });

  describe('Step transitions and modal open state handling', () => {
    it('transitions to confirm step when a result is selected', () => {
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleResultSelect({ id: 'rem-1', title: 'Solo Leveling' });
      });

      expect(result.current.step).toBe('confirm');
      expect(result.current.selectedResult).toEqual({ id: 'rem-1', title: 'Solo Leveling' });
    });

    it('resets state completely when handleOpenChange(false) is called', () => {
      const onOpenChange = vi.fn();
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange,
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.setSelectedProviderId('webtoons');
        result.current.setSearchQuery('Custom Query');
        result.current.handleResultSelect({ id: 'rem-1', title: 'Custom Title' });
      });

      expect(result.current.step).toBe('confirm');
      expect(result.current.selectedProviderId).toBe('webtoons');
      expect(result.current.searchQuery).toBe('Custom Query');
      expect(result.current.selectedResult).not.toBeNull();

      act(() => {
        result.current.handleOpenChange(false);
      });

      expect(onOpenChange).toHaveBeenCalledWith(false);
      expect(result.current.step).toBe('pick');
      expect(result.current.selectedProviderId).toBe('mangadex');
      expect(result.current.searchQuery).toBe('Solo Leveling');
      expect(result.current.selectedResult).toBeNull();
      expect(result.current.searchResults).toEqual([]);
      expect(result.current.searchError).toBeNull();
    });

    it('calls onOpenChange(true) without resetting state when opening dialog', () => {
      const onOpenChange = vi.fn();
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: false,
            onOpenChange,
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleOpenChange(true);
      });

      expect(onOpenChange).toHaveBeenCalledWith(true);
    });

    it('resets state directly via resetState()', () => {
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.setStep('confirm');
        result.current.setSelectedProviderId('webtoons');
        result.current.resetState();
      });

      expect(result.current.step).toBe('pick');
      expect(result.current.selectedProviderId).toBe('mangadex');
    });
  });

  describe('handleConfirm and addProviderMutation', () => {
    it('executes addProviderMutation successfully with query invalidations, toast, and callbacks', async () => {
      const onSuccess = vi.fn();
      const onOpenChange = vi.fn();
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

      const addedManga: Manga = { id: 'm-1', title: 'Solo Leveling' };
      vi.mocked(api.addProvider).mockResolvedValue(addedManga);

      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange,
            onSuccess,
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleResultSelect({
          id: 'rem-99',
          contentRemoteId: 'cr-99',
          title: 'Solo Leveling',
        });
      });

      await act(async () => {
        result.current.handleConfirm();
      });

      await waitFor(() => {
        expect(api.addProvider).toHaveBeenCalledWith(
          'm-1',
          {
            provider_id: 'mangadex',
            provider_manga_id: 'rem-99',
            manga_title: 'Solo Leveling',
          },
          false
        );
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.manga.details('m-1') });
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.library.all });
        expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.chapters.list('m-1') });
        expect(showToast).toHaveBeenCalledWith('Provider "MangaDex" added', 'success');
        expect(onOpenChange).toHaveBeenCalledWith(false);
        expect(onSuccess).toHaveBeenCalledWith(addedManga);
      });
    });

    it('falls back to contentRemoteId when selectedResult has no id', async () => {
      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleResultSelect({
          contentRemoteId: 'cr-only',
          title: 'Solo Leveling',
        } as Manga);
      });

      await act(async () => {
        result.current.handleConfirm();
      });

      await waitFor(() => {
        expect(api.addProvider).toHaveBeenCalledWith(
          'm-1',
          {
            provider_id: 'mangadex',
            provider_manga_id: 'cr-only',
            manga_title: 'Solo Leveling',
          },
          false
        );
      });
    });

    it('shows error toast when addProviderMutation fails', async () => {
      vi.mocked(api.addProvider).mockRejectedValue(new Error('Provider conflict'));

      const { result } = renderHook(
        () =>
          useAddProviderSearch({
            mangaId: 'm-1',
            sources: mockSources,
            mangaTitle: 'Solo Leveling',
            open: true,
            onOpenChange: vi.fn(),
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleResultSelect({ id: 'rem-1', title: 'Solo Leveling' });
      });

      await act(async () => {
        result.current.handleConfirm();
      });

      await waitFor(() => {
        expect(showToast).toHaveBeenCalledWith('Failed to add provider: Provider conflict', 'error');
      });
    });
  });
});
