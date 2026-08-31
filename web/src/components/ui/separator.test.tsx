import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { Separator } from './separator';

describe('UI/Separator', () => {
  it('renders horizontal separator by default', () => {
    const { container } = render(<Separator />);
    const sep = container.firstChild as HTMLElement;
    expect(sep).toHaveAttribute('data-slot', 'separator');
    expect(sep).toHaveAttribute('data-orientation', 'horizontal');
    expect(sep).toHaveClass('bg-border');
  });

  it('renders vertical separator and applies custom class', () => {
    const { container } = render(<Separator orientation="vertical" className="my-custom-sep" />);
    const sep = container.firstChild as HTMLElement;
    expect(sep).toHaveAttribute('data-slot', 'separator');
    expect(sep).toHaveAttribute('data-orientation', 'vertical');
    expect(sep).toHaveClass('my-custom-sep');
  });
});
