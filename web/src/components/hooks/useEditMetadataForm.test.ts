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
  description: 'Soul reaper story',
  readingMode: 'rtl',
  contentRating: 'safe',
  publisher: 'Shueisha',
  releaseYear: 2001,
  country: 'JP',
  tags: ['Action', 'Supernatural'],
  shelves: ['Shonen'],
  externalLinks: [{ provider: 'anilist', label: 'AniList', url: 'https://anilist.co' }],
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
    expect(result.current.aliasesInput).toBe('Kurosaki');
    expect(result.current.description).toBe('Soul reaper story');
    expect(result.current.readingMode).toBe('rtl');
    expect(result.current.publisher).toBe('Shueisha');
    expect(result.current.releaseYear).toBe(2001);
    expect(result.current.tagsInput).toBe('Action, Supernatural');
    expect(result.current.shelvesInput).toBe('Shonen');
    expect(result.current.externalLinks.length).toBe(1);
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

  it('submits serialized payload and triggers mutation', () => {
    const onOpenChange = vi.fn();
    const { result } = renderHook(() =>
      useEditMetadataForm({
        manga: mockManga,
        open: true,
        onOpenChange,
      })
    );

    act(() => {
      result.current.setTitle('Bleach Updated');
      result.current.setAliasesInput('Kurosaki, Substitute');
    });

    act(() => {
      result.current.handleSubmit({ preventDefault: vi.fn() } as any);
    });

    expect(mutate).toHaveBeenCalledWith(
      expect.objectContaining({
        mangaId: 'm-1',
        fields: expect.objectContaining({
          title: 'Bleach Updated',
          aliases: ['Kurosaki', 'Substitute'],
        }),
      }),
      expect.any(Object)
    );
  });
});
