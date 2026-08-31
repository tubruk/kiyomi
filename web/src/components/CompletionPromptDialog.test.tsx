import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CompletionPromptDialog } from './CompletionPromptDialog';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { api } from '@/api/client';

const queryClient = new QueryClient();

const renderDialog = (props: React.ComponentProps<typeof CompletionPromptDialog>) =>
  render(
    <QueryClientProvider client={queryClient}>
      <CompletionPromptDialog {...props} />
    </QueryClientProvider>
  );

describe('CompletionPromptDialog', () => {
  it('renders dialog with manga title', () => {
    renderDialog({
      open: true,
      mangaId: 'manga-1',
      mangaTitle: 'Frieren: Beyond Journey\'s End',
      onClose: vi.fn(),
    });

    expect(screen.getByText("You've finished this manga!")).toBeInTheDocument();
    expect(
      screen.getByText(/"Frieren: Beyond Journey's End" is now complete\. Would you like to mark it as completed\?/i)
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /not now/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /mark completed/i })).toBeInTheDocument();
  });

  it('renders fallback description when title is not provided', () => {
    renderDialog({
      open: true,
      mangaId: 'manga-1',
      onClose: vi.fn(),
    });

    expect(
      screen.getByText('Would you like to mark this manga as completed?')
    ).toBeInTheDocument();
  });

  it('calls onClose when "Not Now" is clicked', () => {
    const handleClose = vi.fn();
    renderDialog({
      open: true,
      mangaId: 'manga-1',
      onClose: handleClose,
    });

    fireEvent.click(screen.getByRole('button', { name: /not now/i }));
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('triggers mutation and calls onClose when "Mark Completed" is clicked', async () => {
    const patchSpy = vi.spyOn(api, 'patchLibraryManga').mockResolvedValue({} as any);
    const handleClose = vi.fn();

    renderDialog({
      open: true,
      mangaId: 'manga-123',
      onClose: handleClose,
    });

    fireEvent.click(screen.getByRole('button', { name: /mark completed/i }));

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith('manga-123', {
        user_status: 'completed',
        meta: { user_status: 'completed' },
      });
    });
    expect(handleClose).toHaveBeenCalledTimes(1);
  });
});
