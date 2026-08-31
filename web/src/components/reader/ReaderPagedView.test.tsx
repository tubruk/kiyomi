import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ReaderPagedView } from './ReaderPagedView';
import { Page } from '../../types/api';

describe('ReaderPagedView', () => {
  const mockPages: Page[] = [
    { index: 0, url: 'http://example.com/p1.jpg' },
    { index: 1, url: 'http://example.com/p2.jpg' },
    { index: 2, url: 'http://example.com/p3.jpg' },
  ];

  it('renders 3 slots and center page image', () => {
    render(
      <ReaderPagedView
        readingMode="rtl"
        currentPage={2}
        pages={mockPages}
        chapterId="ch-1"
        fitMode="fit-height"
        hasPrevChapter={true}
        hasNextChapter={true}
        dragOffset={0}
        isDragging={false}
        isAnimating={false}
        onNextPage={vi.fn()}
        onPrevPage={vi.fn()}
        onToggleOverlays={vi.fn()}
      />
    );

    expect(screen.getByTestId('reader-content')).toBeInTheDocument();
    expect(screen.getByTestId('reader-paged-container')).toBeInTheDocument();
    expect(screen.getByTestId('reader-slot-left')).toBeInTheDocument();
    expect(screen.getByTestId('reader-slot-center')).toBeInTheDocument();
    expect(screen.getByTestId('reader-slot-right')).toBeInTheDocument();
    expect(screen.getByAltText('Page 2')).toBeInTheDocument();
  });

  it('handles navigation zone clicks in RTL mode', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();
    const onToggleOverlays = vi.fn();

    render(
      <ReaderPagedView
        readingMode="rtl"
        currentPage={2}
        pages={mockPages}
        chapterId="ch-1"
        fitMode="fit-height"
        hasPrevChapter={true}
        hasNextChapter={true}
        dragOffset={0}
        isDragging={false}
        isAnimating={false}
        onNextPage={onNextPage}
        onPrevPage={onPrevPage}
        onToggleOverlays={onToggleOverlays}
      />
    );

    // Left zone in RTL = Next Page
    fireEvent.click(screen.getByTestId('reader-zone-left'));
    expect(onNextPage).toHaveBeenCalledTimes(1);

    // Right zone in RTL = Prev Page
    fireEvent.click(screen.getByTestId('reader-zone-right'));
    expect(onPrevPage).toHaveBeenCalledTimes(1);

    // Center zone = Toggle Overlays
    fireEvent.click(screen.getByTestId('reader-zone-center'));
    expect(onToggleOverlays).toHaveBeenCalledTimes(1);
  });

  it('handles navigation zone clicks in LTR mode', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();
    const onToggleOverlays = vi.fn();

    render(
      <ReaderPagedView
        readingMode="ltr"
        currentPage={2}
        pages={mockPages}
        chapterId="ch-1"
        fitMode="fit-height"
        hasPrevChapter={true}
        hasNextChapter={true}
        dragOffset={0}
        isDragging={false}
        isAnimating={false}
        onNextPage={onNextPage}
        onPrevPage={onPrevPage}
        onToggleOverlays={onToggleOverlays}
      />
    );

    // Left zone in LTR = Prev Page
    fireEvent.click(screen.getByTestId('reader-zone-left'));
    expect(onPrevPage).toHaveBeenCalledTimes(1);

    // Right zone in LTR = Next Page
    fireEvent.click(screen.getByTestId('reader-zone-right'));
    expect(onNextPage).toHaveBeenCalledTimes(1);
  });

  it('renders boundary cards when on the edge pages', () => {
    render(
      <ReaderPagedView
        readingMode="rtl"
        currentPage={1}
        pages={mockPages}
        chapterId="ch-1"
        fitMode="fit-height"
        hasPrevChapter={false}
        hasNextChapter={true}
        dragOffset={0}
        isDragging={false}
        isAnimating={false}
        onNextPage={vi.fn()}
        onPrevPage={vi.fn()}
        onToggleOverlays={vi.fn()}
      />
    );

    expect(screen.getByTestId('boundary-card-prev')).toBeInTheDocument();
  });
});
