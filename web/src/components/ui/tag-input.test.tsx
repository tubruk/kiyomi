import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TagInput } from './tag-input';

describe('TagInput', () => {
  it('renders badges for existing tags and placeholder when empty', () => {
    const { rerender } = render(<TagInput value={['Action', 'Fantasy']} placeholder="Add tag..." />);

    expect(screen.getByText('Action')).toBeInTheDocument();
    expect(screen.getByText('Fantasy')).toBeInTheDocument();

    rerender(<TagInput value={[]} placeholder="Add tag..." />);
    expect(screen.getByPlaceholderText('Add tag...')).toBeInTheDocument();
  });

  it('adds new tag on Enter key and ignores duplicate case-insensitively', () => {
    const onChange = vi.fn();
    render(<TagInput value={['Action']} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'Comedy' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onChange).toHaveBeenCalledWith(['Action', 'Comedy']);

    // Attempt duplicate
    onChange.mockClear();
    fireEvent.change(input, { target: { value: 'action' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onChange).not.toHaveBeenCalled();
  });

  it('adds tags separated by comma on input change or comma keydown', () => {
    const onChange = vi.fn();
    render(<TagInput value={['Action']} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'Drama, Sci-Fi,' } });

    expect(onChange).toHaveBeenCalledWith(['Action', 'Drama', 'Sci-Fi']);
  });

  it('removes tag when clicking remove button', () => {
    const onChange = vi.fn();
    render(<TagInput value={['Action', 'Fantasy']} onChange={onChange} />);

    const removeBtn = screen.getByTitle('Remove Action');
    fireEvent.click(removeBtn);

    expect(onChange).toHaveBeenCalledWith(['Fantasy']);
  });

  it('removes last tag on Backspace when input is empty', () => {
    const onChange = vi.fn();
    render(<TagInput value={['Action', 'Fantasy']} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    fireEvent.keyDown(input, { key: 'Backspace' });

    expect(onChange).toHaveBeenCalledWith(['Action']);
  });
});
