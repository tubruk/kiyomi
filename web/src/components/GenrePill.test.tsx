import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { GenrePill } from './GenrePill';

describe('GenrePill', () => {
  it('renders genre text correctly', () => {
    render(<GenrePill genre="Action" />);
    expect(screen.getByText('Action')).toBeInTheDocument();
  });
});
