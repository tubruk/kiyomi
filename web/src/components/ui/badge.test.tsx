import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge } from './badge';

describe('UI/Badge', () => {
  it('renders default badge', () => {
    render(<Badge>Default Badge</Badge>);
    const badge = screen.getByText('Default Badge');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('bg-primary');
  });

  it('renders secondary badge', () => {
    render(<Badge variant="secondary">Secondary Badge</Badge>);
    const badge = screen.getByText('Secondary Badge');
    expect(badge).toHaveClass('bg-secondary');
  });

  it('renders destructive badge', () => {
    render(<Badge variant="destructive">Destructive Badge</Badge>);
    const badge = screen.getByText('Destructive Badge');
    expect(badge).toHaveClass('text-destructive');
  });

  it('renders outline badge', () => {
    render(<Badge variant="outline">Outline Badge</Badge>);
    const badge = screen.getByText('Outline Badge');
    expect(badge).toHaveClass('border-border');
  });

  it('renders ghost and link variants', () => {
    const { rerender } = render(<Badge variant="ghost">Ghost</Badge>);
    expect(screen.getByText('Ghost')).toHaveClass('hover:bg-muted');

    rerender(<Badge variant="link">Link</Badge>);
    expect(screen.getByText('Link')).toHaveClass('underline-offset-4');
  });
});
