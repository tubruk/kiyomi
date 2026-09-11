import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useSearch, useNavigate } from '@tanstack/react-router';
import { Loader2 } from 'lucide-react';

import {
  useChapterPages,
  useMangaDetails,
  useChapterList,
  useProviderMangaDetails,
  useProviderChapterList,
  useLibraryManga,
  useUpdateChapterProgressMutation,
  useUpdateLibraryMangaMutation,
} from '../api/hooks';
import { useToast } from '../context/ToastContext';
import { ReaderTopBar } from '../components/ReaderTopBar';
import { ReaderFooter } from '../components/ReaderFooter';
import { ReaderHint } from '../components/ReaderHint';
import { CompletionPromptDialog } from '../components/CompletionPromptDialog';
import { ReaderPagedView, ReaderContinuousView } from '../components/reader';
import { useReadingHint } from '../hooks/useReadingHint';
import { useReaderFitMode } from '../hooks/useReaderFitMode';
import { useReaderKeyboard } from '../hooks/useReaderKeyboard';
import { useReaderGesture } from '../hooks/useReaderGesture';
import { useReaderProgressTracker } from '../hooks/useReaderProgressTracker';
import { useReaderScrollTracker } from '../hooks/useReaderScrollTracker';
import { getPageImageUrl, formatChapterTitleWithPage } from '../lib/utils';

interface ReaderSearch {
  mangaId?: string;
  page?: number | 'last' | string;
}

