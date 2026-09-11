import { useMemo, useState } from 'react';
import { useLibraryManga } from '../../../api/hooks';
import { Manga } from '../../../types/api';

export interface UseMergeFlowOptions {
  keepMangaId: string;
}

export interface UseMergeFlowReturn {
  availableSources: Manga[];
  sourceMangaId: string | null;
  setSourceMangaId: (id: string | null) => void;
  sourceManga: Manga | undefined;
}

export const useMergeFlow = ({ keepMangaId }: UseMergeFlowOptions): UseMergeFlowReturn => {
  const [sourceMangaId, setSourceMangaId] = useState<string | null>(null);
  const { data: libraryMangas } = useLibraryManga();

  const availableSources = useMemo<Manga[]>(
    () => (libraryMangas ?? []).filter((m: Manga) => m.id !== keepMangaId),
    [libraryMangas, keepMangaId]
  );

  const sourceManga = useMemo<Manga | undefined>(
    () => availableSources.find((m: Manga) => m.id === sourceMangaId),
    [availableSources, sourceMangaId]
  );

  return {
    availableSources,
    sourceMangaId,
    setSourceMangaId,
    sourceManga,
  };
};