import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { Skeleton } from './skeleton';

describe('UI/Skeleton', () => {
  it('renders skeleton with animation classes and custom className', () => {
    const { container } = render(<Skeleton className="h-6 w-24" />);
    const el = container.firstChild as HTMLElement;
    expect(el).toHaveClass('animate-pulse');
    expect(el).toHaveClass('h-6');
    expect(el).toHaveClass('w-24');
  });
});
