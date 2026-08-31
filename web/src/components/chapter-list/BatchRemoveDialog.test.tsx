import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BatchRemoveDialog } from './BatchRemoveDialog';

describe('BatchRemoveDialog', () => {
  it('renders modal with singular chapter count description', () => {
    render(
      <BatchRemoveDialog
        open={true}
        onOpenChange={vi.fn()}
        selectedCount={1}
        onConfirm={vi.fn()}
        isRemoving={false}
      />
    );

    expect(screen.getByText('Remove Chapters')).toBeInTheDocument();
    expect(
      screen.getByText(/Are you sure you want to remove 1 selected chapter from your library\?/i)
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove 1 Chapter' })).toBeInTheDocument();
  });

  it('renders modal with plural chapter count description', () => {
    render(
      <BatchRemoveDialog
        open={true}
        onOpenChange={vi.fn()}
        selectedCount={5}
        onConfirm={vi.fn()}
        isRemoving={false}
      />
    );

    expect(
      screen.getByText(/Are you sure you want to remove 5 selected chapters from your library\?/i)
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove 5 Chapters' })).toBeInTheDocument();
  });

  it('calls onConfirm when confirm button clicked', async () => {
    const user = userEvent.setup();
    const handleConfirm = vi.fn();
    render(
      <BatchRemoveDialog
        open={true}
        onOpenChange={vi.fn()}
        selectedCount={2}
        onConfirm={handleConfirm}
        isRemoving={false}
      />
    );

    await user.click(screen.getByRole('button', { name: 'Remove 2 Chapters' }));
    expect(handleConfirm).toHaveBeenCalledTimes(1);
  });

  it('disables buttons during isRemoving', () => {
    render(
      <BatchRemoveDialog
        open={true}
        onOpenChange={vi.fn()}
        selectedCount={2}
        onConfirm={vi.fn()}
        isRemoving={true}
      />
    );

    expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /remove 2 chapters/i })).toBeDisabled();
  });
});
