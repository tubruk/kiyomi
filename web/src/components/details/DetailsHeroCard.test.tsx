import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DetailsHeroCard } from './DetailsHeroCard';
import { Manga } from '../../types/api';

const mockManga: Manga = {
  id: 'manga-1',
  title: 'Jujutsu Kaisen',
  aliases: ['JJK', 'Sorcery Fight'],
  authors: ['Gege Akutami'],
  artists: ['Gege Akutami'],
  genres: ['Action', 'Supernatural'],
  description: 'A boy fights curses.',
  publisher: 'Shueisha',
  releaseYear: 2018,
  country: 'JP',
};

describe('DetailsHeroCard', () => {
  it('renders title, aliases, merged author/artist, genres, and description', () => {
    render(<DetailsHeroCard manga={mockManga} contentProviderName="MangaDex" />);

    expect(screen.getByText('Jujutsu Kaisen')).toBeInTheDocument();
    expect(screen.getByText('JJK, Sorcery Fight')).toBeInTheDocument();
    expect(screen.getByText('Author / Artist:')).toBeInTheDocument();
    expect(screen.getByText('Gege Akutami')).toBeInTheDocument();
    expect(screen.getByText('Action')).toBeInTheDocument();
    expect(screen.getByText('Supernatural')).toBeInTheDocument();
    expect(screen.getByText('A boy fights curses.')).toBeInTheDocument();
    expect(screen.getByText(/MangaDex/)).toBeInTheDocument();
  });

  it('toggles collapsible detailed metadata accordion', () => {
    render(<DetailsHeroCard manga={mockManga} />);

    expect(screen.queryByText('Shueisha')).not.toBeInTheDocument();

    const toggleBtn = screen.getByRole('button', { name: /show detailed metadata/i });
    fireEvent.click(toggleBtn);

    expect(screen.getByText('Shueisha')).toBeInTheDocument();
    expect(screen.getByText('2018')).toBeInTheDocument();
    expect(screen.getByText('JP')).toBeInTheDocument();

    const hideBtn = screen.getByRole('button', { name: /hide details/i });
    fireEvent.click(hideBtn);

    expect(screen.queryByText('Shueisha')).not.toBeInTheDocument();
  });
});
