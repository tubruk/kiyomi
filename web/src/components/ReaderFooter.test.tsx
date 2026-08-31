import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ReaderFooter } from './ReaderFooter';

describe('ReaderFooter', () => {
  it('renders page indicator and jump button for LTR mode', () => {
    const handleScrollTop = vi.fn();
    render(
      <ReaderFooter
        currentPage={5}
        totalPages={20}
        readingMode="ltr"
        fitMode="fit-height"
        onPageChange={vi.fn()}
        onScrollTop={handleScrollTop}
        onFitModeChange={vi.fn()}
        onReadingModeChange={vi.fn()}
      />
    );

    expect(screen.getByTestId('page-indicator')).toHaveTextContent('5 / 20');

    const jumpBtn = screen.getByRole('button', { name: 'Scroll to Top' });
    expect(jumpBtn).toBeInTheDocument();

    fireEvent.click(jumpBtn);
    expect(handleScrollTop).toHaveBeenCalledTimes(1);
  });

  it('renders Scroll to End button for RTL mode', () => {
    const handleScrollTop = vi.fn();
    render(
      <ReaderFooter
        currentPage={5}
        totalPages={20}
        readingMode="rtl"
        fitMode="fit-height"
        onPageChange={vi.fn()}
        onScrollTop={handleScrollTop}
        onFitModeChange={vi.fn()}
        onReadingModeChange={vi.fn()}
      />
    );

    const jumpBtn = screen.getByRole('button', { name: 'Scroll to End' });
    expect(jumpBtn).toBeInTheDocument();
  });
});
