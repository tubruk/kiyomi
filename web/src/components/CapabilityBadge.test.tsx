import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CapabilityBadge } from './CapabilityBadge';

describe('CapabilityBadge', () => {
  it('renders content capability badge', () => {
    render(<CapabilityBadge capability="content" />);
    expect(screen.getByText('Content')).toBeInTheDocument();
  });

  it('renders metadata capability badge', () => {
    render(<CapabilityBadge capability="metadata" />);
    expect(screen.getByText('Metadata')).toBeInTheDocument();
  });

  it('renders tracking capability badge', () => {
    render(<CapabilityBadge capability="tracking" />);
    expect(screen.getByText('Tracking')).toBeInTheDocument();
  });

  it('renders active state with check icon and emerald class', () => {
    const { container } = render(<CapabilityBadge capability="content" active={true} />);
    const badge = screen.getByText('Content').closest('.group\\/badge, span');
    expect(badge).toHaveClass('border-emerald-500/40');
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('renders unavailable state with alert icon and title', () => {
    const { container } = render(<CapabilityBadge capability="content" unavailable={true} />);
    const badge = screen.getByText('Content').closest('.group\\/badge, span');
    expect(badge).toHaveClass('border-amber-500/40');
    expect(badge).toHaveAttribute('title', 'Content not available');
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<CapabilityBadge capability="metadata" className="my-custom-class" />);
    const badge = screen.getByText('Metadata').closest('.group\\/badge, span');
    expect(badge).toHaveClass('my-custom-class');
  });

  it('returns null for invalid capability', () => {
    // @ts-expect-error test invalid prop
    const { container } = render(<CapabilityBadge capability="nonexistent" />);
    expect(container.firstChild).toBeNull();
  });
});
