import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MergeMangaDialog } from './MergeMangaDialog';
import { api } from '../../api/client';
import * as toastContext from '../../context/ToastContext';
import * as router from '@tanstack/react-router';
import { Manga } from '../../types/api';
import { ToastProvider } from '../../context/ToastContext';

const importMetadataDialogSpy = vi.fn();

vi.mock('../../context/ToastContext', () => ({
  useToast: vi.fn(),
  ToastProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock('@tanstack/react-router', () => ({
  useNavigate: vi.fn(),
}));

vi.mock('../import-metadata/ImportMetadataDialog', () => ({
  ImportMetadataDialog: (props: any) => {
    importMetadataDialogSpy(props);
    return (
      <div
        data-testid="import-metadata-dialog"
        data-mode={props.mode}
        data-incoming-manga-id={props.incomingManga?.id ?? ''}
        data-source-manga-ids={(props.sourceMangaIds ?? []).join(',')}
        data-manga-id={props.manga?.id ?? ''}
      >
        Import metadata dialog (mocked)
      </div>
    );
  },
}));

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
    aliases: [],
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

describe('MergeMangaDialog', () => {
  const showToast = vi.fn();
  const navigate = vi.fn();
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
    vi.mocked(router.useNavigate).mockReturnValue(navigate);

    queryClient.setQueryData(['library', 'manga'], libraryMangas);
    queryClient.setQueryData(['manga', 'detail', 'm-keep'], keepManga);

    vi.spyOn(api, 'mergeLibraryManga').mockResolvedValue({
      id: 'm-keep',
      title: 'Merged',
    } as Manga);
  });

  const renderDialog = (
    props: Partial<React.ComponentProps<typeof MergeMangaDialog>> = {}
  ) => {
    const defaultProps: React.ComponentProps<typeof MergeMangaDialog> = {
      open: true,
      keepMangaId: 'm-keep',
      onOpenChange: vi.fn(),
      ...props,
    };
    return render(
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <MergeMangaDialog {...defaultProps} />
        </ToastProvider>
      </QueryClientProvider>
    );
  };

  it('renders the source picker listing all library manga except keep', () => {
    renderDialog();

    expect(screen.getByText('Merge with other manga')).toBeInTheDocument();
    expect(screen.getByText('Frieren (MAL)')).toBeInTheDocument();
    expect(screen.getByText('Frieren (MangaFox)')).toBeInTheDocument();

    // The keep manga is rendered in the header banner, so we look for it once
    // there — but it should NOT appear in the radio list.
    expect(screen.getAllByText('Frieren at the Funeral').length).toBeGreaterThan(0);
    expect(
      screen.queryByRole('radio', { name: /select frieren at the funeral/i })
    ).not.toBeInTheDocument();
  });

  it('Compare button is disabled until a source is selected', () => {
    renderDialog();

    const compareButton = screen.getByRole('button', { name: /^compare$/i });
    expect(compareButton).toBeDisabled();

    fireEvent.click(screen.getByRole('radio', { name: /select frieren \(mal\)/i }));

    expect(compareButton).not.toBeDisabled();
  });

  it('selecting a different source replaces the previous selection', () => {
    renderDialog();

    fireEvent.click(screen.getByRole('radio', { name: /select frieren \(mal\)/i }));
    expect(
      screen.getByRole('radio', { name: /select frieren \(mal\)/i })
    ).toBeChecked();

    fireEvent.click(screen.getByRole('radio', { name: /select frieren \(mangafox\)/i }));
    expect(
      screen.getByRole('radio', { name: /select frieren \(mangafox\)/i })
    ).toBeChecked();
    expect(
      screen.getByRole('radio', { name: /select frieren \(mal\)/i })
    ).not.toBeChecked();
  });

  it('clicking Compare renders ImportMetadataDialog with mode=merge + incomingManga', async () => {
    importMetadataDialogSpy.mockClear();

    renderDialog();

    fireEvent.click(screen.getByRole('radio', { name: /select frieren \(mal\)/i }));
    fireEvent.click(screen.getByRole('button', { name: /^compare$/i }));

    await waitFor(() => {
      expect(screen.getByTestId('import-metadata-dialog')).toBeInTheDocument();
    });

    const dialog = screen.getByTestId('import-metadata-dialog');
    expect(dialog).toHaveAttribute('data-mode', 'merge');
    expect(dialog).toHaveAttribute('data-incoming-manga-id', 'm-src-1');
    expect(dialog).toHaveAttribute('data-source-manga-ids', 'm-src-1');
    expect(dialog).toHaveAttribute('data-manga-id', 'm-keep');

    // The picker should no longer be visible.
    expect(screen.queryByText('Merge with other manga')).not.toBeInTheDocument();
  });

  it('search filter narrows the available source list', async () => {
    renderDialog();

    const search = screen.getByPlaceholderText('Search library...') as HTMLInputElement;
    fireEvent.change(search, { target: { value: 'MangaFox' } });

    await waitFor(() => {
      expect(screen.queryByText('Frieren (MAL)')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Frieren (MangaFox)')).toBeInTheDocument();
  });

  it('Cancel button closes the dialog', () => {
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });

    fireEvent.click(screen.getByRole('button', { name: /cancel/i }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('selecting another source while a selection exists replaces it before advancing', async () => {
    importMetadataDialogSpy.mockClear();

    renderDialog();

    fireEvent.click(screen.getByRole('radio', { name: /select frieren \(mal\)/i }));
    fireEvent.click(screen.getByRole('radio', { name: /select frieren \(mangafox\)/i }));
    fireEvent.click(screen.getByRole('button', { name: /^compare$/i }));

    await waitFor(() => {
      expect(screen.getByTestId('import-metadata-dialog')).toBeInTheDocument();
    });

    const dialog = screen.getByTestId('import-metadata-dialog');
    expect(dialog).toHaveAttribute('data-incoming-manga-id', 'm-src-2');
    expect(dialog).toHaveAttribute('data-source-manga-ids', 'm-src-2');
  });

  it('does not render ImportMetadataDialog while still on the picker', () => {
    renderDialog();

    expect(screen.queryByTestId('import-metadata-dialog')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('radio', { name: /select frieren \(mal\)/i }));
    // Still on the picker — no advance yet.
    expect(screen.queryByTestId('import-metadata-dialog')).not.toBeInTheDocument();
  });
});