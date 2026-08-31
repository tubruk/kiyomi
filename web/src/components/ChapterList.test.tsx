import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ChapterList } from './ChapterList';
import { Chapter } from '../types/api';

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
}));

const mockChapters: Chapter[] = [
  {
    id: 'ch-1',
    name: 'Chapter 1',
    number: 1,
    title: 'Chapter 1',
    providerId: 'prov-1',
    is_downloaded: true,
    downloaded_pages: 10,
    page_count: 10,
  },
  {
    id: 'ch-2',
    name: 'Chapter 2',
    number: 2,
    title: 'Chapter 2',
    providerId: 'prov-1',
    is_downloaded: false,
    downloaded_pages: 0,
    page_count: 10,
  },
];

describe('ChapterList', () => {
  it('renders header with compact download badge and provider info', () => {
    render(
      <ChapterList
        chapters={mockChapters}
        mangaId="manga-1"
        sortBy="number"
        order="asc"
        onSortByChange={vi.fn()}
        onOrderToggle={vi.fn()}
        isLoading={false}
        isError={false}
        isInLibrary={true}
        contentProviderName="MangaDex"
        onRefreshChapters={vi.fn()}
      />
    );

    // Title
    expect(screen.getByText('Chapters (2)')).toBeInTheDocument();

    // Compact Download Badge
    const badge = screen.getByTitle('1 of 2 chapters downloaded');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent('1/2');
    expect(badge).not.toHaveTextContent('downloaded');

    // Provider Info
    expect(screen.getByText(/provided by/i)).toBeInTheDocument();
    expect(screen.getByText('MangaDex')).toBeInTheDocument();

    // Refresh Button in header controls
    const refreshBtn = screen.getByRole('button', { name: /refresh chapter list/i });
    expect(refreshBtn).toBeInTheDocument();
  });
});
