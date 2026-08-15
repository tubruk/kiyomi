import React from 'react';
import { Manga } from '../types/api';
import { MangaCard } from './MangaCard';
import { SkeletonCard } from './SkeletonCard';

interface MangaGridProps {
  mangas?: Manga[];
  isLoading?: boolean;
  isError?: boolean;
  errorMessage?: string;
}

export const MangaGrid: React.FC<MangaGridProps> = ({
  mangas = [],
  isLoading = false,
  isError = false,
  errorMessage = 'Failed to load titles',
}) => {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
        {Array.from({ length: 12 }).map((_, i) => (
          <SkeletonCard key={i} />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-8 text-center text-sm text-destructive">
        <p className="font-semibold">{errorMessage}</p>
      </div>
    );
  }

  if (!mangas || mangas.length === 0) {
    return (
      <div className="rounded-xl border border-border bg-card p-12 text-center text-sm text-muted-foreground">
        No manga found.
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
      {mangas.map((manga) => (
        <MangaCard key={manga.id} manga={manga} />
      ))}
    </div>
  );
};
