import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { ErrorDetailsModal } from './ErrorDetailsModal';

describe('ErrorDetailsModal', () => {
  let writeTextMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: writeTextMock,
      },
      writable: true,
      configurable: true,
    });
  });

  it('renders modal with title, message, and details when open', () => {
    render(
      <ErrorDetailsModal
        open={true}
        onOpenChange={vi.fn()}
        title="Custom Error"
        message="Something went wrong"
        details="Stack trace: line 10 at fetch"
      />
    );

    expect(screen.getByText('Custom Error')).toBeInTheDocument();
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('Stack trace: line 10 at fetch')).toBeInTheDocument();
  });

  it('copies error message to clipboard when clicking Copy Message', async () => {
    render(
      <ErrorDetailsModal
        open={true}
        onOpenChange={vi.fn()}
        title="Fetch Failed"
        message="Network error 500"
      />
    );

    const copyBtn = screen.getByRole('button', { name: /copy message/i });
    await act(async () => {
      fireEvent.click(copyBtn);
    });

    expect(writeTextMock).toHaveBeenCalledWith('Network error 500');
    expect(screen.getByText('Copied!')).toBeInTheDocument();
  });

  it('copies full structured error when clicking Copy Full Error', async () => {
    render(
      <ErrorDetailsModal
        open={true}
        onOpenChange={vi.fn()}
        title="Timeout"
        message="Request timed out"
        details={'Detail line 1\nDetail line 2'}
      />
    );

    const copyFullBtn = screen.getByRole('button', { name: /copy full error/i });
    await act(async () => {
      fireEvent.click(copyFullBtn);
    });

    expect(writeTextMock).toHaveBeenCalledWith(
      'Title: Timeout\n\nMessage: Request timed out\n\nDetails:\nDetail line 1\nDetail line 2'
    );
    expect(screen.getByText('Copied!')).toBeInTheDocument();
  });

  it('calls onOpenChange(false) when clicking Close button in footer', () => {
    const handleOpenChange = vi.fn();
    render(
      <ErrorDetailsModal
        open={true}
        onOpenChange={handleOpenChange}
        title="Error"
        message="Something failed"
      />
    );

    const closeButtons = screen.getAllByRole('button', { name: /close/i });
    fireEvent.click(closeButtons[closeButtons.length - 1]);

    expect(handleOpenChange).toHaveBeenCalledWith(false, expect.anything());
  });
});
