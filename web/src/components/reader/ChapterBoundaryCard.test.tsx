import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ChapterBoundaryCard } from './ChapterBoundaryCard';
import { Chapter } from '../../types/api';

describe('ChapterBoundaryCard', () => {
  it('renders previous boundary card when previous chapter exists', () => {
    const targetChapter: Chapter = {
      id: 'ch-1',
      number: 5,
      name: 'Chapter 5',
    };

    render(
      <ChapterBoundaryCard
        type="prev"
        hasChapter={true}
        targetChapter={targetChapter}
      />
    );

    const card = screen.getByTestId('boundary-card-prev');
    expect(card).toBeInTheDocument();
    expect(screen.getByText('Previous chapter (5)')).toBeInTheDocument();
    expect(
      screen.getByText('Swipe or click to read previous chapter')
    ).toBeInTheDocument();
  });

  it('renders previous boundary card at beginning of manga when no previous chapter exists', () => {
    render(
      <ChapterBoundaryCard
        type="prev"
        hasChapter={false}
      />
    );

    const card = screen.getByTestId('boundary-card-prev');
    expect(card).toBeInTheDocument();
    expect(screen.getByText('No previous chapter')).toBeInTheDocument();
    expect(
      screen.getByText('You are at the beginning of the manga')
    ).toBeInTheDocument();
  });

  it('renders next boundary card when next chapter exists', () => {
    const targetChapter: Chapter = {
      id: 'ch-3',
      number: 7,
      title: 'Prologue Part 2',
      name: '',
    };

    render(
      <ChapterBoundaryCard
        type="next"
        hasChapter={true}
        targetChapter={targetChapter}
      />
    );

    const card = screen.getByTestId('boundary-card-next');
    expect(card).toBeInTheDocument();
    expect(screen.getByText('Next chapter (7)')).toBeInTheDocument();
    expect(
      screen.getByText('Swipe or click to continue to next chapter')
    ).toBeInTheDocument();
  });

  it('renders next boundary card with chapter title/name when number is absent', () => {
    const targetChapter: Chapter = {
      id: 'ch-extra',
      number: undefined as unknown as number,
      title: 'Extra Chapter',
      name: '',
    };

    render(
      <ChapterBoundaryCard
        type="next"
        hasChapter={true}
        targetChapter={targetChapter}
      />
    );

    expect(screen.getByText('Next chapter (Extra Chapter)')).toBeInTheDocument();
  });

  it('renders next boundary card when no next chapter exists', () => {
    render(
      <ChapterBoundaryCard
        type="next"
        hasChapter={false}
      />
    );

    const card = screen.getByTestId('boundary-card-next');
    expect(card).toBeInTheDocument();
    expect(screen.getByText('No next chapter')).toBeInTheDocument();
    expect(
      screen.getByText('You have reached the latest chapter')
    ).toBeInTheDocument();
  });

  it('triggers onNavigate callback when clicked', () => {
    const onNavigate = vi.fn();
    render(
      <ChapterBoundaryCard
        type="next"
        hasChapter={true}
        onNavigate={onNavigate}
      />
    );

    fireEvent.click(screen.getByTestId('boundary-card-next'));
    expect(onNavigate).toHaveBeenCalledTimes(1);
  });

  it('applies custom className', () => {
    render(
      <ChapterBoundaryCard
        type="prev"
        hasChapter={false}
        className="custom-boundary-class"
      />
    );

    const card = screen.getByTestId('boundary-card-prev');
    expect(card.className).toContain('custom-boundary-class');
  });
});
