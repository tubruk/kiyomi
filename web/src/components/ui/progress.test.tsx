import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Progress, ProgressLabel, ProgressValue } from './progress';

describe('UI/Progress', () => {
  it('renders progress bar with track and indicator', () => {
    render(<Progress value={60} data-testid="prog" />);
    const root = screen.getByTestId('prog');
    expect(root).toBeInTheDocument();
    expect(root).toHaveAttribute('data-slot', 'progress');

    const track = root.querySelector('[data-slot="progress-track"]');
    expect(track).toBeInTheDocument();

    const indicator = root.querySelector('[data-slot="progress-indicator"]');
    expect(indicator).toBeInTheDocument();
  });

  it('renders progress with custom label and value children', () => {
    render(
      <Progress value={75} data-testid="prog">
        <ProgressLabel>Downloading</ProgressLabel>
        <ProgressValue />
      </Progress>
    );

    expect(screen.getByText('Downloading')).toBeInTheDocument();
    expect(screen.getByTestId('prog')).toBeInTheDocument();
  });
});
