import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Pagination } from './Pagination';

describe('Pagination', () => {
  it('renders page information when totalPages is provided', () => {
    const handlePageChange = vi.fn();
    render(
      <Pagination
        currentPage={2}
        totalPages={10}
        onPageChange={handlePageChange}
      />
    );

    expect(screen.getByText('Page 2 of 10')).toBeInTheDocument();
  });

  it('renders simple page information when totalPages is not provided', () => {
    const handlePageChange = vi.fn();
    render(
      <Pagination
        currentPage={3}
        hasNextPage={true}
        onPageChange={handlePageChange}
      />
    );

    expect(screen.getByText('Page 3')).toBeInTheDocument();
  });

  it('disables previous button on page 1 and enables next button if has next', async () => {
    const user = userEvent.setup();
    const handlePageChange = vi.fn();
    render(
      <Pagination
        currentPage={1}
        totalPages={5}
        onPageChange={handlePageChange}
      />
    );

    const prevButton = screen.getByRole('button', { name: /previous/i });
    const nextButton = screen.getByRole('button', { name: /next/i });

    expect(prevButton).toBeDisabled();
    expect(nextButton).not.toBeDisabled();

    await user.click(nextButton);
    expect(handlePageChange).toHaveBeenCalledWith(2);
  });

  it('enables previous button on page > 1 and calls onPageChange on click', async () => {
    const user = userEvent.setup();
    const handlePageChange = vi.fn();
    render(
      <Pagination
        currentPage={3}
        totalPages={5}
        onPageChange={handlePageChange}
      />
    );

    const prevButton = screen.getByRole('button', { name: /previous/i });
    expect(prevButton).not.toBeDisabled();

    await user.click(prevButton);
    expect(handlePageChange).toHaveBeenCalledWith(2);
  });

  it('disables next button on the last page', () => {
    const handlePageChange = vi.fn();
    render(
      <Pagination
        currentPage={5}
        totalPages={5}
        onPageChange={handlePageChange}
      />
    );

    const nextButton = screen.getByRole('button', { name: /next/i });
    expect(nextButton).toBeDisabled();
  });

  it('handles hasNextPage boolean correctly when totalPages is undefined', async () => {
    const handlePageChange = vi.fn();
    const { rerender } = render(
      <Pagination
        currentPage={1}
        hasNextPage={false}
        onPageChange={handlePageChange}
      />
    );

    expect(screen.getByRole('button', { name: /next/i })).toBeDisabled();

    rerender(
      <Pagination
        currentPage={1}
        hasNextPage={true}
        onPageChange={handlePageChange}
      />
    );

    expect(screen.getByRole('button', { name: /next/i })).not.toBeDisabled();
  });
});
