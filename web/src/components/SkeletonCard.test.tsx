import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { SkeletonCard } from './SkeletonCard';

describe('SkeletonCard', () => {
  it('renders without crashing and contains skeleton placeholders', () => {
    const { container } = render(<SkeletonCard />);
    const skeletons = container.querySelectorAll('.animate-pulse, [data-slot=skeleton]');
    expect(skeletons.length).toBeGreaterThan(0);
    expect(container.firstChild).toHaveClass('rounded-lg');
  });
});
