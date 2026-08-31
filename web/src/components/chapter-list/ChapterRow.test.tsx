import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ChapterRow } from './ChapterRow';
import { Chapter } from '../../types/api';

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
}));

const mockChapter: Chapter = {
  id: 'ch-1',
  name: 'Chapter 1: The Beginning',
  number: 1,
  title: 'Chapter 1: The Beginning',
  providerId: 'provider-1',
  is_downloaded: true,
  downloaded_pages: 10,
  page_count: 10,
};

describe('ChapterRow', () => {
  it('renders chapter details correctly', () => {
    render(
      <ChapterRow
        chapter={mockChapter}
        mangaId="manga-1"
        isSelectionMode={false}
        isSelected={false}
        isProcessing={false}
        isPulling={false}
        isDeletingFiles={false}
        isRemoving={false}
        openMenuId={null}
        onOpenMenuChange={vi.fn()}
        onToggleSelect={vi.fn()}
      />
    );

    expect(screen.getByText('1')).toBeInTheDocument();
    expect(screen.getByText('Chapter 1: The Beginning')).toBeInTheDocument();
  });

  it('renders dropdown menu items with correct icons and alignment when open', () => {
    const handlePull = vi.fn();
    const handleDelete = vi.fn();
    const handleRemove = vi.fn();

    render(
      <ChapterRow
        chapter={mockChapter}
        mangaId="manga-1"
        isInLibrary={true}
        isSelectionMode={false}
        isSelected={false}
        isProcessing={false}
        isPulling={false}
        isDeletingFiles={false}
        isRemoving={false}
        openMenuId="ch-1"
        onOpenMenuChange={vi.fn()}
        onToggleSelect={vi.fn()}
        onPullChapter={handlePull}
        onDeleteFiles={handleDelete}
        onRemoveChapter={handleRemove}
      />
    );

    const pullItem = screen.getByRole('menuitem', { name: /pull files/i });
    expect(pullItem).toBeInTheDocument();
    const pullSvg = pullItem.querySelector('svg');
    expect(pullSvg).toHaveClass('lucide-download');

    const deleteFilesItem = screen.getByRole('menuitem', { name: /delete files/i });
    expect(deleteFilesItem).toBeInTheDocument();
    const deleteSvg = deleteFilesItem.querySelector('svg');
    expect(deleteSvg).toHaveClass('lucide-file-x');

    const removeItem = screen.getByRole('menuitem', { name: /remove chapter/i });
    expect(removeItem).toBeInTheDocument();
    const removeSvg = removeItem.querySelector('svg');
    expect(removeSvg).toHaveClass('lucide-trash-2');
    expect(removeItem).not.toHaveClass('justify-center');
    expect(removeItem).toHaveClass('text-destructive');
  });

  it('hides delete files menu item when chapter has no files on disk', () => {
    const chapterWithoutFiles: Chapter = {
      ...mockChapter,
      is_downloaded: false,
      downloaded_pages: 0,
    };

    render(
      <ChapterRow
        chapter={chapterWithoutFiles}
        mangaId="manga-1"
        isInLibrary={true}
        isSelectionMode={false}
        isSelected={false}
        isProcessing={false}
        isPulling={false}
        isDeletingFiles={false}
        isRemoving={false}
        openMenuId="ch-1"
        onOpenMenuChange={vi.fn()}
        onToggleSelect={vi.fn()}
        onPullChapter={vi.fn()}
        onDeleteFiles={vi.fn()}
        onRemoveChapter={vi.fn()}
      />
    );

    expect(screen.getByRole('menuitem', { name: /pull files/i })).toBeInTheDocument();
    expect(screen.queryByRole('menuitem', { name: /delete files/i })).not.toBeInTheDocument();
    expect(screen.getByRole('menuitem', { name: /remove chapter/i })).toBeInTheDocument();
  });

  it('renders spinning loader for pulling state in dropdown menu', () => {
    render(
      <ChapterRow
        chapter={mockChapter}
        mangaId="manga-1"
        isInLibrary={true}
        isSelectionMode={false}
        isSelected={false}
        isProcessing={true}
        isPulling={true}
        isDeletingFiles={false}
        isRemoving={false}
        openMenuId="ch-1"
        onOpenMenuChange={vi.fn()}
        onToggleSelect={vi.fn()}
        onPullChapter={vi.fn()}
      />
    );

    const pullItem = screen.getByRole('menuitem', { name: /pull files/i });
    const pullSvg = pullItem.querySelector('svg');
    expect(pullSvg).toHaveClass('animate-spin');
  });

  it('renders spinning loader for deleting files state in dropdown menu', () => {
    render(
      <ChapterRow
        chapter={mockChapter}
        mangaId="manga-1"
        isInLibrary={true}
        isSelectionMode={false}
        isSelected={false}
        isProcessing={true}
        isPulling={false}
        isDeletingFiles={true}
        isRemoving={false}
        openMenuId="ch-1"
        onOpenMenuChange={vi.fn()}
        onToggleSelect={vi.fn()}
        onDeleteFiles={vi.fn()}
      />
    );

    const deleteItem = screen.getByRole('menuitem', { name: /delete files/i });
    const deleteSvg = deleteItem.querySelector('svg');
    expect(deleteSvg).toHaveClass('animate-spin');
  });
});
