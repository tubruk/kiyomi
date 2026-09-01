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
  publishers: ['Shueisha', 'VIZ Media'],
  releaseYear: 2018,
  startDate: '2018-03-05',
  endDate: '2024-09-30',
  country: 'JP',
  externalLinks: [
    { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/101517' },
  ],
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

  it('renders separate author and artist sections when author and artist differ', () => {
    const multiCreatorManga: Manga = {
      ...mockManga,
      authors: ['ONE'],
      artists: ['Yusuke Murata'],
    };

    render(<DetailsHeroCard manga={multiCreatorManga} />);

    expect(screen.getByText('Author:')).toBeInTheDocument();
    expect(screen.getByText('ONE')).toBeInTheDocument();
    expect(screen.getByText('Artist:')).toBeInTheDocument();
    expect(screen.getByText('Yusuke Murata')).toBeInTheDocument();
  });

  it('toggles collapsible detailed metadata accordion and renders multiple publishers, dates, and country', () => {
    render(<DetailsHeroCard manga={mockManga} />);

    expect(screen.queryByText('Shueisha, VIZ Media')).not.toBeInTheDocument();

    const toggleBtn = screen.getByRole('button', { name: /show detailed metadata/i });
    fireEvent.click(toggleBtn);

    // Plural label when multiple publishers
    expect(screen.getByText('Publishers')).toBeInTheDocument();
    expect(screen.getByText('Shueisha, VIZ Media')).toBeInTheDocument();
    expect(screen.getByText('Release Year')).toBeInTheDocument();
    expect(screen.getByText('2018')).toBeInTheDocument();
    expect(screen.getByText('Start Date')).toBeInTheDocument();
    expect(screen.getByText('2018-03-05')).toBeInTheDocument();
    expect(screen.getByText('End Date')).toBeInTheDocument();
    expect(screen.getByText('2024-09-30')).toBeInTheDocument();
    expect(screen.getByText('Country')).toBeInTheDocument();
    expect(screen.getByText('JP')).toBeInTheDocument();
    expect(screen.getByText('AniList')).toBeInTheDocument();

    const hideBtn = screen.getByRole('button', { name: /hide details/i });
    fireEvent.click(hideBtn);

    expect(screen.queryByText('Shueisha, VIZ Media')).not.toBeInTheDocument();
  });

  it('renders singular "Publisher" label when manga has only one publisher', () => {
    const singlePubManga: Manga = {
      ...mockManga,
      publishers: ['Kodansha'],
    };

    render(<DetailsHeroCard manga={singlePubManga} />);

    const toggleBtn = screen.getByRole('button', { name: /show detailed metadata/i });
    fireEvent.click(toggleBtn);

    expect(screen.getByText('Publisher')).toBeInTheDocument();
    expect(screen.getByText('Kodansha')).toBeInTheDocument();
  });

  it('renders metadata correctly when properties are inside meta object with snake_case keys', () => {
    const metaManga: Manga = {
      id: 'manga-meta',
      title: 'Solo Leveling',
      meta: {
        publishers: ['D&C Media', 'Yen Press'],
        start_date: '2018-03-04',
        end_date: '2021-12-29',
        country: 'KR',
        release_year: 2018,
      },
    };

    render(<DetailsHeroCard manga={metaManga} />);

    const toggleBtn = screen.getByRole('button', { name: /show detailed metadata/i });
    fireEvent.click(toggleBtn);

    expect(screen.getByText('Publishers')).toBeInTheDocument();
    expect(screen.getByText('D&C Media, Yen Press')).toBeInTheDocument();
    expect(screen.getByText('Start Date')).toBeInTheDocument();
    expect(screen.getByText('2018-03-04')).toBeInTheDocument();
    expect(screen.getByText('End Date')).toBeInTheDocument();
    expect(screen.getByText('2021-12-29')).toBeInTheDocument();
    expect(screen.getByText('Country')).toBeInTheDocument();
    expect(screen.getByText('KR')).toBeInTheDocument();
  });

  it('renders loading skeleton when isMangaLoading is true', () => {
    render(<DetailsHeroCard isMangaLoading={true} />);

    expect(screen.getByText('Loading Manga...')).toBeInTheDocument();
  });
});
