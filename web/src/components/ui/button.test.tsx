import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './button';

describe('UI/Button', () => {
  it('renders with children and triggers click event', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click Me</Button>);

    const button = screen.getByRole('button', { name: 'Click Me' });
    expect(button).toBeInTheDocument();
    expect(button).toHaveClass('bg-primary');

    await user.click(button);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('renders disabled state and prevents click', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();
    render(<Button disabled onClick={handleClick}>Disabled</Button>);

    const button = screen.getByRole('button', { name: 'Disabled' });
    expect(button).toBeDisabled();

    await user.click(button);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('renders variants and sizes', () => {
    const { rerender } = render(<Button variant="destructive" size="sm">Small Destructive</Button>);
    const button = screen.getByRole('button', { name: 'Small Destructive' });
    expect(button).toHaveClass('text-destructive');
    expect(button).toHaveClass('h-7');

    rerender(<Button variant="outline" size="lg">Large Outline</Button>);
    const outlineBtn = screen.getByRole('button', { name: 'Large Outline' });
    expect(outlineBtn).toHaveClass('border-border');
    expect(outlineBtn).toHaveClass('h-9');
  });
});
