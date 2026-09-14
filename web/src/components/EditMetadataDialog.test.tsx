import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { EditMetadataDialog } from './EditMetadataDialog';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { api } from '../api/client';
import { Manga } from '../types/api';
import { ToastProvider } from '../context/ToastContext';

const mockManga: Manga = {
  id: 'manga-edit-1',
  title: 'Chainsaw Man',
  aliases: ['CSM', 'Denji the Chainsaw'],
  description: 'A boy merges with a chainsaw devil.',
  authors: ['Tatsuki Fujimoto'],
  artists: ['Tatsuki Fujimoto'],
  publishers: ['Shueisha', 'VIZ Media'],
  publisher: 'Shueisha',
  readingMode: 'rtl',
  contentRating: 'mature',
  releaseYear: 2018,
  startDate: '2018-12-03',
  endDate: '2020-12-14',
  country: 'JP',
  tags: ['Action', 'Horror', 'Supernatural'],
  shelves: ['Shonen Jump', 'Top Rated'],
  externalLinks: [
    { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/105778' },
  ],
  metadata: {
    title: 'Chainsaw Man',
    aliases: ['CSM', 'Denji the Chainsaw'],
    description: 'A boy merges with a chainsaw devil.',
    authors: ['Tatsuki Fujimoto'],
    artists: ['Tatsuki Fujimoto'],
    tags: ['Action', 'Horror', 'Supernatural'],
    collections: ['Shonen Jump', 'Top Rated'],
    publishers: ['Shueisha', 'VIZ Media'],
    releaseYear: 2018,
    startDate: '2018-12-03',
    endDate: '2020-12-14',
    country: 'JP',
    content_rating: 'mature',
    externalLinks: [
      { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/105778' },
    ],
  },
  user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
  bindings: { providers: [] },
};

const renderDialog = (
  props: Partial<React.ComponentProps<typeof EditMetadataDialog>> = {}
) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  const defaultProps: React.ComponentProps<typeof EditMetadataDialog> = {
    manga: mockManga,
    open: true,
    onOpenChange: vi.fn(),
    onSaved: vi.fn(),
    ...props,
  };

  return {
    ...render(
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <EditMetadataDialog {...defaultProps} />
        </ToastProvider>
      </QueryClientProvider>
    ),
    defaultProps,
  };
};

describe('EditMetadataDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders dialog with all initial metadata and interactive TagInput chips', () => {
    renderDialog();

    expect(screen.getByText('Edit Series Metadata')).toBeInTheDocument();

    // Check title and synopsis
    expect(screen.getByDisplayValue('Chainsaw Man')).toBeInTheDocument();
    expect(screen.getByDisplayValue('A boy merges with a chainsaw devil.')).toBeInTheDocument();

    // Check TagInput chips for all array-based fields
    expect(screen.getByText('CSM')).toBeInTheDocument();
    expect(screen.getByText('Denji the Chainsaw')).toBeInTheDocument();
    expect(screen.getAllByText('Tatsuki Fujimoto').length).toBe(2); // author and artist
    expect(screen.getByText('Shueisha')).toBeInTheDocument();
    expect(screen.getByText('VIZ Media')).toBeInTheDocument();
    expect(screen.getByText('Action')).toBeInTheDocument();
    expect(screen.getByText('Horror')).toBeInTheDocument();
    expect(screen.getByText('Supernatural')).toBeInTheDocument();
    expect(screen.getByText('Shonen Jump')).toBeInTheDocument();
    expect(screen.getByText('Top Rated')).toBeInTheDocument();

    // Check dates and scalars
    expect(screen.getByDisplayValue('2018')).toBeInTheDocument();
    expect(screen.getByDisplayValue('2018-12-03')).toBeInTheDocument();
    expect(screen.getByDisplayValue('2020-12-14')).toBeInTheDocument();
    expect(screen.getByDisplayValue('JP')).toBeInTheDocument();

    // Check external links
    expect(screen.getByDisplayValue('AniList')).toBeInTheDocument();
    expect(screen.getByDisplayValue('https://anilist.co/manga/105778')).toBeInTheDocument();
  });

  it('handles adding new chips via Enter and comma across tag input fields', () => {
    renderDialog();

    // Add publisher via Enter
    const pubInput = screen.getByLabelText('Add publisher...');
    fireEvent.change(pubInput, { target: { value: 'Kodansha' } });
    fireEvent.keyDown(pubInput, { key: 'Enter' });
    expect(screen.getByText('Kodansha')).toBeInTheDocument();

    // Add publisher via comma
    fireEvent.change(pubInput, { target: { value: 'Yen Press, Glénat' } });
    fireEvent.keyDown(pubInput, { key: ',' });
    expect(screen.getByText('Yen Press')).toBeInTheDocument();
    expect(screen.getByText('Glénat')).toBeInTheDocument();

    // Add tag via Enter
    const tagInput = screen.getByLabelText('Add tag or genre...');
    fireEvent.change(tagInput, { target: { value: 'Dark Fantasy' } });
    fireEvent.keyDown(tagInput, { key: 'Enter' });
    expect(screen.getByText('Dark Fantasy')).toBeInTheDocument();

    // Add alias via Enter
    const aliasInput = screen.getByLabelText('Add alternative title...');
    fireEvent.change(aliasInput, { target: { value: 'Chainsaw Hero' } });
    fireEvent.keyDown(aliasInput, { key: 'Enter' });
    expect(screen.getByText('Chainsaw Hero')).toBeInTheDocument();

    // Add author via Enter
    const authorInput = screen.getByLabelText('Add author...');
    fireEvent.change(authorInput, { target: { value: 'Assistant Author' } });
    fireEvent.keyDown(authorInput, { key: 'Enter' });
    expect(screen.getByText('Assistant Author')).toBeInTheDocument();

    // Add artist via Enter
    const artistInput = screen.getByLabelText('Add artist...');
    fireEvent.change(artistInput, { target: { value: 'Colorist Artist' } });
    fireEvent.keyDown(artistInput, { key: 'Enter' });
    expect(screen.getByText('Colorist Artist')).toBeInTheDocument();

    // Add shelf via Enter
    const shelfInput = screen.getByLabelText('Add shelf or collection...');
    fireEvent.change(shelfInput, { target: { value: 'Favorites' } });
    fireEvent.keyDown(shelfInput, { key: 'Enter' });
    expect(screen.getByText('Favorites')).toBeInTheDocument();
  });

  it('handles removing chips via (x) button click', () => {
    renderDialog();

    expect(screen.getByText('Shueisha')).toBeInTheDocument();
    const removeShueishaBtn = screen.getByRole('button', { name: 'Remove Shueisha' });
    fireEvent.click(removeShueishaBtn);
    expect(screen.queryByText('Shueisha')).not.toBeInTheDocument();

    expect(screen.getByText('Horror')).toBeInTheDocument();
    const removeHorrorBtn = screen.getByRole('button', { name: 'Remove Horror' });
    fireEvent.click(removeHorrorBtn);
    expect(screen.queryByText('Horror')).not.toBeInTheDocument();

    expect(screen.getByText('CSM')).toBeInTheDocument();
    const removeCsmBtn = screen.getByRole('button', { name: 'Remove CSM' });
    fireEvent.click(removeCsmBtn);
    expect(screen.queryByText('CSM')).not.toBeInTheDocument();
  });

  it('submits form sending updated array fields, start/end dates, and scalar values', async () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    const onOpenChange = vi.fn();
    const onSaved = vi.fn();

    renderDialog({
      onOpenChange,
      onSaved,
    });

    // Update title
    const titleInput = screen.getByDisplayValue('Chainsaw Man');
    fireEvent.change(titleInput, { target: { value: 'Chainsaw Man Part 2' } });

    // Update dates
    const startDateInput = screen.getByDisplayValue('2018-12-03');
    fireEvent.change(startDateInput, { target: { value: '2022-07-13' } });

    const endDateInput = screen.getByDisplayValue('2020-12-14');
    fireEvent.change(endDateInput, { target: { value: '2025-01-01' } });

    // Update release year and country
    const yearInput = screen.getByDisplayValue('2018');
    fireEvent.change(yearInput, { target: { value: '2022' } });

    const countryInput = screen.getByDisplayValue('JP');
    fireEvent.change(countryInput, { target: { value: 'US' } });

    // Add a new publisher chip
    const pubInput = screen.getByLabelText('Add publisher...');
    fireEvent.change(pubInput, { target: { value: 'Shogakukan' } });
    fireEvent.keyDown(pubInput, { key: 'Enter' });

    // Remove VIZ Media publisher chip
    const removeVizBtn = screen.getByRole('button', { name: 'Remove VIZ Media' });
    fireEvent.click(removeVizBtn);

    // Submit form
    const saveBtn = screen.getByRole('button', { name: /save changes/i });
    fireEvent.click(saveBtn);

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith(
        'manga-edit-1',
        expect.objectContaining({
          metadata: expect.objectContaining({
            title: 'Chainsaw Man Part 2',
            publishers: ['Shueisha', 'Shogakukan'],
            publisher: 'Shueisha',
            startDate: '2022-07-13',
            start_date: '2022-07-13',
            endDate: '2025-01-01',
            end_date: '2025-01-01',
            releaseYear: 2022,
            release_year: 2022,
            country: 'US',
            authors: ['Tatsuki Fujimoto'],
            artists: ['Tatsuki Fujimoto'],
            tags: ['Action', 'Horror', 'Supernatural'],
            shelves: ['Shonen Jump', 'Top Rated'],
            collections: ['Shonen Jump', 'Top Rated'],
          }),
        })
      );
      expect(onOpenChange).toHaveBeenCalledWith(false);
      expect(onSaved).toHaveBeenCalled();
    });
  });

  it('handles adding and removing external links in form', async () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);

    renderDialog();

    // Click Add Link
    const addLinkBtn = screen.getByRole('button', { name: /add link/i });
    fireEvent.click(addLinkBtn);

    // Enter URL for the newly added link
    const urlInputs = screen.getAllByPlaceholderText('https://...');
    expect(urlInputs.length).toBe(2);
    fireEvent.change(urlInputs[1], { target: { value: 'https://myanimelist.net/manga/116778' } });

    // Submit
    const saveBtn = screen.getByRole('button', { name: /save changes/i });
    fireEvent.click(saveBtn);

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith(
        'manga-edit-1',
        expect.objectContaining({
          metadata: expect.objectContaining({
            externalLinks: [
              { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/105778' },
              { provider: 'custom', label: 'Custom Link', url: 'https://myanimelist.net/manga/116778' },
            ],
          }),
        })
      );
    });
  });

  it('cancels without saving when Cancel button is clicked', () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    const onOpenChange = vi.fn();

    renderDialog({ onOpenChange });

    const cancelBtn = screen.getByRole('button', { name: /cancel/i });
    fireEvent.click(cancelBtn);

    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(patchSpy).not.toHaveBeenCalled();
  });
});