export const ReaderPage: React.FC = () => {
  const params = useParams({ strict: false }) as Record<string, string | undefined>;
  const chapterId = params.chapterId || '';
  const searchParams = useSearch({ strict: false }) as ReaderSearch;
  const navigate = useNavigate();
  const { showToast } = useToast();

  const mangaId = params.mangaId || searchParams.mangaId;
  const rawProviderId = params.providerId;
  const providerId = rawProviderId && rawProviderId !== 'chapters' ? rawProviderId : undefined;
  const remoteId = params.remoteId;

  const [showOverlays, setShowOverlays] = useState(true);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const pageRefs = useRef<(HTMLDivElement | null)[]>([]);
  const prevChapterIdRef = useRef<string | null>(null);

  // Fetch library list to resolve effective local manga ID if needed
  const { data: libraryManga = [] } = useLibraryManga();
  const libraryEntry = mangaId
    ? libraryManga.find((m) => m.id === mangaId)
    : providerId && remoteId
    ? libraryManga.find(
        (m) =>
          (m.contentProviderId === providerId ||
            m.sourceId === providerId ||
            m.meta?.content?.provider_id === providerId) &&
          (m.contentRemoteId === remoteId ||
            m.url === remoteId ||
            m.id === remoteId ||
            m.meta?.content?.provider_manga_id === remoteId)
      )
    : undefined;

  const effectiveMangaId = mangaId || libraryEntry?.id;

  // Fetch manga details (local vs remote)
  const { data: localManga } = useMangaDetails(effectiveMangaId || '', {
    enabled: Boolean(effectiveMangaId),
  });
  const { data: remoteManga } = useProviderMangaDetails(providerId || '', remoteId || '', {
    enabled: Boolean(!effectiveMangaId && providerId && remoteId),
  });
  const manga = effectiveMangaId ? localManga : remoteManga;
  const rawChapterProviderId =
    manga?.contentProviderId || manga?.meta?.content?.provider_id || providerId || '';
  const chapterProviderId = rawChapterProviderId !== 'chapters' ? rawChapterProviderId : '';

  // Fetch chapter pages
  const {
    data: pagesData,
    isLoading: isPagesLoading,
    isError: isPagesError,
  } = useChapterPages(effectiveMangaId || '', chapterId, {
    enabled: Boolean(chapterId && effectiveMangaId),
  });

  const pages = pagesData?.pages || [];

  // Fetch chapters list to enable prev/next navigation
  const { data: localChaptersData } = useChapterList(effectiveMangaId || '', {
    enabled: Boolean(effectiveMangaId),
  });
  const { data: remoteChaptersData } = useProviderChapterList(providerId || '', remoteId || '', {
    enabled: Boolean(!effectiveMangaId && providerId && remoteId),
  });

  const chaptersData = effectiveMangaId ? localChaptersData : remoteChaptersData;
  const chapters = chaptersData?.chapters || [];
  const currentChapterIndex = chapters.findIndex((c) => c.id === chapterId);
  const currentChapter = chapters[currentChapterIndex];

  const hasPrevChapter = currentChapterIndex > 0;
  const hasNextChapter = currentChapterIndex >= 0 && currentChapterIndex < chapters.length - 1;
  const prevChapter = hasPrevChapter ? chapters[currentChapterIndex - 1] : undefined;
  const nextChapter = hasNextChapter ? chapters[currentChapterIndex + 1] : undefined;

  const rawReadingMode = (
    manga?.content?.reading_mode ||
    manga?.meta?.content?.reading_mode ||
    manga?.readingMode ||
    manga?.reading_mode ||
    manga?.readingDirection ||
    manga?.meta?.reading_direction ||
    ''
  )
    .trim()
    .toLowerCase();

  const readingMode = ['rtl', 'ltr', 'vertical', 'longstrip'].includes(rawReadingMode)
    ? rawReadingMode
    : 'rtl';

  const isPaged = readingMode === 'rtl' || readingMode === 'ltr';

  // Reading mode hint & fit mode
  const { hint: readingHint, dismissHint: dismissReadingHint } = useReadingHint(
    effectiveMangaId,
    readingMode
  );
  const { fitMode, setFitMode } = useReaderFitMode();

  // Mutations
  const updateProgressMutation = useUpdateChapterProgressMutation();
  const updateLibraryMangaMutation = useUpdateLibraryMangaMutation();

  const isChapterRead = Boolean(currentChapter?.meta?.is_read ?? (currentChapter as any)?.is_read);
  const chapterLastReadPage =
    currentChapter?.meta?.last_read_page ?? (currentChapter as any)?.last_read_page ?? 0;

  // Scroll tracking (continuous mode)
  const { scrollToPage, setReportedPage } = useReaderScrollTracker({
    isPaged,
    pagesCount: pages.length,
    pageRefs,
    onPageVisible: (page) => {
      setCurrentPage(page);
    },
  });

  // Progress tracking
  const {
    currentPage,
    setCurrentPage,
    showCompletionDialog,
    setShowCompletionDialog,
    markChapterRead,
  } = useReaderProgressTracker({
    effectiveMangaId,
    chapterId,
    chapterProviderId,
    pagesCount: pages.length,
    isPaged,
    isChapterRead,
    chapterLastReadPage,
    searchPage: searchParams.page,
    hasNextChapter,
    userStatus: manga?.user_status || manga?.meta?.user_status,
    onScrollToPage: (page) => {
      scrollToPage(page, false);
      setReportedPage(page);
    },
    onUpdateProgress: (params) => {
      updateProgressMutation.mutate({
        mangaId: params.mangaId,
        chapterId: params.chapterId,
        isRead: params.progress?.is_read,
        lastReadPage: params.progress?.last_read_page,
      });
    },
  });

  // Toast notification on chapter transition
  useEffect(() => {
    if (chapterId && currentChapter && prevChapterIdRef.current !== chapterId) {
      const chNum = currentChapter.meta?.number ?? (currentChapter as any).number;
      const chTitle = currentChapter.meta?.title ?? (currentChapter as any).title;
      showToast(
        chTitle ? `Chapter ${chNum}: ${chTitle}` : `Chapter ${chNum}`,
        'info',
        undefined,
        'subtle'
      );
      prevChapterIdRef.current = chapterId;
    }
  }, [chapterId, currentChapter, showToast]);

  // Update document title with chapter and page info
  useEffect(() => {
    if (manga?.title || currentChapter) {
      const rawChTitle = currentChapter?.title || currentChapter?.name || 'Chapter View';
      const chNum = currentChapter?.number ?? currentChapter?.meta?.number;
      const formattedCh = formatChapterTitleWithPage(rawChTitle, currentPage, pages.length, chNum);
      const mTitle = manga?.title || 'Kiyomi';
      document.title = `${mTitle} - ${formattedCh}`;
    }
  }, [manga?.title, currentChapter, currentPage, pages.length]);

  const handleSelectChapter = useCallback(
    (targetChapterId: string, targetPage?: number | 'last') => {
      const search = targetPage !== undefined ? { page: targetPage } : undefined;
      if (mangaId) {
        navigate({
          to: '/manga/$mangaId/chapter/$chapterId',
          params: { mangaId, chapterId: targetChapterId },
          search,
        });
      } else if (providerId && remoteId) {
        navigate({
          to: '/providers/$providerId/manga/$remoteId/chapter/$chapterId',
          params: { providerId, remoteId, chapterId: targetChapterId },
          search,
        });
      } else {
        navigate({
          to: '/reader/$chapterId',
          params: { chapterId: targetChapterId },
          search,
        });
      }
    },
    [mangaId, providerId, remoteId, navigate]
  );

  const handlePrevChapter = useCallback(() => {
    if (hasPrevChapter && prevChapter) {
      handleSelectChapter(prevChapter.id, 'last');
    }
  }, [hasPrevChapter, prevChapter, handleSelectChapter]);

  const handleNextChapter = useCallback(() => {
    markChapterRead(pages.length);
    if (hasNextChapter && nextChapter) {
      handleSelectChapter(nextChapter.id);
    }
  }, [markChapterRead, pages.length, hasNextChapter, nextChapter, handleSelectChapter]);

  const getContainerWidth = useCallback(() => {
    return (
      containerRef.current?.clientWidth || (typeof window !== 'undefined' ? window.innerWidth : 800)
    );
  }, []);

  const goToNextPageRef = useRef<() => void>(() => {});
  const goToPrevPageRef = useRef<() => void>(() => {});

  // Gesture Tracker
  const {
    dragOffset,
    isDragging,
    isAnimating,
    didDragRef,
    setDragOffset,
    setIsAnimating,
    handlers: gestureHandlers,
  } = useReaderGesture({
    readingMode,
    isPaged,
    currentPage,
    totalPages: pages.length,
    hasNextChapter,
    hasPrevChapter,
    containerWidth: getContainerWidth,
    onNextPage: () => goToNextPageRef.current(),
    onPrevPage: () => goToPrevPageRef.current(),
  });

  // Directional Paged Navigation Handlers
  const goToNextPage = useCallback(() => {
    if (isAnimating) return;
    const slideDistance = getContainerWidth();

    if (currentPage < pages.length) {
      setIsAnimating(true);
      const targetOffset = readingMode === 'rtl' ? slideDistance : -slideDistance;
      setDragOffset(targetOffset);
      setTimeout(() => {
        setDragOffset(0);
        setCurrentPage((prev) => prev + 1);
        setIsAnimating(false);
        setTimeout(() => {
          if (didDragRef.current) didDragRef.current = false;
        }, 50);
      }, 220);
    } else {
      markChapterRead(pages.length);
      if (hasNextChapter) {
        setIsAnimating(true);
        const targetOffset = readingMode === 'rtl' ? slideDistance : -slideDistance;
        setDragOffset(targetOffset);
        setTimeout(() => {
          setDragOffset(0);
          setIsAnimating(false);
          handleNextChapter();
          setTimeout(() => {
            if (didDragRef.current) didDragRef.current = false;
          }, 50);
        }, 220);
      }
    }
  }, [
    isAnimating,
    getContainerWidth,
    currentPage,
    pages.length,
    readingMode,
    markChapterRead,
    hasNextChapter,
    handleNextChapter,
    setCurrentPage,
    setDragOffset,
    setIsAnimating,
    didDragRef,
  ]);

  const goToPrevPage = useCallback(() => {
    if (isAnimating) return;
    const slideDistance = getContainerWidth();

    if (currentPage > 1) {
      setIsAnimating(true);
      const targetOffset = readingMode === 'rtl' ? -slideDistance : slideDistance;
      setDragOffset(targetOffset);
      setTimeout(() => {
        setDragOffset(0);
        setCurrentPage((prev) => prev - 1);
        setIsAnimating(false);
        setTimeout(() => {
          if (didDragRef.current) didDragRef.current = false;
        }, 50);
      }, 220);
    } else if (hasPrevChapter) {
      setIsAnimating(true);
      const targetOffset = readingMode === 'rtl' ? -slideDistance : slideDistance;
      setDragOffset(targetOffset);
      setTimeout(() => {
        setDragOffset(0);
        setIsAnimating(false);
        handlePrevChapter();
        setTimeout(() => {
          if (didDragRef.current) didDragRef.current = false;
        }, 50);
      }, 220);
    }
  }, [
    isAnimating,
    getContainerWidth,
    currentPage,
    readingMode,
    hasPrevChapter,
    handlePrevChapter,
    setCurrentPage,
    setDragOffset,
    setIsAnimating,
    didDragRef,
  ]);

  goToNextPageRef.current = goToNextPage;
  goToPrevPageRef.current = goToPrevPage;

  // Keyboard navigation
  useReaderKeyboard({
    readingMode,
    isPaged,
    onNextPage: goToNextPage,
    onPrevPage: goToPrevPage,
    onFirstPage: () => handlePageChange(1),
    onLastPage: () => handlePageChange(pages.length),
    onToggleOverlays: () => setShowOverlays((prev) => !prev),
  });

  // Preload adjacent images in Paged mode
  useEffect(() => {
    if (!isPaged || pages.length === 0) return;
    const nextIdx = currentPage;
    if (nextIdx < pages.length) {
      const nextImg = new Image();
      const p = pages[nextIdx];
      nextImg.src = getPageImageUrl(p, effectiveMangaId, chapterId, undefined, manga?.url);
    }
    const nextNextIdx = currentPage + 1;
    if (nextNextIdx < pages.length) {
      const nextNextImg = new Image();
      const p = pages[nextNextIdx];
      nextNextImg.src = getPageImageUrl(p, effectiveMangaId, chapterId, undefined, manga?.url);
    }
    const prevIdx = currentPage - 2;
    if (prevIdx >= 0) {
      const prevImg = new Image();
      const p = pages[prevIdx];
      prevImg.src = getPageImageUrl(p, effectiveMangaId, chapterId, undefined, manga?.url);
    }
  }, [currentPage, isPaged, pages, effectiveMangaId, chapterId, manga?.url]);

  const handlePageChange = (targetPage: number) => {
    if (targetPage !== currentPage) {
      setDragOffset(0);
      setIsAnimating(false);
      setCurrentPage(targetPage);
    }
    if (!isPaged) {
      scrollToPage(targetPage, true);
    }
    navigate({ search: { ...searchParams, page: targetPage } as any, replace: true });
  };

  const handleScrollTop = () => {
    if (isPaged) {
      if (currentPage !== 1) {
        setDragOffset(0);
        setIsAnimating(false);
        setCurrentPage(1);
      }
    } else {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  };

  const handleReadingModeChange = (mode: string) => {
    dismissReadingHint();
    if (effectiveMangaId) {
      updateLibraryMangaMutation.mutate(
        {
          mangaId: effectiveMangaId,
          fields: {
            reading_mode: mode,
            readingDirection: mode,
            content: { reading_mode: mode },
          },
        },
        {
          onError: (err: any) => {
            showToast(err?.message || 'Failed to update reading mode', 'error');
          },
        }
      );
    }
  };

  return (
    <div className="min-h-screen transition-colors duration-200 bg-black pt-14">
      <div
        className={`fixed top-0 left-0 right-0 z-50 transition-transform duration-300 ease-in-out ${
          showOverlays ? 'translate-y-0' : '-translate-y-full'
        }`}
      >
        <ReaderTopBar
          mangaTitle={manga?.title}
          chapterTitle={
            chapters[currentChapterIndex]?.title ||
            chapters[currentChapterIndex]?.name ||
            'Chapter View'
          }
          currentPage={currentPage}
          totalPages={pages.length}
          chapterNumber={
            chapters[currentChapterIndex]?.number ?? chapters[currentChapterIndex]?.meta?.number
          }
          mangaId={mangaId}
          providerId={providerId}
          remoteId={remoteId}
          chapters={chapters}
          currentChapterId={chapterId}
          readingMode={readingMode}
          hasPrevChapter={hasPrevChapter}
          hasNextChapter={hasNextChapter}
          onPrevChapter={handlePrevChapter}
          onNextChapter={handleNextChapter}
          onSelectChapter={handleSelectChapter}
        />
      </div>

      <main
        className={`mx-auto ${
          isPaged
            ? 'w-full max-w-5xl px-2 py-1 pb-20 flex items-center justify-center min-h-[calc(100vh-7.5rem)] overflow-hidden'
            : readingMode === 'longstrip'
            ? 'max-w-3xl px-0 py-0 pb-24'
            : 'max-w-4xl px-2 py-4 pb-24'
        }`}
      >
        {isPagesLoading ? (
          <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
            <Loader2 className="size-8 animate-spin mb-3 text-primary" aria-hidden />
            {currentChapter ? (
              <p className="text-sm font-medium text-muted-foreground/70">
                {currentChapter.meta?.number ?? (currentChapter as any).number}
                {(currentChapter.meta?.title ?? (currentChapter as any).title) && (
                  <> — {currentChapter.meta?.title ?? (currentChapter as any).title}</>
                )}
              </p>
            ) : (
              <p className="text-sm font-medium">Fetching chapter pages...</p>
            )}
          </div>
        ) : isPagesError ? (
          <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-8 text-center text-sm text-destructive my-12">
            Failed to fetch chapter pages.
          </div>
        ) : pages.length === 0 ? (
          <div className="rounded-xl border border-border bg-card p-12 text-center text-sm text-muted-foreground my-12">
            No page images found in chapter.
          </div>
        ) : isPaged ? (
          <ReaderPagedView
            readingMode={readingMode}
            currentPage={currentPage}
            pages={pages}
            effectiveMangaId={effectiveMangaId}
            chapterId={chapterId}
            mangaUrl={manga?.url}
            fitMode={fitMode}
            hasPrevChapter={hasPrevChapter}
            hasNextChapter={hasNextChapter}
            prevChapter={prevChapter}
            nextChapter={nextChapter}
            dragOffset={dragOffset}
            isDragging={isDragging}
            isAnimating={isAnimating}
            didDragRef={didDragRef}
            containerRef={containerRef}
            gestureHandlers={gestureHandlers}
            onNextPage={goToNextPage}
            onPrevPage={goToPrevPage}
            onToggleOverlays={() => setShowOverlays((prev) => !prev)}
            onPrevChapter={handlePrevChapter}
            onNextChapter={handleNextChapter}
          />
        ) : (
          <ReaderContinuousView
            readingMode={readingMode}
            pages={pages}
            effectiveMangaId={effectiveMangaId}
            chapterId={chapterId}
            mangaUrl={manga?.url}
            fitMode={fitMode}
            pageRefs={pageRefs}
            onToggleOverlays={() => setShowOverlays((prev) => !prev)}
            hasPrevChapter={hasPrevChapter}
            hasNextChapter={hasNextChapter}
            prevChapter={prevChapter}
            nextChapter={nextChapter}
            onPrevChapter={handlePrevChapter}
            onNextChapter={handleNextChapter}
          />
        )}
      </main>

      {pages.length > 0 && (
        <div
          className="fixed bottom-0 left-0 right-0 z-50"
          style={{
            transform: showOverlays ? 'translateY(0)' : 'translateY(100%)',
            transition: 'transform 300ms ease-in-out',
          }}
        >
          <ReaderFooter
            currentPage={currentPage}
            totalPages={pages.length}
            readingMode={readingMode}
            fitMode={fitMode}
            onPageChange={handlePageChange}
            onScrollTop={handleScrollTop}
            onFitModeChange={setFitMode}
            onReadingModeChange={handleReadingModeChange}
          />
        </div>
      )}

      {readingHint && <ReaderHint hint={readingHint} onDismiss={dismissReadingHint} />}

      <CompletionPromptDialog
        open={showCompletionDialog}
        mangaId={effectiveMangaId || ''}
        mangaTitle={manga?.title}
        onClose={() => setShowCompletionDialog(false)}
      />
    </div>
  );
};
