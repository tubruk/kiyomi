export const queryKeys = {
  sources: {
    all: ['sources'] as const,
  },
  library: {
    all: ['library'] as const,
    mangas: () => [...queryKeys.library.all, 'manga'] as const,
    pull: (mangaId: string, providerId?: string) =>
      [...queryKeys.library.all, 'pull', mangaId, providerId ?? ''] as const,
    refresh: (mangaId: string) =>
      [...queryKeys.library.all, 'refresh', mangaId] as const,
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
    providerList: (mangaId: string, providerId: string) =>
      [...queryKeys.chapters.all, mangaId, 'provider', providerId] as const,
    remoteList: (providerId: string, remoteId: string) =>
      ['chapters', 'remote', providerId, remoteId] as const,
    pages: (chapterId: string, mangaId?: string, providerId?: string) =>
      [...queryKeys.chapters.all, 'pages', chapterId, mangaId ?? '', providerId ?? ''] as const,
  },
  plugins: {
    all: ['plugins'] as const,
    logs: (id: string) => ['plugins', 'logs', id] as const,
  },
  collisions: {
    all: ['collisions'] as const,
  },
  system: {
    cache: ['system', 'cache'] as const,
  },
  jobs: {
    all: ['jobs'] as const,
    list: (filter?: any) => ['jobs', 'list', filter] as const,
  },
  info: ['info'] as const,
};
