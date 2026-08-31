import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ChapterDownloadBadge } from './ChapterDownloadBadge';

describe('ChapterDownloadBadge', () => {
  it('returns null when no status is active', () => {
    const { container } = render(<ChapterDownloadBadge />);
    expect(container.firstChild).toBeNull();
  });

  it('renders removing state spinner', () => {
    render(<ChapterDownloadBadge isRemoving={true} />);
    const icon = screen.getByLabelText('Removing');
    expect(icon).toBeInTheDocument();
    expect(icon.closest('div')).toHaveAttribute('title', 'Removing chapter from library...');
  });

  it('renders deleting files state spinner', () => {
    render(<ChapterDownloadBadge isDeletingFiles={true} />);
    const icon = screen.getByLabelText('Deleting files');
    expect(icon).toBeInTheDocument();
    expect(icon.closest('div')).toHaveAttribute('title', 'Deleting chapter files...');
  });

  it('renders pulling state without percentage when progress unknown', () => {
    render(<ChapterDownloadBadge isPulling={true} />);
    const icon = screen.getByLabelText('Pulling chapter');
    expect(icon).toBeInTheDocument();
    expect(icon.closest('div')).toHaveAttribute('title', 'Pulling chapter...');
  });

  it('renders pulling progress percentage when percent is provided', () => {
    render(<ChapterDownloadBadge isPulling={true} percent={45} />);
    expect(screen.getByText('45%')).toBeInTheDocument();
    expect(screen.getByLabelText('Pulling progress')).toBeInTheDocument();
  });

  it('computes percentage from downloadedPages and pageCount', () => {
    render(
      <ChapterDownloadBadge
        isPartiallyDownloaded={true}
        downloadedPages={10}
        pageCount={20}
      />
    );
    expect(screen.getByText('50%')).toBeInTheDocument();
    expect(screen.getByLabelText('Pulling progress')).toBeInTheDocument();
  });

  it('renders downloaded state with hard drive icon and details tooltip', () => {
    render(
      <ChapterDownloadBadge
        isDownloaded={true}
        downloadedPages={25}
        pageCount={25}
      />
    );
    const icon = screen.getByLabelText('Downloaded to disk');
    expect(icon).toBeInTheDocument();
    expect(icon.closest('div')).toHaveAttribute('title', 'Downloaded (25/25 pages)');
  });

  it('renders downloaded fallback tooltip when page counts missing', () => {
    render(<ChapterDownloadBadge isDownloaded={true} />);
    expect(screen.getByLabelText('Downloaded to disk').closest('div')).toHaveAttribute(
      'title',
      'Downloaded (Local storage)'
    );
  });
});
