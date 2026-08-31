import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import {
  ImportMetadataDialog,
  areArraySetsEqual,
  mergeStringArrays,
  areExternalLinkSetsEqual,
  mergeExternalLinkArrays,
} from './ImportMetadataDialog';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { api } from '../api/client';
import { Manga, Source } from '../types/api';
import { ToastProvider } from '../context/ToastContext';

const mockSources: Source[] = [
  {
    id: 'mangadex',
    name: 'MangaDex',
    capabilities: ['content', 'metadata'],
  },
  {
    id: 'anilist',
    name: 'AniList',
    capabilities: ['metadata'],
  },
];

const mockManga: Manga = {
  id: 'local-manga-1',
  title: 'Frieren at the Funeral',
  description: 'A story of an elf mage after the hero defeated the demon king.',
  coverUrl: 'https://example.com/current-cover.jpg',
  authors: ['Kanehito Yamada'],
  artists: ['Tsukasa Abe'],
  tags: ['Fantasy', 'Adventure', 'Drama'],
  aliases: ['Sousou no Frieren'],
  publisher: 'Shogakukan',
  releaseYear: 2020,
  contentRating: 'safe',
  country: 'JP',
  readingMode: 'rtl',
  meta: {
    providers: [
      {
        provider_id: 'mangadex',
        provider_manga_id: 'md-frieren-123',
        manga_title: 'Frieren at the Funeral',
      },
    ],
  },
};

const mockRemoteManga: Manga = {
  id: 'remote-1',
  title: 'Frieren: Beyond Journey\'s End',
  description: 'The adventure is over but life goes on for an elf mage.',
  coverUrl: 'https://example.com/remote-cover.jpg',
  authors: ['Kanehito Yamada', 'Author 2'],
  artists: ['Tsukasa Abe'],
  tags: ['Fantasy', 'Magic', 'Adventure'],
  aliases: ['Frieren the Slayer'],
  publisher: 'VIZ Media',
  releaseYear: 2021,
  contentRating: 'suggestive',
  country: 'JP',
  readingMode: 'rtl',
};

const renderDialog = (
  props: Partial<React.ComponentProps<typeof ImportMetadataDialog>> = {}
) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const defaultProps: React.ComponentProps<typeof ImportMetadataDialog> = {
    manga: mockManga,
    sources: mockSources,
    open: true,
    onOpenChange: vi.fn(),
    ...props,
  };

  return {
    ...render(
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <ImportMetadataDialog {...defaultProps} />
        </ToastProvider>
      </QueryClientProvider>
    ),
    defaultProps,
  };
};

