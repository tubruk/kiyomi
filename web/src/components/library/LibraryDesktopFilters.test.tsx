import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LibraryDesktopFilters } from './LibraryDesktopFilters';
import type { ShelfItem } from './hooks/useLibraryFilters';

const mockShelves: ShelfItem[] = [
  { id: 'all', label: 'All' },
  { id: 'favorites', label: 'Favorites', isFavorite: true },
  { id: 'reading', label: 'Reading' },
  { id: 'completed', label: 'Completed' },
];

const mockCounts: Record<string, number> = {
  all: 25,
  favorites: 5,
  reading: 12,
  completed: 8,
};

const mockTags = ['Action', 'Romance', 'Fantasy'];

describe('LibraryDesktopFilters', () => {
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

  it('renders shelf filter tabs with counts when hasManga is true', () => {
    render(<LibraryDesktopFilters {...defaultProps} />);

    expect(screen.getByRole('tab', { name: /all/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /favorites/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /reading/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /completed/i })).toBeInTheDocument();

    expect(screen.getByText('25')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('8')).toBeInTheDocument();
  });

  it('does not render shelf filter tabs when hasManga is false', () => {
    render(<LibraryDesktopFilters {...defaultProps} hasManga={false} />);

    expect(screen.queryByRole('tab', { name: /all/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: /favorites/i })).not.toBeInTheDocument();
  });

  it('calls onSelectShelf when a shelf tab is clicked', () => {
    const onSelectShelf = vi.fn();
    render(<LibraryDesktopFilters {...defaultProps} onSelectShelf={onSelectShelf} />);

    const favoritesTab = screen.getByRole('tab', { name: /favorites/i });
    fireEvent.click(favoritesTab);

    expect(onSelectShelf).toHaveBeenCalledWith('favorites', expect.anything());
  });

  it('renders sort selector with current sort label', () => {
    render(<LibraryDesktopFilters {...defaultProps} sortBy="added_desc" />);

    expect(screen.getByText('Recently Added')).toBeInTheDocument();
  });

  it('changes sort option via sort dropdown', async () => {
    const user = userEvent.setup();
    const onSortByChange = vi.fn();
    render(
      <LibraryDesktopFilters
        {...defaultProps}
        onSortByChange={onSortByChange}
      />
    );

    const sortTrigger = screen.getByText('Recently Added');
    await user.click(sortTrigger);

    const titleAscOption = await screen.findByRole('option', { name: 'Title (A to Z)' });
    await user.click(titleAscOption);

    expect(onSortByChange).toHaveBeenCalledWith('title_asc');
  });

  it('renders tag filter dropdown and calls onSelectTag when a tag is selected', async () => {
    const user = userEvent.setup();
    const onSelectTag = vi.fn();
    render(
      <LibraryDesktopFilters
        {...defaultProps}
        onSelectTag={onSelectTag}
      />
    );

    const tagTrigger = screen.getByText('All Tags');
    await user.click(tagTrigger);

    const actionOption = await screen.findByRole('option', { name: 'Action' });
    await user.click(actionOption);

    expect(onSelectTag).toHaveBeenCalledWith('Action');
  });

  it('does not render tag dropdown when allTags is empty', () => {
    render(<LibraryDesktopFilters {...defaultProps} allTags={[]} />);

    expect(screen.queryByText('All Tags')).not.toBeInTheDocument();
  });

  it('renders selected tag indicator badge and clears tag when × is clicked', () => {
    const onSelectTag = vi.fn();
    render(
      <LibraryDesktopFilters
        {...defaultProps}
        selectedTag="Romance"
        onSelectTag={onSelectTag}
      />
    );

    expect(screen.getByText('Filtered by tag:')).toBeInTheDocument();
    expect(screen.getAllByText('Romance').length).toBeGreaterThanOrEqual(1);

    const clearTagBtn = screen.getByRole('button', { name: '×' });
    fireEvent.click(clearTagBtn);

    expect(onSelectTag).toHaveBeenCalledWith('');
  });

  it('handles search input typing and calls onSearchChange', () => {
    const onSearchChange = vi.fn();
    render(
      <LibraryDesktopFilters
        {...defaultProps}
        filterSearch=""
        onSearchChange={onSearchChange}
      />
    );

    const searchInput = screen.getByPlaceholderText('Search by title or author...');
    fireEvent.change(searchInput, { target: { value: 'Berserk' } });

    expect(onSearchChange).toHaveBeenCalledWith('Berserk');
  });
});
