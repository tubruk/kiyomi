import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DetailsActionBar } from './DetailsActionBar';
import type { Manga } from '../../types/api';

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children, to, ...props }: any) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));

const mergeDialogPropsSpy = vi.fn();

vi.mock('@/components/merge-manga/MergeMangaDialog', () => ({
  MergeMangaDialog: (props: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    keepMangaId: string;
  }) => {
    mergeDialogPropsSpy(props);
    if (!props.open) return null;
    return (
      <div data-testid="merge-manga-dialog" data-keep-manga-id={props.keepMangaId}>
        Merge dialog open
      </div>
    );
  },
}));

const emptyMetadata = {
  title: '',
  aliases: [],
  description: '',
  authors: [],
  artists: [],
  tags: [],
  collections: [],
  publishers: [],
};
const emptyUserState = { status: 'reading' as const, rating: 0, favorite: false, notes: '' };
const emptyBindings = { providers: [] };

const makeManga = (overrides: Partial<Manga>): Manga => ({
  id: 'm-1',
  title: '',
  metadata: emptyMetadata,
  user_state: emptyUserState,
  bindings: emptyBindings,
  ...overrides,
});

describe('DetailsActionBar', () => {
  it('renders "Back to Explore" and "Add to Library" on remote routes', () => {
    const onAdd = vi.fn();
    render(
      <DetailsActionBar
        isRemoteRoute={true}
        providerIdParam="mangafox"
        isInLibrary={false}
        manga={makeManga({ id: 'm-1', title: 'Remote Manga' })}
        onAddToLibrary={onAdd}
      />
    );

    expect(screen.getByText('Back to Explore')).toBeInTheDocument();
    const addBtn = screen.getByRole('button', { name: /add to library/i });
    expect(addBtn).toBeInTheDocument();

    fireEvent.click(addBtn);
    expect(onAdd).toHaveBeenCalled();
  });

  it('renders "View in Library" link when remote manga is already in library', () => {
    render(
      <DetailsActionBar
        isRemoteRoute={true}
        providerIdParam="mangafox"
        isInLibrary={true}
        targetMangaId="local-1"
        manga={makeManga({ id: 'local-1', title: 'Saved Manga' })}
      />
    );

    expect(screen.getByText('View in Library')).toBeInTheDocument();
  });

  it('renders "Back to Library" and reading CTA button on local library view', () => {
    const onPrimaryCta = vi.fn();
    render(
      <DetailsActionBar
        isRemoteRoute={false}
        isInLibrary={true}
        targetMangaId="local-1"
        manga={makeManga({ id: 'local-1', title: 'Local Manga' })}
        hasChapters={true}
        readingCta={{ label: 'Continue Ch. 5', chapterId: 'ch-5', page: 1 }}
        onPrimaryCta={onPrimaryCta}
      />
    );

    expect(screen.getByText('Back to Library')).toBeInTheDocument();
    const ctaBtn = screen.getByRole('button', { name: /continue ch\. 5/i });
    expect(ctaBtn).toBeInTheDocument();

    fireEvent.click(ctaBtn);
    expect(onPrimaryCta).toHaveBeenCalled();
  });

  it('renders the "Merge with..." menu item on library manga routes', () => {
    render(
      <DetailsActionBar
        isRemoteRoute={false}
        isInLibrary={true}
        targetMangaId="local-1"
        manga={makeManga({ id: 'local-1', title: 'Local Manga' })}
        onRemoveFromLibrary={() => {}}
      />
    );

    // Dropdown items only mount once the menu trigger is opened.
    fireEvent.click(screen.getByRole('button', { name: /metadata options/i }));
    expect(screen.getByText('Merge with...')).toBeInTheDocument();
  });

  it('does NOT render the "Merge with..." menu item on remote routes', () => {
    render(
      <DetailsActionBar
        isRemoteRoute={true}
        providerIdParam="mangafox"
        isInLibrary={false}
        manga={makeManga({ id: 'm-1', title: 'Remote Manga' })}
        onAddToLibrary={() => {}}
      />
    );

    // No dropdown trigger is rendered on remote routes.
    expect(screen.queryByRole('button', { name: /metadata options/i })).not.toBeInTheDocument();
    expect(screen.queryByText('Merge with...')).not.toBeInTheDocument();
  });

  it('clicking "Merge with..." opens the dialog with the correct keepMangaId', () => {
    mergeDialogPropsSpy.mockClear();

    render(
      <DetailsActionBar
        isRemoteRoute={false}
        isInLibrary={true}
        targetMangaId="local-42"
        manga={makeManga({ id: 'local-42', title: 'Local Manga' })}
        onRemoveFromLibrary={() => {}}
      />
    );

    // Dialog should be closed initially.
    expect(screen.queryByTestId('merge-manga-dialog')).not.toBeInTheDocument();
    const lastBefore = mergeDialogPropsSpy.mock.calls.at(-1)?.[0];
    expect(lastBefore?.open).toBe(false);

    // Open the dropdown menu and click the Merge with... item.
    fireEvent.click(screen.getByRole('button', { name: /metadata options/i }));
    fireEvent.click(screen.getByText('Merge with...'));

    // Dialog should now be open with the keepMangaId set from targetMangaId.
    const dialog = screen.getByTestId('merge-manga-dialog');
    expect(dialog).toBeInTheDocument();
    expect(dialog).toHaveAttribute('data-keep-manga-id', 'local-42');

    const lastAfter = mergeDialogPropsSpy.mock.calls.at(-1)?.[0];
    expect(lastAfter?.open).toBe(true);
    expect(lastAfter?.keepMangaId).toBe('local-42');
  });
});