describe('ImportMetadataDialog helper utilities', () => {
  it('correctly compares array sets case-insensitively and unordered', () => {
    expect(areArraySetsEqual(['Fantasy', 'Adventure'], ['adventure', 'FANTASY'])).toBe(true);
    expect(areArraySetsEqual(['Fantasy', 'Adventure'], ['Fantasy', 'Magic'])).toBe(false);
    expect(areArraySetsEqual([], [])).toBe(true);
    expect(areArraySetsEqual(undefined, [])).toBe(true);
    expect(areArraySetsEqual(['A', 'B'], ['a'])).toBe(false);
  });

  it('merges string arrays deduplicating case-insensitively', () => {
    const merged = mergeStringArrays(['Fantasy', 'Magic'], ['fantasy', 'Adventure', 'MAGIC']);
    expect(merged).toEqual(['Fantasy', 'Magic', 'Adventure']);
  });

  it('correctly compares external link sets by URL case-insensitively and unordered', () => {
    expect(
      areExternalLinkSetsEqual(
        [{ provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' }],
        [{ provider: 'custom', label: 'MD', url: 'HTTPS://MANGADEX.ORG/TITLE/1' }]
      )
    ).toBe(true);
    expect(
      areExternalLinkSetsEqual(
        [{ provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' }],
        [{ provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/1' }]
      )
    ).toBe(false);
    expect(areExternalLinkSetsEqual([], [])).toBe(true);
    expect(areExternalLinkSetsEqual(undefined, [])).toBe(true);
  });

  it('merges external link arrays deduplicating by URL case-insensitively', () => {
    const merged = mergeExternalLinkArrays(
      [{ provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/1' }],
      [
        { provider: 'anilist', label: 'Duplicate', url: 'https://ANILIST.CO/MANGA/1' },
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
      ]
    );
    expect(merged).toEqual([
      { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/1' },
      { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
    ]);
  });
});

describe('ImportMetadataDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders search step with bound provider badges and mode switcher', () => {
    renderDialog();

    expect(screen.getByText('Import Metadata')).toBeInTheDocument();
    expect(screen.getByText('Keyword Search')).toBeInTheDocument();
    expect(screen.getByText('Remote ID / URL')).toBeInTheDocument();
  });

  it('performs keyword search and transitions to comparison upon selecting a result', async () => {
    vi.spyOn(api, 'searchManga').mockResolvedValue({
      mangas: [mockRemoteManga],
      hasNext: false,
      page: 1,
    });
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog();

    // Trigger search
    const searchBtn = screen.getByRole('button', { name: /^search$/i });
    fireEvent.click(searchBtn);

    await waitFor(() => {
      expect(screen.getByText('Frieren: Beyond Journey\'s End')).toBeInTheDocument();
    });

    // Select search result
    fireEvent.click(screen.getByText('Frieren: Beyond Journey\'s End'));

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
      expect(screen.getByText('Diff only')).toBeInTheDocument();
      expect(screen.getByText('Show all')).toBeInTheDocument();
    });
  });

  it('performs direct Remote ID / URL lookup and transitions to comparison', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog();

    // Switch to direct ID/URL mode
    fireEvent.click(screen.getByText('Remote ID / URL'));

    const input = screen.getByPlaceholderText(/enter remote id or full url/i);
    fireEvent.change(input, { target: { value: 'remote-123' } });

    const lookupBtn = screen.getByRole('button', { name: /lookup/i });
    fireEvent.click(lookupBtn);

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
      expect(screen.getAllByText('Frieren: Beyond Journey\'s End').length).toBeGreaterThan(0);
    });
  });

  it('auto-fetches when initialProviderId and initialRemoteId are provided', async () => {
    const detailsSpy = vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(detailsSpy).toHaveBeenCalledWith('mangadex', 'md-123');
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });
  });

  it('hides identical fields when Diff only is active and shows them when Show all is selected', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue({
      ...mockManga,
      id: 'remote-1',
      title: 'Frieren at the Funeral', // identical title
      description: 'A story of an elf mage after the hero defeated the demon king.', // identical
      authors: ['kanehito yamada'], // identical case-insensitive
      artists: ['tsukasa abe'], // identical
      tags: ['drama', 'adventure', 'fantasy'], // identical unordered set
      publisher: 'shogakukan', // identical
      releaseYear: 2020, // identical
    });

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // In Diff only mode with identical data, diff banner is shown
    expect(screen.getByText(/all incoming metadata matches your current manga/i)).toBeInTheDocument();

    // Switch to Show all
    fireEvent.click(screen.getByText('Show all'));

    expect(screen.getByText('Title')).toBeInTheDocument();
    expect(screen.getByText('Authors')).toBeInTheDocument();
    expect(screen.getByText('Attributes')).toBeInTheDocument();
  });

  it('supports title inline "+ Add to Aliases" action and hides when already present case-insensitively', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // Initially selectedTitle is 'incoming', target is current title ('Frieren at the Funeral')
    const addCurrentAliasBtn = screen.getByRole('button', { name: /add current to aliases/i });
    expect(addCurrentAliasBtn).toBeInTheDocument();
    fireEvent.click(addCurrentAliasBtn);

    // After adding, 'Frieren at the Funeral' is in aliases, so button disappears
    expect(screen.queryByRole('button', { name: /add current to aliases/i })).not.toBeInTheDocument();
    expect(screen.getByTitle('Remove "Frieren at the Funeral"')).toBeInTheDocument();

    // Switch selected title to 'current' -> target title is incoming ('Frieren: Beyond Journey\'s End')
    const keepCurrentTitleCard = screen.getByRole('button', { name: /keep current frieren at the funeral/i });
    fireEvent.click(keepCurrentTitleCard);

    // Now "+ Add Incoming to Aliases" should appear
    const addIncomingAliasBtn = screen.getByRole('button', { name: /add incoming to aliases/i });
    expect(addIncomingAliasBtn).toBeInTheDocument();
    fireEvent.click(addIncomingAliasBtn);

    // Button should now be hidden as incoming title is in aliases
    expect(screen.queryByRole('button', { name: /add incoming to aliases/i })).not.toBeInTheDocument();
    expect(screen.getByTitle('Remove "Frieren: Beyond Journey\'s End"')).toBeInTheDocument();
  });

  it('hides "+ Add to Aliases" button initially if target title is already in aliases case-insensitively', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue({
      ...mockRemoteManga,
      // Target alias when incoming is selected will be current title ('Frieren at the Funeral')
      // If aliases already has it in lowercase:
      aliases: ['frieren at the funeral'],
    });

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // Button should be hidden from the start because 'frieren at the funeral' matches case-insensitively
    expect(screen.queryByRole('button', { name: /add (incoming|current) to aliases/i })).not.toBeInTheDocument();
  });

  it('supports typeable pill input for adding custom aliases on Enter and comma', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText(/^aliases \(/i)).toBeInTheDocument();
    });

    const aliasInput = screen.getByLabelText('Add alias');

    // Add via Enter
    fireEvent.change(aliasInput, { target: { value: 'Custom Alias One' } });
    fireEvent.keyDown(aliasInput, { key: 'Enter' });

    expect(screen.getByText('Custom Alias One')).toBeInTheDocument();

    // Add via comma
    fireEvent.change(aliasInput, { target: { value: 'Custom Alias Two, Custom Alias Three' } });
    fireEvent.keyDown(aliasInput, { key: ',' });

    expect(screen.getByText('Custom Alias Two')).toBeInTheDocument();
    expect(screen.getByText('Custom Alias Three')).toBeInTheDocument();
  });

  it('supports typeable pill input for adding custom tags on Enter and comma', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText(/^tags & genres \(/i)).toBeInTheDocument();
    });

    const tagInput = screen.getByLabelText('Add tag');

    // Add via Enter
    fireEvent.change(tagInput, { target: { value: 'Supernatural' } });
    fireEvent.keyDown(tagInput, { key: 'Enter' });

    expect(screen.getByText('Supernatural')).toBeInTheDocument();

    // Add via comma
    fireEvent.change(tagInput, { target: { value: 'Slice of Life, Psychological' } });
    fireEvent.keyDown(tagInput, { key: ',' });

    expect(screen.getByText('Slice of Life')).toBeInTheDocument();
    expect(screen.getByText('Psychological')).toBeInTheDocument();
  });

  it('supports interactive tag pill removal, quick action buttons, and click-to-re-add', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText(/tags & genres/i)).toBeInTheDocument();
    });

    // Remove "Magic" tag pill
    const removeMagicBtn = screen.getByTitle('Remove tag "Magic"');
    fireEvent.click(removeMagicBtn);

    // "Magic" should now be in unselected tags
    expect(screen.getByRole('button', { name: /^magic$/i })).toBeInTheDocument();

    // Click "Magic" to re-add
    fireEvent.click(screen.getByRole('button', { name: /^magic$/i }));
    expect(screen.getByTitle('Remove tag "Magic"')).toBeInTheDocument();

    // Test quick buttons
    fireEvent.click(screen.getByRole('button', { name: 'Clear' }));
    expect(screen.queryByTitle('Remove tag "Magic"')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Merge' }));
    expect(screen.getByTitle('Remove tag "Magic"')).toBeInTheDocument();
    expect(screen.getByTitle('Remove tag "Adventure"')).toBeInTheDocument();
  });

  it('supports interactive alias pill removal and click-to-re-add', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText(/^aliases \(/i)).toBeInTheDocument();
    });

    // Dismiss "Sousou no Frieren"
    const removeBtn = screen.getByTitle('Remove "Sousou no Frieren"');
    fireEvent.click(removeBtn);

    // Click available chip to re-add
    const reAddChip = screen.getByRole('button', { name: /^sousou no frieren$/i });
    fireEvent.click(reAddChip);

    expect(screen.getByTitle('Remove "Sousou no Frieren"')).toBeInTheDocument();
  });

  it('supports quick actions "Accept Incoming" and "Keep Current"', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // Click Keep Current
    fireEvent.click(screen.getByTitle('Keep all current metadata'));

    // Click Accept Incoming
    fireEvent.click(screen.getByTitle('Accept all incoming metadata'));
  });

  it('submits patch and adds provider binding on confirmation serializing finalized aliases and tags', async () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    const addProviderSpy = vi.spyOn(api, 'addProvider').mockResolvedValue({} as any);
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(mockRemoteManga);

    const onOpenChange = vi.fn();
    const onSuccess = vi.fn();

    renderDialog({
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
      onOpenChange,
      onSuccess,
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // Add custom alias and custom tag
    const aliasInput = screen.getByLabelText('Add alias');
    fireEvent.change(aliasInput, { target: { value: 'Custom Final Alias' } });
    fireEvent.keyDown(aliasInput, { key: 'Enter' });

    const tagInput = screen.getByLabelText('Add tag');
    fireEvent.change(tagInput, { target: { value: 'Custom Final Tag' } });
    fireEvent.keyDown(tagInput, { key: 'Enter' });

    // Submit import
    const submitBtn = screen.getByRole('button', { name: /import & bind provider/i });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith('local-manga-1', expect.objectContaining({
        aliases: expect.arrayContaining(['Sousou no Frieren', 'Frieren the Slayer', 'Custom Final Alias']),
        tags: expect.arrayContaining(['Fantasy', 'Adventure', 'Drama', 'Magic', 'Custom Final Tag']),
      }));
      expect(addProviderSpy).toHaveBeenCalledWith('local-manga-1', {
        provider_id: 'mangadex',
        provider_manga_id: 'remote-1',
        manga_title: 'Frieren: Beyond Journey\'s End',
      });
      expect(onOpenChange).toHaveBeenCalledWith(false);
      expect(onSuccess).toHaveBeenCalled();
    });
  });

  it('routes search result and comparison cover images through getProxyImageUrl with fallback handling', async () => {
    const remoteWithUrl: Manga = {
      ...mockRemoteManga,
      url: 'https://mangadex.org/title/remote-1',
    };
    vi.spyOn(api, 'searchManga').mockResolvedValue({
      mangas: [remoteWithUrl],
      hasNext: false,
      page: 1,
    });
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue(remoteWithUrl);

    renderDialog();

    // Trigger search
    fireEvent.click(screen.getByRole('button', { name: /^search$/i }));

    await waitFor(() => {
      expect(screen.getByText('Frieren: Beyond Journey\'s End')).toBeInTheDocument();
    });

    // Check search result cover img src
    const searchResultImg = document.querySelector('img[src*="/api/v1/proxy/image"]') as HTMLImageElement;
    expect(searchResultImg).toBeInTheDocument();
    expect(searchResultImg.src).toContain(encodeURIComponent('https://example.com/remote-cover.jpg'));
    expect(searchResultImg.src).toContain(encodeURIComponent('https://mangadex.org/title/remote-1'));

    // Trigger onError on search result img
    fireEvent.error(searchResultImg);
    expect(searchResultImg.style.display).toBe('none');

    // Click search result to transition to comparison
    fireEvent.click(screen.getByText('Frieren: Beyond Journey\'s End'));

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // Compare step: check current cover and incoming cover imgs
    const coverImgs = Array.from(document.querySelectorAll('img')).filter((img) =>
      img.src.includes('cover')
    );
    expect(coverImgs.length).toBe(2);

    const incomingImg = coverImgs.find((img) =>
      img.src.includes(encodeURIComponent('https://example.com/remote-cover.jpg'))
    );
    expect(incomingImg).toBeDefined();

    // Trigger onError on incoming img
    if (incomingImg) {
      fireEvent.error(incomingImg);
      expect(incomingImg.style.display).toBe('none');
    }
  });

  it('auto-populates external tracking link in patchLibraryManga when remote manga has URL', async () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    vi.spyOn(api, 'addProvider').mockResolvedValue({} as any);
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue({
      ...mockRemoteManga,
      url: 'https://mangadex.org/title/remote-1',
    });

    renderDialog({
      manga: {
        ...mockManga,
        externalLinks: [
          { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/123' },
        ],
      },
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /import & bind provider/i }));

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith(
        'local-manga-1',
        expect.objectContaining({
          externalLinks: [
            { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/123' },
            { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/remote-1' },
          ],
        })
      );
    });
  });

  it('does not duplicate external link if provider URL already exists in manga.externalLinks', async () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    vi.spyOn(api, 'addProvider').mockResolvedValue({} as any);
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue({
      ...mockRemoteManga,
      url: 'https://mangadex.org/title/remote-1',
    });

    renderDialog({
      manga: {
        ...mockManga,
        externalLinks: [
          { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/remote-1' },
        ],
      },
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /import & bind provider/i }));

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith(
        'local-manga-1',
        expect.not.objectContaining({
          externalLinks: expect.anything(),
        })
      );
    });
  });

  it('does not render empty badge container when search results have no tags', async () => {
    vi.spyOn(api, 'searchManga').mockResolvedValue({
      mangas: [
        {
          id: 'no-tags-1',
          title: 'No Tags Manga',
          author: 'Author A',
          tags: [],
          genres: [],
        },
      ],
      hasNext: false,
      page: 1,
    });

    renderDialog();

    fireEvent.click(screen.getByRole('button', { name: /^search$/i }));

    await waitFor(() => {
      expect(screen.getByText('No Tags Manga')).toBeInTheDocument();
    });

    // The item should have title and author, but no tag badge container
    const authorEl = screen.getByText('Author A');
    expect(authorEl.parentElement?.querySelector('.overflow-hidden')).toBeNull();
  });

  it('renders external links comparison section and supports switching between Current, Incoming, and Merged modes', async () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    vi.spyOn(api, 'addProvider').mockResolvedValue({} as any);
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue({
      ...mockRemoteManga,
      url: 'https://mangadex.org/title/remote-1',
    });

    renderDialog({
      manga: {
        ...mockManga,
        externalLinks: [
          { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/123' },
        ],
      },
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // Check that External Links section is visible
    expect(screen.getByText('External Links')).toBeInTheDocument();

    // Switch to 'incoming' mode by clicking the incoming card
    const incomingCard = screen.getByRole('button', { name: /incoming:\s*\(1\)/i });
    fireEvent.click(incomingCard);

    // Submit import
    fireEvent.click(screen.getByRole('button', { name: /import & bind provider/i }));

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith(
        'local-manga-1',
        expect.objectContaining({
          external_links: [
            { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/remote-1' },
          ],
          externalLinks: [
            { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/remote-1' },
          ],
        })
      );
    });
  });

  it('hides external links in diff only mode when link sets are equal, and shows them in show all mode', async () => {
    vi.spyOn(api, 'getProviderMangaDetails').mockResolvedValue({
      ...mockRemoteManga,
      url: 'https://mangadex.org/title/remote-1',
    });

    renderDialog({
      manga: {
        ...mockManga,
        // Current matches incoming exactly
        externalLinks: [
          { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/remote-1' },
        ],
        // Match other fields to have no diffs or test diff only
      },
      initialProviderId: 'mangadex',
      initialRemoteId: 'md-123',
    });

    await waitFor(() => {
      expect(screen.getByText('Compare & Import Metadata')).toBeInTheDocument();
    });

    // In Diff only mode, since external links are equal, the External Links section should not be rendered
    expect(screen.queryByText('External Links')).not.toBeInTheDocument();

    // Switch to Show all
    fireEvent.click(screen.getByText('Show all'));

    // Now External Links section should be visible
    expect(screen.getByText('External Links')).toBeInTheDocument();
  });
});


