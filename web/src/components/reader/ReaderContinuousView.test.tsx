import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ReaderContinuousView } from './ReaderContinuousView';
import { Page } from '../../types/api';

describe('ReaderContinuousView', () => {
  const mockPages: Page[] = [
    { index: 0, url: 'http://example.com/p1.jpg' },
    { index: 1, url: 'http://example.com/p2.jpg' },
  ];

  it('renders pages in vertical mode with page indicators', () => {
    const pageRefs = { current: [] };
    const onToggleOverlays = vi.fn();

    render(
      <ReaderContinuousView
        readingMode="vertical"
        pages={mockPages}
        chapterId="ch-1"
        fitMode="fit-width"
        pageRefs={pageRefs}
        onToggleOverlays={onToggleOverlays}
        hasPrevChapter={true}
        hasNextChapter={true}
      />
    );

    expect(screen.getByTestId('reader-content')).toBeInTheDocument();
    expect(screen.getByAltText('Page 1')).toBeInTheDocument();
    expect(screen.getByAltText('Page 2')).toBeInTheDocument();
    expect(screen.getByText('p.1')).toBeInTheDocument();
    expect(screen.getByText('p.2')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-card-prev')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-card-next')).toBeInTheDocument();
  });

  it('renders pages in longstrip mode without page number labels', () => {
    const pageRefs = { current: [] };

    render(
      <ReaderContinuousView
        readingMode="longstrip"
        pages={mockPages}
        chapterId="ch-1"
        fitMode="fit-width"
        pageRefs={pageRefs}
        onToggleOverlays={vi.fn()}
      />
    );

    expect(screen.getByAltText('Page 1')).toBeInTheDocument();
    expect(screen.queryByText('p.1')).not.toBeInTheDocument();
  });

  it('triggers onToggleOverlays when clicking container', () => {
    const pageRefs = { current: [] };
    const onToggleOverlays = vi.fn();

    render(
      <ReaderContinuousView
        readingMode="vertical"
        pages={mockPages}
        chapterId="ch-1"
        fitMode="fit-height"
        pageRefs={pageRefs}
        onToggleOverlays={onToggleOverlays}
      />
    );

    fireEvent.click(screen.getByTestId('reader-content'));
    expect(onToggleOverlays).toHaveBeenCalledTimes(1);
  });
});
