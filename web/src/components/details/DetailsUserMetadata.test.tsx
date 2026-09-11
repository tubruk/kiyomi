import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DetailsUserMetadata } from './DetailsUserMetadata';
import { Manga } from '../../types/api';

const mockManga: Manga = {
  id: 'manga-1',
  title: 'Chainsaw Man',
  userStatus: 'reading',
  userRating: 8,
  userFavorite: true,
  userNotes: 'Great arc!',
  metadata: {
    title: 'Chainsaw Man',
    aliases: [],
    description: '',
    authors: [],
    artists: [],
    tags: [],
    collections: [],
    publishers: [],
  },
  user_state: {
    status: 'reading',
    rating: 8,
    favorite: true,
    notes: 'Great arc!',
  },
  bindings: { providers: [] },
};

describe('DetailsUserMetadata', () => {
  it('renders reading status, rating score, favorite state, and notes', () => {
    render(<DetailsUserMetadata manga={mockManga} onUpdate={vi.fn()} />);

    expect(screen.getByText('Reading')).toBeInTheDocument();
    expect(screen.getByText('8/10')).toBeInTheDocument();
    expect(screen.getByLabelText('Remove from Favorites')).toBeInTheDocument();
    expect(screen.getByText('Great arc!')).toBeInTheDocument();
  });

  it('triggers onUpdate when rating star is clicked', () => {
    const onUpdate = vi.fn();
    render(<DetailsUserMetadata manga={mockManga} onUpdate={onUpdate} />);

    const starBtn = screen.getByLabelText('Rate 10/10 (5 Stars)');
    fireEvent.click(starBtn);

    expect(onUpdate).toHaveBeenCalledWith({ user_rating: 10 });
  });

  it('triggers onUpdate when favorite toggle is clicked', () => {
    const onUpdate = vi.fn();
    render(<DetailsUserMetadata manga={mockManga} onUpdate={onUpdate} />);

    const favBtn = screen.getByLabelText('Remove from Favorites');
    fireEvent.click(favBtn);

    expect(onUpdate).toHaveBeenCalledWith({ user_favorite: false });
  });

  it('allows editing and saving user notes', () => {
    const onUpdate = vi.fn();
    render(<DetailsUserMetadata manga={mockManga} onUpdate={onUpdate} />);

    const editBtn = screen.getByText('Edit');
    fireEvent.click(editBtn);

    const textarea = screen.getByPlaceholderText('Add private personal notes...');
    fireEvent.change(textarea, { target: { value: 'Updated note' } });

    const saveBtn = screen.getByRole('button', { name: 'Save' });
    fireEvent.click(saveBtn);

    expect(onUpdate).toHaveBeenCalledWith({ user_notes: 'Updated note' });
  });
});
