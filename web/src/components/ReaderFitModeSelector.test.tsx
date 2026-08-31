import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderFitModeSelector } from './ReaderFitModeSelector';

describe('ReaderFitModeSelector', () => {
  it('renders trigger button and is initially closed', () => {
    render(
      <ReaderFitModeSelector
        fitMode="fit-height"
        readingMode="rtl"
        onFitModeChange={vi.fn()}
        onReadingModeChange={vi.fn()}
      />
    );

    const trigger = screen.getByRole('button', { name: /reader settings/i });
    expect(trigger).toBeInTheDocument();
    expect(trigger).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('opens menu on button click and allows selecting fit mode', async () => {
    const user = userEvent.setup();
    const handleFitModeChange = vi.fn();
    const handleReadingModeChange = vi.fn();

    render(
      <ReaderFitModeSelector
        fitMode="fit-height"
        readingMode="rtl"
        onFitModeChange={handleFitModeChange}
        onReadingModeChange={handleReadingModeChange}
      />
    );

    const trigger = screen.getByRole('button', { name: /reader settings/i });
    await user.click(trigger);

    expect(trigger).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('menu')).toBeInTheDocument();

    const fitWidthOption = screen.getByRole('menuitem', { name: /fit width/i });
    await user.click(fitWidthOption);

    expect(handleFitModeChange).toHaveBeenCalledWith('fit-width');
  });

  it('allows selecting reading direction', async () => {
    const user = userEvent.setup();
    const handleFitModeChange = vi.fn();
    const handleReadingModeChange = vi.fn();

    render(
      <ReaderFitModeSelector
        fitMode="fit-height"
        readingMode="rtl"
        onFitModeChange={handleFitModeChange}
        onReadingModeChange={handleReadingModeChange}
      />
    );

    await user.click(screen.getByRole('button', { name: /reader settings/i }));

    const verticalOption = screen.getByRole('menuitem', { name: /vertical/i });
    await user.click(verticalOption);

    expect(handleReadingModeChange).toHaveBeenCalledWith('vertical');
  });

  it('closes menu when clicking outside', async () => {
    const user = userEvent.setup();
    render(
      <div>
        <div data-testid="outside">Outside area</div>
        <ReaderFitModeSelector
          fitMode="fit-height"
          readingMode="rtl"
          onFitModeChange={vi.fn()}
          onReadingModeChange={vi.fn()}
        />
      </div>
    );

    await user.click(screen.getByRole('button', { name: /reader settings/i }));
    expect(screen.getByRole('menu')).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByTestId('outside'));
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });
});
