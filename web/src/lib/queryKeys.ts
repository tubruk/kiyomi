export const queryKeys = {
  sources: {
    all: ['sources'] as const,
  },
  library: {
    all: ['library'] as const,
    mangas: () => [...queryKeys.library.all, 'manga'] as const,
  },
  manga: {
    all: ['manga'] as const,
    details: (id: string) => [...queryKeys.manga.all, 'detail', id] as const,
    providerDetails: (providerId: string, remoteId: string) =>
      [...queryKeys.manga.all, 'provider', providerId, remoteId] as const,
  },
  explore: {
    all: ['explore'] as const,
    catalog: (providerId: string, mode: string, query: string, page: number) =>
      [...queryKeys.explore.all, providerId, mode, query, page] as const,
  },
  chapters: {
    all: ['chapters'] as const,
    list: (mangaId: string) =>
      [...queryKeys.chapters.all, mangaId] as const,
    providerList: (providerId: string, remoteId: string) =>
      ['chapters', 'provider', providerId, remoteId] as const,
    pages: (chapterId: string, mangaId?: string, providerId?: string) =>
      [...queryKeys.chapters.all, 'pages', chapterId, mangaId ?? '', providerId ?? ''] as const,
  },
};

export const chapterKeys = queryKeys.chapters;

