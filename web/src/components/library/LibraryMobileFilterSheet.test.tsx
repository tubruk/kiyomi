import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LibraryMobileFilterSheet } from './LibraryMobileFilterSheet';
import type { ShelfItem } from './hooks/useLibraryFilters';

const mockShelves: ShelfItem[] = [
  { id: 'all', label: 'All' },
  { id: 'favorites', label: 'Favorites', isFavorite: true },
  { id: 'reading', label: 'Reading' },
  { id: 'completed', label: 'Completed' },
];

const mockCounts: Record<string, number> = {
  all: 30,
  favorites: 8,
  reading: 15,
  completed: 7,
};

const mockTags = ['Action', 'Comedy', 'Drama'];

describe('LibraryMobileFilterSheet', () => {
  const defaultProps = {
    hasManga: true,
    allShelves: mockShelves,
    counts: mockCounts,
    activeShelf: 'all',
    onSelectShelf: vi.fn(),
    sortBy: 'added_desc',
    onSortByChange: vi.fn(),
    allTags: mockTags,
    selectedTag: '',
    onSelectTag: vi.fn(),
    filterSearch: '',
    onSearchChange: vi.fn(),
  };

  it('renders search input and calls onSearchChange on input', () => {
    const onSearchChange = vi.fn();
    render(
      <LibraryMobileFilterSheet
        {...defaultProps}
        onSearchChange={onSearchChange}
      />
    );

    const searchInput = screen.getByPlaceholderText('Search...');
    expect(searchInput).toBeInTheDocument();

    fireEvent.change(searchInput, { target: { value: 'One Piece' } });
    expect(onSearchChange).toHaveBeenCalledWith('One Piece');
  });

  it('opens sheet drawer when Filter & Sort button is clicked', async () => {
    const user = userEvent.setup();
    render(<LibraryMobileFilterSheet {...defaultProps} />);

    const filterBtn = screen.getByRole('button', { name: /filter & sort/i });
    expect(filterBtn).toBeInTheDocument();

    await user.click(filterBtn);

    expect(screen.getByRole('heading', { name: 'Filter & Sort' })).toBeInTheDocument();
    expect(screen.getByText('Sort By')).toBeInTheDocument();
    expect(screen.getByText('Shelf / Status')).toBeInTheDocument();
    expect(screen.getByText('Filter by Tag')).toBeInTheDocument();
  });

  it('renders shelf pills inside sheet and calls onSelectShelf on click', async () => {
    const user = userEvent.setup();
    const onSelectShelf = vi.fn();
    render(
      <LibraryMobileFilterSheet
        {...defaultProps}
        onSelectShelf={onSelectShelf}
      />
    );

    const filterBtn = screen.getByRole('button', { name: /filter & sort/i });
    await user.click(filterBtn);

    expect(screen.getByText('All (30)')).toBeInTheDocument();
    expect(screen.getByText('Favorites (8)')).toBeInTheDocument();
    expect(screen.getByText('Reading (15)')).toBeInTheDocument();
    expect(screen.getByText('Completed (7)')).toBeInTheDocument();

    const favoritesPill = screen.getByText('Favorites (8)');
    await user.click(favoritesPill);

    expect(onSelectShelf).toHaveBeenCalledWith('favorites');
  });

  it('renders tag pills inside sheet and calls onSelectTag on click', async () => {
    const user = userEvent.setup();
    const onSelectTag = vi.fn();
    render(
      <LibraryMobileFilterSheet
        {...defaultProps}
        onSelectTag={onSelectTag}
      />
    );

    const filterBtn = screen.getByRole('button', { name: /filter & sort/i });
    await user.click(filterBtn);

    const actionTag = screen.getByRole('button', { name: 'Action' });
    await user.click(actionTag);
    expect(onSelectTag).toHaveBeenCalledWith('Action');

    const allTagsBtn = screen.getByRole('button', { name: 'All Tags' });
    await user.click(allTagsBtn);
    expect(onSelectTag).toHaveBeenCalledWith('');
  });

  it('does not render Shelf section when hasManga is false', async () => {
    const user = userEvent.setup();
    render(<LibraryMobileFilterSheet {...defaultProps} hasManga={false} />);

    const filterBtn = screen.getByRole('button', { name: /filter & sort/i });
    await user.click(filterBtn);

    expect(screen.queryByText('Shelf / Status')).not.toBeInTheDocument();
  });

  it('does not render Tag section when allTags is empty', async () => {
    const user = userEvent.setup();
    render(<LibraryMobileFilterSheet {...defaultProps} allTags={[]} />);

    const filterBtn = screen.getByRole('button', { name: /filter & sort/i });
    await user.click(filterBtn);

    expect(screen.queryByText('Filter by Tag')).not.toBeInTheDocument();
  });
});
