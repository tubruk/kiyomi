import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ChapterListHeader } from './ChapterListHeader';

describe('ChapterListHeader', () => {
  it('renders chapter count, download count badge, and provider name', () => {
    render(
      <ChapterListHeader
        chapterCount={25}
        downloadedCount={10}
        contentProviderName="MangaDex (EN)"
        isInLibrary={true}
        isSelectionMode={false}
        onEnterSelectionMode={vi.fn()}
        filterQuery=""
        onFilterQueryChange={vi.fn()}
        sortBy="number"
        order="asc"
        onSortByChange={vi.fn()}
        onOrderToggle={vi.fn()}
      />
    );

    expect(screen.getByText('Chapters (25)')).toBeInTheDocument();
    expect(screen.getByText('10/25')).toBeInTheDocument();
    expect(screen.getByText('Provided by')).toBeInTheDocument();
    expect(screen.getByText('MangaDex (EN)')).toBeInTheDocument();
  });

  it('triggers onFilterQueryChange when user types in search input', () => {
    const onFilterQueryChange = vi.fn();
    render(
      <ChapterListHeader
        chapterCount={10}
        isSelectionMode={false}
        onEnterSelectionMode={vi.fn()}
        filterQuery=""
        onFilterQueryChange={onFilterQueryChange}
        sortBy="source"
        order="asc"
        onSortByChange={vi.fn()}
        onOrderToggle={vi.fn()}
      />
    );

    const input = screen.getByPlaceholderText('Filter chapters...');
    fireEvent.change(input, { target: { value: 'Chapter 5' } });
    expect(onFilterQueryChange).toHaveBeenCalledWith('Chapter 5');
  });

  it('triggers onOrderToggle when order button is clicked', () => {
    const onOrderToggle = vi.fn();
    render(
      <ChapterListHeader
        chapterCount={10}
        isSelectionMode={false}
        onEnterSelectionMode={vi.fn()}
        filterQuery=""
        onFilterQueryChange={vi.fn()}
        sortBy="source"
        order="desc"
        onSortByChange={vi.fn()}
        onOrderToggle={onOrderToggle}
      />
    );

    const orderBtn = screen.getByRole('button', { name: /sort descending/i });
    fireEvent.click(orderBtn);
    expect(onOrderToggle).toHaveBeenCalled();
  });

  it('triggers onRefreshChapters when refresh button is clicked', () => {
    const onRefresh = vi.fn();
    render(
      <ChapterListHeader
        chapterCount={10}
        isSelectionMode={false}
        onEnterSelectionMode={vi.fn()}
        filterQuery=""
        onFilterQueryChange={vi.fn()}
        sortBy="source"
        order="desc"
        onSortByChange={vi.fn()}
        onOrderToggle={vi.fn()}
        onRefreshChapters={onRefresh}
      />
    );

    const refreshBtn = screen.getByRole('button', { name: /refresh chapter list/i });
    fireEvent.click(refreshBtn);
    expect(onRefresh).toHaveBeenCalled();
  });

  it('shows select button when in library and triggers onEnterSelectionMode', () => {
    const onEnterSelectionMode = vi.fn();
    render(
      <ChapterListHeader
        chapterCount={10}
        isInLibrary={true}
        isSelectionMode={false}
        onEnterSelectionMode={onEnterSelectionMode}
        filterQuery=""
        onFilterQueryChange={vi.fn()}
        sortBy="source"
        order="desc"
        onSortByChange={vi.fn()}
        onOrderToggle={vi.fn()}
      />
    );

    const selectBtn = screen.getByRole('button', { name: /select chapters/i });
    fireEvent.click(selectBtn);
    expect(onEnterSelectionMode).toHaveBeenCalled();
  });
});
