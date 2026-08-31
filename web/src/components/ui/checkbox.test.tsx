import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Checkbox } from './checkbox';

describe('UI/Checkbox', () => {
  it('renders unchecked checkbox by default', () => {
    render(<Checkbox aria-label="Accept terms" />);
    const cb = screen.getByRole('checkbox', { name: 'Accept terms' });
    expect(cb).toBeInTheDocument();
    expect(cb).toHaveAttribute('data-slot', 'checkbox');
  });

  it('renders custom className', () => {
    render(<Checkbox aria-label="Option" className="custom-check" />);
    const cb = screen.getByRole('checkbox', { name: 'Option' });
    expect(cb).toHaveClass('custom-check');
  });
});
