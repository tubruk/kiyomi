import { useMemo } from 'react';
import { useParams, useLocation } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { jobsQueryOptions } from '../../../lib/queryOptions';
import {
  useLibraryManga,
  useMangaDetails,
  useProviderMangaDetails,
  useSources,
  useChapterList,
  useProviderChapterList,
} from '../../../api/hooks';
import { Manga, Source, Chapter, Job } from '../../../types/api';
import { getReadingCtaInfo } from '../../../utils/readingCta';

export interface UseDetailsMangaOptions {
  optimisticPullIds?: Set<string>;
}

export interface UseDetailsMangaReturn {
  params: { mangaId?: string; providerId?: string; remoteId?: string };
  isRemoteRoute: boolean;
  providerIdParam: string;
  remoteIdParam: string;
  targetMangaId: string;
  isInLibrary: boolean;
  libraryEntry?: Manga;
  manga?: Manga;
  remoteManga?: Manga;
  isMangaLoading: boolean;
  sources: Source[];
  activeContentProviderId?: string;
  activeProvider?: Source;
  contentProviderName?: string;
  chapters: Chapter[];
  isChaptersLoading: boolean;
  isChaptersError: boolean;
  mangaJobs: Job[];
  activeJobChapterIds: string[];
  hasActivePullJobs: boolean;
  readingCta: ReturnType<typeof getReadingCtaInfo>;
  isUnavailable: boolean;
  hasZeroChapters: boolean;
}

export function useDetailsManga(options: UseDetailsMangaOptions = {}): UseDetailsMangaReturn {
  const { optimisticPullIds = new Set() } = options;
  const location = useLocation();
  const params = useParams({ strict: false }) as { mangaId?: string; providerId?: string; remoteId?: string };

  const isRemoteRoute =
    location.pathname.startsWith('/explore/') ||
    location.pathname.startsWith('/providers/') ||
    Boolean(params.providerId && params.remoteId);

  const providerIdParam = params.providerId || '';
  const remoteIdParam = params.remoteId || '';

  // 1. Fetch Library List to check if in Library
  const { data: libraryManga = [] } = useLibraryManga();

  // Fetch remote manga details early so libraryEntry can access libraryMangaId
  const { data: remoteDetailsManga, isLoading: isRemoteMangaLoading } = useProviderMangaDetails(
    providerIdParam,
    remoteIdParam,
    { enabled: isRemoteRoute && Boolean(providerIdParam && remoteIdParam) }
  );

  const libraryEntry = isRemoteRoute
    ? remoteDetailsManga?.libraryMangaId
      ? ({ id: remoteDetailsManga.libraryMangaId } as Manga)
      : undefined
    : libraryManga.find((m) => m.id === params.mangaId);

  const isInLibrary = Boolean(libraryEntry);
  const targetMangaId = libraryEntry?.id || params.mangaId || '';

  // Background jobs for the current manga
  const { data: mangaJobs = [] } = useQuery({
    ...jobsQueryOptions({ 'metadata.manga_id': targetMangaId, all: true }),
    enabled: Boolean(targetMangaId && isInLibrary),
  });

  const activeJobChapterIds = useMemo(() => {
    return (mangaJobs as Job[])
      .filter(
        (j) =>
          j.type === 'pull_chapter' &&
          (j.status === 'pending' || j.status === 'running') &&
          j.metadata?.chapter_id
      )
      .map((j) => j.metadata!.chapter_id as string);
  }, [mangaJobs]);

  const hasActivePullJobs = activeJobChapterIds.length > 0 || optimisticPullIds.size > 0;

  // 2. Fetch Manga Details (Remote vs Local)
  const { data: localDetailsManga, isLoading: isLocalMangaLoading } = useMangaDetails(targetMangaId, {
    enabled: !isRemoteRoute && Boolean(targetMangaId),
  });

  const manga = isRemoteRoute ? remoteDetailsManga : localDetailsManga;
  const isMangaLoading = isRemoteRoute ? isRemoteMangaLoading : isLocalMangaLoading;

  // Fetch remote details of library manga from its content provider to get original provider title & availability
  const providerIdForRemote = manga?.contentProviderId || manga?.sourceId || '';
  const remoteIdForRemote = manga?.contentRemoteId || '';
  const { data: remoteManga } = useProviderMangaDetails(providerIdForRemote, remoteIdForRemote, {
    enabled: !isRemoteRoute && Boolean(providerIdForRemote && remoteIdForRemote),
  });

  // Sources for provider info
  const { data: sources = [] } = useSources();

  const activeContentProviderId = isRemoteRoute
    ? providerIdParam
    : manga?.contentProviderId || manga?.sourceId || manga?.meta?.content?.provider_id;

  // 3. Fetch Manga Chapters
  const {
    data: localChaptersData,
    isLoading: isLocalChaptersLoading,
    isError: isLocalChaptersError,
  } = useChapterList(targetMangaId, {
    enabled: (!isRemoteRoute || isInLibrary) && Boolean(targetMangaId),
    hasActivePullJobs,
  });

  const {
    data: remoteChaptersData,
    isLoading: isRemoteChaptersLoading,
    isError: isRemoteChaptersError,
  } = useProviderChapterList(providerIdParam, remoteIdParam, {
    enabled: isRemoteRoute && !isInLibrary && Boolean(providerIdParam && remoteIdParam),
  });

  const chaptersData = isRemoteRoute && !isInLibrary ? remoteChaptersData : localChaptersData;
  const chapters: Chapter[] = chaptersData?.chapters || [];
  const isChaptersLoading = isRemoteRoute && !isInLibrary ? isRemoteChaptersLoading : isLocalChaptersLoading;
  const isChaptersError = isRemoteRoute && !isInLibrary ? isRemoteChaptersError : isLocalChaptersError;

  const activeProvider = sources.find((s) => s.id === activeContentProviderId);
  const contentProviderName = activeProvider
    ? `${activeProvider.name}${
        activeProvider.language || activeProvider.lang
          ? ` (${(activeProvider.language || activeProvider.lang)!.toUpperCase()})`
          : ''
      }`
    : activeContentProviderId || undefined;

  const readingCta = useMemo(() => {
    return getReadingCtaInfo(chapters, manga);
  }, [chapters, manga]);

  const isUnavailable =
    manga?.availability === 'unavailable' ||
    remoteManga?.availability === 'unavailable' ||
    manga?.meta?.availability === 'unavailable';

  const hasZeroChapters =
    isRemoteRoute && !isInLibrary
      ? !isRemoteChaptersLoading && Boolean(remoteChaptersData) && chapters.length === 0
      : !isLocalChaptersLoading && Boolean(localChaptersData) && chapters.length === 0;

  return {
    params,
    isRemoteRoute,
    providerIdParam,
    remoteIdParam,
    targetMangaId,
    isInLibrary,
    libraryEntry,
    manga,
    remoteManga,
    isMangaLoading,
    sources,
    activeContentProviderId,
    activeProvider,
    contentProviderName,
    chapters,
    isChaptersLoading,
    isChaptersError,
    mangaJobs: mangaJobs as Job[],
    activeJobChapterIds,
    hasActivePullJobs,
    readingCta,
    isUnavailable,
    hasZeroChapters,
  };
}
