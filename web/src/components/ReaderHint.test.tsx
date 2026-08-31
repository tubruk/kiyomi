import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ReaderHint } from './ReaderHint';

describe('ReaderHint', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders nothing when hint is null or not visible', () => {
    const { container: c1 } = render(<ReaderHint hint={null} onDismiss={vi.fn()} />);
    expect(c1.firstChild).toBeNull();

    const { container: c2 } = render(
      <ReaderHint
        hint={{ readingMode: 'rtl', message: 'Tap to read', visible: false }}
        onDismiss={vi.fn()}
      />
    );
    expect(c2.firstChild).toBeNull();
  });

  it('renders mode label and custom message for RTL mode', () => {
    render(
      <ReaderHint
        hint={{ readingMode: 'rtl', message: 'Tap left/right to navigate', visible: true }}
        onDismiss={vi.fn()}
      />
    );

    expect(screen.getByText('Right-to-Left')).toBeInTheDocument();
    expect(screen.getByText('Tap left/right to navigate')).toBeInTheDocument();
  });

  it('renders mode label for Vertical mode', () => {
    render(
      <ReaderHint
        hint={{ readingMode: 'vertical', message: 'Scroll to navigate', visible: true }}
        onDismiss={vi.fn()}
      />
    );

    expect(screen.getByText('Vertical Scroll')).toBeInTheDocument();
    expect(screen.getByText('Scroll to navigate')).toBeInTheDocument();
  });

  it('calls onDismiss when clicked', () => {
    const handleDismiss = vi.fn();
    render(
      <ReaderHint
        hint={{ readingMode: 'ltr', message: 'Tap to read', visible: true }}
        onDismiss={handleDismiss}
      />
    );

    const overlay = screen.getByText('Left-to-Right').closest('div.fixed');
    expect(overlay).toBeInTheDocument();

    if (overlay) {
      fireEvent.click(overlay);
    }
    expect(handleDismiss).toHaveBeenCalledTimes(1);
  });

  it('auto dismisses after 2 seconds', () => {
    const handleDismiss = vi.fn();
    render(
      <ReaderHint
        hint={{ readingMode: 'rtl', message: 'Tap to read', visible: true }}
        onDismiss={handleDismiss}
      />
    );

    expect(handleDismiss).not.toHaveBeenCalled();

    vi.advanceTimersByTime(2000);
    expect(handleDismiss).toHaveBeenCalledTimes(1);
  });
});
