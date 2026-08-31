import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Input } from './input';

describe('UI/Input', () => {
  it('renders input with placeholder and handles changes', () => {
    const handleChange = vi.fn();
    render(<Input placeholder="Search manga..." onChange={handleChange} />);
    const input = screen.getByPlaceholderText('Search manga...');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('data-slot', 'input');

    fireEvent.change(input, { target: { value: 'Berserk' } });
    expect(handleChange).toHaveBeenCalled();
  });

  it('renders disabled input', () => {
    render(<Input placeholder="Disabled" disabled />);
    const input = screen.getByPlaceholderText('Disabled');
    expect(input).toBeDisabled();
  });
});
