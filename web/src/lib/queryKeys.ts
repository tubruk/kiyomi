export const queryKeys = {
  sources: {
    all: ['sources'] as const,
  },
  library: {
    all: ['library'] as const,
    mangas: () => [...queryKeys.library.all, 'manga'] as const,
    providers: (mangaId: string) => [...queryKeys.library.all, 'providers', mangaId] as const,
    pull: (mangaId: string, providerId?: string) =>
      [...queryKeys.library.all, 'pull', mangaId, ...(providerId ? [providerId] : [])] as const,
    refresh: (mangaId: string) =>
      [...queryKeys.library.all, 'refresh', mangaId] as const,
  },
  manga: {
    all: ['manga'] as const,
    details: (id: string) => [...queryKeys.manga.all, 'detail', id] as const,
    providerDetails: (providerId: string, remoteId: string) =>
      [...queryKeys.manga.all, 'provider', providerId, remoteId] as const,
    // Per-concern keys (Stage 3 of manga-metadata-separation).
    // detail uses the bare manga id as a prefix so the three concern keys
    // (metadata / user_state / bindings) are nested under it for granular
    // invalidation by useUpdateUserStateMutation etc.
    detail: (id: string) => [...queryKeys.manga.all, id] as const,
    metadata: (id: string) => [...queryKeys.manga.all, id, 'metadata'] as const,
    userState: (id: string) => [...queryKeys.manga.all, id, 'user_state'] as const,
    bindings: (id: string) => [...queryKeys.manga.all, id, 'bindings'] as const,
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
      [...queryKeys.chapters.all, 'remote', providerId, remoteId] as const,
    pages: (chapterId: string, mangaId?: string, providerId?: string) =>
      [...queryKeys.chapters.all, 'pages', chapterId, ...(mangaId ? [mangaId] : []), ...(providerId ? [providerId] : [])] as const,
  },
  plugins: {
    all: ['plugins'] as const,
    logs: (id: string) => [...queryKeys.plugins.all, 'logs', id] as const,
  },
  collisions: {
    all: ['collisions'] as const,
  },
  system: {
    all: ['system'] as const,
    cache: ['system', 'cache'] as const,
  },
  jobs: {
    all: ['jobs'] as const,
    list: (filter?: any) =>
      filter && Object.keys(filter).length > 0
        ? ([...queryKeys.jobs.all, 'list', filter] as const)
        : ([...queryKeys.jobs.all, 'list'] as const),
    children: (parentId: string) => [...queryKeys.jobs.all, 'children', parentId] as const,
  },
  info: {
    all: ['info'] as const,
  },
};
