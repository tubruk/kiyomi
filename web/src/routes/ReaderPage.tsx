import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useSearch, useNavigate } from '@tanstack/react-router';
import { Loader2, BookOpen, CheckCircle2 } from 'lucide-react';

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
import { useReadingHint } from '../hooks/useReadingHint';
import { useReaderFitMode } from '../hooks/useReaderFitMode';
import { getProxyImageUrl, formatChapterTitleWithPage } from '../lib/utils';
import { Chapter } from '../types/api';

interface ReaderSearch {
  mangaId?: string;
  page?: number | 'last' | string;
}

interface ChapterBoundaryCardProps {
  type: 'prev' | 'next';
  hasChapter: boolean;
  targetChapter?: Chapter;
}

const ChapterBoundaryCard: React.FC<ChapterBoundaryCardProps> = ({
  type,
  hasChapter,
  targetChapter,
}) => {
  const isPrev = type === 'prev';
  const chapterIdent = targetChapter
    ? targetChapter.number || targetChapter.name || targetChapter.title || ''
    : '';

  const label = isPrev
    ? hasChapter && targetChapter
      ? `Previous chapter (${chapterIdent})`
      : 'No previous chapter'
    : hasChapter && targetChapter
    ? `Next chapter (${chapterIdent})`
    : 'No next chapter';

  const subtitle = hasChapter
    ? isPrev
      ? 'Swipe or click to read previous chapter'
      : 'Swipe or click to continue to next chapter'
    : isPrev
    ? 'You are at the beginning of the manga'
    : 'You have reached the latest chapter';

  return (
    <div
      data-testid={isPrev ? 'boundary-card-prev' : 'boundary-card-next'}
      className="flex flex-col items-center justify-center p-8 mx-4 max-w-sm w-full rounded-2xl border border-border/70 bg-card/90 backdrop-blur-lg shadow-xl text-center select-none"
    >
      <div className="size-14 rounded-full bg-primary/10 flex items-center justify-center text-primary mb-4 border border-primary/20">
        {hasChapter ? (
          <BookOpen className="size-6" aria-hidden />
        ) : (
          <CheckCircle2 className="size-6 text-muted-foreground" aria-hidden />
        )}
      </div>
      <h3 className="text-base sm:text-lg font-semibold text-foreground tracking-tight">
        {label}
      </h3>
      <p className="mt-1.5 text-xs text-muted-foreground leading-relaxed">
        {subtitle}
      </p>
    </div>
  );
};

export const ReaderPage: React.FC = () => {
  const params = useParams({ strict: false }) as Record<string, string | undefined>;
  const chapterId = params.chapterId || '';
  const searchParams = useSearch({ strict: false }) as ReaderSearch;
  const navigate = useNavigate();
  const { showToast } = useToast();

  const mangaId = params.mangaId || searchParams.mangaId;
  const providerId = params.providerId;
  const remoteId = params.remoteId;

  const [currentPage, setCurrentPage] = useState(1);
  const [showOverlays, setShowOverlays] = useState(true);
  const [dragOffset, setDragOffset] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isAnimating, setIsAnimating] = useState(false);
  const [showCompletionDialog, setShowCompletionDialog] = useState(false);

  const containerRef = useRef<HTMLDivElement | null>(null);
  const pageRefs = useRef<(HTMLDivElement | null)[]>([]);
  const prevChapterIdRef = useRef<string | null>(null);

  // Fetch library list to resolve effective local manga ID if needed
  const { data: libraryManga = [] } = useLibraryManga();
  const libraryEntry = mangaId
    ? libraryManga.find((m) => m.id === mangaId)
    : (providerId && remoteId)
    ? libraryManga.find(
        (m) =>
          (m.contentProviderId === providerId || m.sourceId === providerId || m.meta?.content?.provider_id === providerId) &&
          (m.contentRemoteId === remoteId || m.url === remoteId || m.id === remoteId || m.meta?.content?.provider_manga_id === remoteId)
      )
    : undefined;

  const effectiveMangaId = mangaId || libraryEntry?.id;

  // Fetch manga details (local vs remote)
  const { data: localManga } = useMangaDetails(effectiveMangaId || '', { enabled: Boolean(effectiveMangaId) });
  const { data: remoteManga } = useProviderMangaDetails(providerId || '', remoteId || '', {
    enabled: Boolean(!effectiveMangaId && providerId && remoteId),
  });
  const manga = effectiveMangaId ? localManga : remoteManga;

  // Fetch chapter pages
  const {
    data: pagesData,
    isLoading: isPagesLoading,
    isError: isPagesError,
  } = useChapterPages(chapterId, {
    mangaId: effectiveMangaId || remoteId || undefined,
    providerId: manga?.contentProviderId || manga?.meta?.content?.provider_id || providerId,
    enabled: Boolean(chapterId),
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

  // Toast notification on chapter transition
  useEffect(() => {
    if (chapterId && currentChapter && prevChapterIdRef.current !== chapterId) {
      const chNum = currentChapter.meta?.number ?? (currentChapter as any).number;
      const chTitle = currentChapter.meta?.title ?? (currentChapter as any).title;
      showToast(chTitle ? `Chapter ${chNum}: ${chTitle}` : `Chapter ${chNum}`, 'info', undefined, 'subtle');
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

  const hasPrevChapter = currentChapterIndex > 0;
  const hasNextChapter =
    currentChapterIndex >= 0 && currentChapterIndex < chapters.length - 1;

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
  ).trim().toLowerCase();

  const readingMode = ['rtl', 'ltr', 'vertical', 'longstrip'].includes(rawReadingMode)
    ? rawReadingMode
    : 'rtl';

  const isPaged = readingMode === 'rtl' || readingMode === 'ltr';

  // Reading mode hint
  const { hint: readingHint, dismissHint: dismissReadingHint } = useReadingHint(effectiveMangaId, readingMode);

  // Fit mode
  const { fitMode, setFitMode } = useReaderFitMode();

  // Helper to get fit mode classes for page images
  const getFitModeClasses = (baseClasses: string): string => {
    const fitWidthClass = 'max-w-full h-auto';
    const fitHeightClass = 'max-h-[calc(100vh-7.5rem)] w-auto max-w-full object-contain';
    const fitOriginalClass = 'max-w-none h-auto';

    let fitClass: string;
    switch (fitMode) {
      case 'fit-width':
        fitClass = fitWidthClass;
        break;
      case 'fit-original':
        fitClass = fitOriginalClass;
        break;
      case 'fit-height':
      default:
        fitClass = fitHeightClass;
        break;
    }

    // Replace fit-related classes in baseClasses, then append fitClass
    const withoutFit = baseClasses
      .replace(/max-w-full|max-w-none/g, '')
      .replace(/max-h-\[calc\(100vh-7\.5rem\)\]/g, '')
      .replace(/w-auto|h-auto/g, '')
      .replace(/object-contain/g, '')
      .replace(/\s+/g, ' ')
      .trim();

    return `${withoutFit} ${fitClass}`.trim();
  };

  // Progress update mutation
  const updateProgressMutation = useUpdateChapterProgressMutation();

  // Manga metadata update mutation
  const updateLibraryMangaMutation = useUpdateLibraryMangaMutation();

  const isChapterRead = Boolean(currentChapter?.meta?.is_read ?? (currentChapter as any)?.is_read);
  const chapterLastReadPage = currentChapter?.meta?.last_read_page ?? (currentChapter as any)?.last_read_page ?? 0;

  const hasMarkedRead = useRef(false);
  const hasResumed = useRef(false);
  const hasDismissedCompletion = useRef(false);
  const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const animTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const touchStartPos = useRef<{ x: number; y: number } | null>(null);
  const didDragRef = useRef(false);

  const getContainerWidth = useCallback(() => {
    return containerRef.current?.clientWidth || (typeof window !== 'undefined' ? window.innerWidth : 800);
  }, []);

  // Reset chapter tracking state on chapter switch
  useEffect(() => {
    hasMarkedRead.current = isChapterRead;
    hasResumed.current = false;
    hasDismissedCompletion.current = false;
    lastReportedPage.current = 1;
    setDragOffset(0);
    setIsDragging(false);
    setIsAnimating(false);
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current);
    }
    if (animTimeoutRef.current) {
      clearTimeout(animTimeoutRef.current);
    }
  }, [chapterId, isChapterRead]);


  // Restore initial reading position to last_read_page or requested target page when loading a chapter
  useEffect(() => {
    if (!chapterId || pages.length === 0 || hasResumed.current) return;

    let targetResumePage = 1;
    const rawPage = searchParams.page;

    if (rawPage === 'last') {
      targetResumePage = pages.length;
    } else if (
      rawPage !== undefined &&
      rawPage !== null &&
      rawPage !== '' &&
      !isNaN(Number(rawPage))
    ) {
      const parsedPage = Number(rawPage);
      targetResumePage = Math.min(Math.max(1, parsedPage), pages.length);
    } else if (!isChapterRead && chapterLastReadPage > 1) {
      targetResumePage = Math.min(Math.max(1, chapterLastReadPage), pages.length);
    }

    if (isPaged) {
      hasResumed.current = true;
      setCurrentPage(targetResumePage);
      lastReportedPage.current = targetResumePage;
      return;
    }

    if (targetResumePage > 1) {
      hasResumed.current = true;
      setCurrentPage(targetResumePage);
      lastReportedPage.current = targetResumePage;
      setTimeout(() => {
        const targetEl = pageRefs.current[targetResumePage - 1];
        if (targetEl) {
          targetEl.scrollIntoView({ behavior: 'instant', block: 'start' });
        }
      }, 0);
    } else {
      window.scrollTo({ top: 0, behavior: 'instant' });
      setCurrentPage(1);
      lastReportedPage.current = 1;
    }
  }, [chapterId, pages.length, searchParams.page, isChapterRead, chapterLastReadPage, isPaged]);

  // Debounced auto-save (1.5s idle) & auto mark-as-read on last page
  useEffect(() => {
    if (!effectiveMangaId || !chapterId || pages.length === 0) return;

    // Auto mark-as-read when reaching last page
    if (currentPage === pages.length) {
      if (!hasMarkedRead.current) {
        hasMarkedRead.current = true;
        updateProgressMutation.mutate({
          mangaId: effectiveMangaId,
          chapterId,
          progress: { is_read: true, last_read_page: currentPage },
        });

        // Trigger completion prompt if on last chapter and manga is in "reading" status
        if (
          effectiveMangaId &&
          !hasNextChapter &&
          !hasDismissedCompletion.current &&
          (manga?.user_status === 'reading' || manga?.meta?.user_status === 'reading')
        ) {
          hasDismissedCompletion.current = true;
          setShowCompletionDialog(true);
        }
      }
      return;
    }

    // Debounced auto-save (1.5s idle) sending { last_read_page: P }
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current);
    }

    saveTimeoutRef.current = setTimeout(() => {
      updateProgressMutation.mutate({
        mangaId: effectiveMangaId,
        chapterId,
        progress: { last_read_page: currentPage },
      });
    }, 1500);

    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, [currentPage, effectiveMangaId, chapterId, pages.length]);

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
    if (effectiveMangaId && chapterId) {
      hasMarkedRead.current = true;
      updateProgressMutation.mutate({
        mangaId: effectiveMangaId,
        chapterId,
        progress: { is_read: true },
      });
    }
    if (hasNextChapter && nextChapter) {
      handleSelectChapter(nextChapter.id);
    }
  }, [effectiveMangaId, chapterId, hasNextChapter, nextChapter, handleSelectChapter, updateProgressMutation]);

  // Directional Paged Navigation Handlers
  const goToNextPage = useCallback(() => {
    if (isAnimating) return;
    const slideDistance = getContainerWidth();

    if (currentPage < pages.length) {
      setIsAnimating(true);
      // In RTL, next page is in Left Slot (+slideDistance). In LTR, next page is in Right Slot (-slideDistance).
      const targetOffset = readingMode === 'rtl' ? slideDistance : -slideDistance;
      setDragOffset(targetOffset);
      animTimeoutRef.current = setTimeout(() => {
        setDragOffset(0);
        setCurrentPage((prev) => prev + 1);
        setIsAnimating(false);
        setTimeout(() => {
          didDragRef.current = false;
        }, 50);
      }, 220);
    } else {
      // Past last page -> mark read and transition to next chapter if available
      if (effectiveMangaId && chapterId) {
        hasMarkedRead.current = true;
        updateProgressMutation.mutate({
          mangaId: effectiveMangaId,
          chapterId,
          progress: { is_read: true, last_read_page: pages.length },
        });
      }
      if (hasNextChapter) {
        setIsAnimating(true);
        const targetOffset = readingMode === 'rtl' ? slideDistance : -slideDistance;
        setDragOffset(targetOffset);
        animTimeoutRef.current = setTimeout(() => {
          setDragOffset(0);
          setIsAnimating(false);
          handleNextChapter();
          setTimeout(() => {
            didDragRef.current = false;
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
    effectiveMangaId,
    chapterId,
    updateProgressMutation,
    hasNextChapter,
    handleNextChapter,
  ]);

  const goToPrevPage = useCallback(() => {
    if (isAnimating) return;
    const slideDistance = getContainerWidth();

    if (currentPage > 1) {
      setIsAnimating(true);
      // In RTL, prev page is in Right Slot (-slideDistance). In LTR, prev page is in Left Slot (+slideDistance).
      const targetOffset = readingMode === 'rtl' ? -slideDistance : slideDistance;
      setDragOffset(targetOffset);
      animTimeoutRef.current = setTimeout(() => {
        setDragOffset(0);
        setCurrentPage((prev) => prev - 1);
        setIsAnimating(false);
        setTimeout(() => {
          didDragRef.current = false;
        }, 50);
      }, 220);
    } else if (hasPrevChapter) {
      setIsAnimating(true);
      const targetOffset = readingMode === 'rtl' ? -slideDistance : slideDistance;
      setDragOffset(targetOffset);
      animTimeoutRef.current = setTimeout(() => {
        setDragOffset(0);
        setIsAnimating(false);
        handlePrevChapter();
        setTimeout(() => {
          didDragRef.current = false;
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
  ]);

  // Preload adjacent images in Paged mode
  useEffect(() => {
    if (!isPaged || pages.length === 0) return;
    const nextIdx = currentPage; // 0-indexed index for currentPage + 1
    if (nextIdx < pages.length) {
      const nextImg = new Image();
      const p = pages[nextIdx];
      nextImg.src = p.assetUrl || getProxyImageUrl(p.url, manga?.url);
    }
    const nextNextIdx = currentPage + 1; // 0-indexed index for currentPage + 2
    if (nextNextIdx < pages.length) {
      const nextNextImg = new Image();
      const p = pages[nextNextIdx];
      nextNextImg.src = p.assetUrl || getProxyImageUrl(p.url, manga?.url);
    }
    const prevIdx = currentPage - 2; // 0-indexed index for currentPage - 1
    if (prevIdx >= 0) {
      const prevImg = new Image();
      const p = pages[prevIdx];
      prevImg.src = p.assetUrl || getProxyImageUrl(p.url, manga?.url);
    }
  }, [currentPage, isPaged, pages, manga?.url]);

  // Keyboard navigation for Paged mode
  useEffect(() => {
    if (!isPaged) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable ||
        target.closest('[role="dialog"]')
      ) {
        return;
      }

      if (readingMode === 'rtl') {
        if (e.key === 'ArrowLeft' || e.key === ' ' || e.code === 'Space') {
          e.preventDefault();
          goToNextPage();
        } else if (e.key === 'ArrowRight') {
          e.preventDefault();
          goToPrevPage();
        }
      } else {
        // LTR
        if (e.key === 'ArrowRight' || e.key === ' ' || e.code === 'Space') {
          e.preventDefault();
          goToNextPage();
        } else if (e.key === 'ArrowLeft') {
          e.preventDefault();
          goToPrevPage();
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isPaged, readingMode, goToNextPage, goToPrevPage]);

  // Touch Drag & Swipe Handlers for mobile & touchscreens
  const handleTouchStart = (e: React.TouchEvent) => {
    if (!isPaged || pages.length === 0 || isAnimating) return;
    if (e.touches.length === 1) {
      touchStartPos.current = {
        x: e.touches[0].clientX,
        y: e.touches[0].clientY,
      };
      didDragRef.current = false;
      setIsDragging(false);
      setDragOffset(0);
    }
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!touchStartPos.current || !isPaged || isAnimating || e.touches.length !== 1) return;

    const currentX = e.touches[0].clientX;
    const currentY = e.touches[0].clientY;
    const deltaX = currentX - touchStartPos.current.x;
    const deltaY = currentY - touchStartPos.current.y;

    // Check if horizontal drag dominates vertical scroll
    if (Math.abs(deltaX) > 10 && Math.abs(deltaX) > Math.abs(deltaY)) {
      didDragRef.current = true;
      setIsDragging(true);
      setDragOffset(deltaX);
    }
  };

  const handleTouchEnd = () => {
    if (!touchStartPos.current || !isPaged) return;

    const currentDrag = dragOffset;
    const wasDragging = didDragRef.current;
    touchStartPos.current = null;
    setIsDragging(false);

    if (!wasDragging || isAnimating) {
      setDragOffset(0);
      return;
    }

    const containerWidth = getContainerWidth();
    const threshold = Math.max(50, containerWidth * 0.15);

    if (Math.abs(currentDrag) >= threshold) {
      // Determine intended navigation direction based on readingMode and swipe
      // RTL: Drag right (currentDrag > 0) -> Next Page, Drag left (currentDrag < 0) -> Prev Page
      // LTR: Drag left (currentDrag < 0) -> Next Page, Drag right (currentDrag > 0) -> Prev Page
      const isNext = readingMode === 'rtl' ? currentDrag > 0 : currentDrag < 0;

      if (isNext) {
        if (currentPage < pages.length || hasNextChapter) {
          goToNextPage();
        } else {
          setIsAnimating(true);
          setDragOffset(0);
          setTimeout(() => {
            setIsAnimating(false);
            didDragRef.current = false;
          }, 220);
        }
      } else {
        if (currentPage > 1 || hasPrevChapter) {
          goToPrevPage();
        } else {
          setIsAnimating(true);
          setDragOffset(0);
          setTimeout(() => {
            setIsAnimating(false);
            didDragRef.current = false;
          }, 220);
        }
      }
    } else {
      // Below threshold: spring back smoothly to 0
      setIsAnimating(true);
      setDragOffset(0);
      setTimeout(() => {
        setIsAnimating(false);
        didDragRef.current = false;
      }, 220);
    }
  };

  const handleTouchCancel = () => {
    touchStartPos.current = null;
    setIsDragging(false);
    setDragOffset(0);
    setTimeout(() => {
      didDragRef.current = false;
    }, 100);
  };

  const handlePageChange = (targetPage: number) => {
    if (targetPage !== currentPage) {
      setDragOffset(0);
      setIsDragging(false);
      setIsAnimating(false);
      setCurrentPage(targetPage);
    }
    if (!isPaged) {
      const targetEl = pageRefs.current[targetPage - 1];
      if (targetEl) {
        targetEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    }
    // Sync page to URL for bookmarking
    navigate({ search: { ...searchParams, page: targetPage } as any, replace: true });
  };

  // Scroll tracking — passive + rAF throttle (for continuous vertical / longstrip modes)
  const lastReportedPage = useRef(1);
  const rafPending = useRef(false);
  const handleScroll = useCallback(() => {
    if (isPaged || rafPending.current) return;
    rafPending.current = true;
    requestAnimationFrame(() => {
      rafPending.current = false;
      if (pages.length === 0) return;
      let visiblePage = 1;
      pageRefs.current.forEach((el, idx) => {
        if (el) {
          const rect = el.getBoundingClientRect();
          if (rect.top <= window.innerHeight * 0.5) {
            visiblePage = idx + 1;
          }
        }
      });
      if (visiblePage !== lastReportedPage.current) {
        lastReportedPage.current = visiblePage;
        setCurrentPage(visiblePage);
      }
    });
  }, [isPaged, pages.length]);

  useEffect(() => {
    if (isPaged) return;
    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [isPaged, handleScroll]);

  const handleScrollTop = () => {
    if (isPaged) {
      if (currentPage !== 1) {
        setDragOffset(0);
        setIsDragging(false);
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
      updateLibraryMangaMutation.mutate({
        mangaId: effectiveMangaId,
        fields: {
          reading_mode: mode,
          readingDirection: mode,
          content: { reading_mode: mode },
        },
      });
    }
  };

  const transitionStyle = isDragging
    ? 'none'
    : isAnimating
    ? 'transform 220ms cubic-bezier(0.16, 1, 0.3, 1)'
    : 'none';

  const currentPageData = pages[currentPage - 1];

  const renderPageOrBoundary = (slotType: 'prev' | 'next') => {
    if (slotType === 'prev') {
      if (currentPage > 1) {
        const prevPageData = pages[currentPage - 2];
        if (!prevPageData) return null;
        return (
          <img
            key={`page-${currentPage - 1}`}
            src={prevPageData.assetUrl || getProxyImageUrl(prevPageData.url, manga?.url)}
            alt={`Page ${currentPage - 1}`}
            className={getFitModeClasses('rounded-sm shadow-md select-none pointer-events-none')}
            draggable={false}
            onError={(e) => {
              if (prevPageData.url && e.currentTarget.src !== prevPageData.url) {
                e.currentTarget.src = prevPageData.url;
              }
            }}
          />
        );
      }
      return (
        <ChapterBoundaryCard
          type="prev"
          hasChapter={hasPrevChapter}
          targetChapter={prevChapter}
        />
      );
    } else {
      if (currentPage < pages.length) {
        const nextPageData = pages[currentPage];
        if (!nextPageData) return null;
        return (
          <img
            key={`page-${currentPage + 1}`}
            src={nextPageData.assetUrl || getProxyImageUrl(nextPageData.url, manga?.url)}
            alt={`Page ${currentPage + 1}`}
            className={getFitModeClasses('rounded-sm shadow-md select-none pointer-events-none')}
            draggable={false}
            onError={(e) => {
              if (nextPageData.url && e.currentTarget.src !== nextPageData.url) {
                e.currentTarget.src = nextPageData.url;
              }
            }}
          />
        );
      }
      return (
        <ChapterBoundaryCard
          type="next"
          hasChapter={hasNextChapter}
          targetChapter={nextChapter}
        />
      );
    }
  };

  const beforeSlotContent = renderPageOrBoundary('prev');
  const centerSlotContent = currentPageData ? (
    <img
      key={`page-${currentPage}`}
      src={currentPageData.assetUrl || getProxyImageUrl(currentPageData.url, manga?.url)}
      alt={`Page ${currentPage}`}
      className={getFitModeClasses('rounded-sm shadow-md select-none pointer-events-none')}
      draggable={false}
      onError={(e) => {
        if (currentPageData.url && e.currentTarget.src !== currentPageData.url) {
          e.currentTarget.src = currentPageData.url;
        }
      }}
    />
  ) : null;
  const afterSlotContent = renderPageOrBoundary('next');

  const leftSlotContent = readingMode === 'rtl' ? afterSlotContent : beforeSlotContent;
  const rightSlotContent = readingMode === 'rtl' ? beforeSlotContent : afterSlotContent;

  return (
    <div className={`min-h-screen transition-colors duration-200 bg-black pt-14`}>
      <div className={`fixed top-0 left-0 right-0 z-50 transition-transform duration-300 ease-in-out ${showOverlays ? 'translate-y-0' : '-translate-y-full'}`}>
        <ReaderTopBar
          mangaTitle={manga?.title}
          chapterTitle={chapters[currentChapterIndex]?.title || chapters[currentChapterIndex]?.name || 'Chapter View'}
          currentPage={currentPage}
          totalPages={pages.length}
          chapterNumber={chapters[currentChapterIndex]?.number ?? chapters[currentChapterIndex]?.meta?.number}
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
          /* Horizontal Paged View (RTL & LTR) */
          <div
            data-testid="reader-content"
            className="relative flex flex-col items-center justify-center w-full min-h-[calc(100vh-7.5rem)] select-none overflow-hidden"
          >
            <div
              ref={containerRef}
              data-testid="reader-paged-container"
              className="relative flex items-center justify-center w-full min-h-[calc(100vh-7.5rem)] max-h-[calc(100vh-7.5rem)] h-[calc(100vh-7.5rem)] overflow-hidden touch-pan-y select-none"
              onTouchStart={handleTouchStart}
              onTouchMove={handleTouchMove}
              onTouchEnd={handleTouchEnd}
              onTouchCancel={handleTouchCancel}
            >
              {/* Left Slot */}
              <div
                data-testid="reader-slot-left"
                className="absolute inset-0 flex items-center justify-center will-change-transform pointer-events-none"
                style={{
                  transform: `translateX(calc(-100% + ${dragOffset}px))`,
                  transition: transitionStyle,
                }}
              >
                {leftSlotContent}
              </div>

              {/* Center Slot */}
              <div
                data-testid="reader-slot-center"
                className="absolute inset-0 flex items-center justify-center will-change-transform pointer-events-none"
                style={{
                  transform: `translateX(${dragOffset}px)`,
                  transition: transitionStyle,
                }}
              >
                {centerSlotContent}
              </div>

              {/* Right Slot */}
              <div
                data-testid="reader-slot-right"
                className="absolute inset-0 flex items-center justify-center will-change-transform pointer-events-none"
                style={{
                  transform: `translateX(calc(100% + ${dragOffset}px))`,
                  transition: transitionStyle,
                }}
              >
                {rightSlotContent}
              </div>

              {/* Click Navigation Zones */}
              <div className="absolute inset-0 flex select-none z-10 pointer-events-auto">
                {/* Left 30% Zone */}
                <div
                  data-testid="reader-zone-left"
                  role="button"
                  tabIndex={-1}
                  aria-label={readingMode === 'rtl' ? 'Next Page' : 'Previous Page'}
                  className="w-[30%] h-full cursor-pointer touch-manipulation"
                  onTouchEnd={(e) => {
                    e.stopPropagation();
                    if (didDragRef.current || isAnimating) return;
                    if (readingMode === 'rtl') {
                      goToNextPage();
                    } else {
                      goToPrevPage();
                    }
                  }}
                  onClick={() => {
                    if (didDragRef.current || isAnimating) return;
                    if (readingMode === 'rtl') {
                      goToNextPage();
                    } else {
                      goToPrevPage();
                    }
                  }}
                />
                {/* Center 40% Zone */}
                <div
                  data-testid="reader-zone-center"
                  role="button"
                  tabIndex={-1}
                  aria-label="Toggle Overlays"
                  className="w-[40%] h-full cursor-pointer touch-manipulation"
                  onTouchEnd={(e) => {
                    e.stopPropagation();
                    e.preventDefault();
                    if (didDragRef.current || isAnimating) return;
                    setShowOverlays((prev) => !prev);
                  }}
                  onClick={() => {
                    if (didDragRef.current || isAnimating) return;
                    setShowOverlays((prev) => !prev);
                  }}
                />
                {/* Right 30% Zone */}
                <div
                  data-testid="reader-zone-right"
                  role="button"
                  tabIndex={-1}
                  aria-label={readingMode === 'rtl' ? 'Previous Page' : 'Next Page'}
                  className="w-[30%] h-full cursor-pointer touch-manipulation"
                  onTouchEnd={(e) => {
                    e.stopPropagation();
                    if (didDragRef.current || isAnimating) return;
                    if (readingMode === 'rtl') {
                      goToPrevPage();
                    } else {
                      goToNextPage();
                    }
                  }}
                  onClick={() => {
                    if (didDragRef.current || isAnimating) return;
                    if (readingMode === 'rtl') {
                      goToPrevPage();
                    } else {
                      goToNextPage();
                    }
                  }}
                />
              </div>
            </div>
          </div>
        ) : (
          /* Continuous Vertical View (Longstrip & Vertical) */
          <div
            data-testid="reader-content"
            className={
              readingMode === 'longstrip'
                ? 'flex flex-col gap-0 items-center w-full max-w-3xl mx-auto cursor-pointer'
                : 'flex flex-col gap-6 items-center w-full max-w-3xl mx-auto my-4 cursor-pointer'
            }
            onClick={() => setShowOverlays((prev) => !prev)}
          >
            {pages.map((p) => {
              const originalIndex = p.index;
              const pageNum = originalIndex + 1;
              const imgSrc = p.assetUrl || getProxyImageUrl(p.url, manga?.url);

              if (readingMode === 'longstrip') {
                return (
                  <div
                    key={p.index}
                    ref={(el) => (pageRefs.current[originalIndex] = el)}
                    className="w-full text-center leading-none"
                  >
                    <img
                      src={imgSrc}
                      alt={`Page ${pageNum}`}
                      className={getFitModeClasses('block transition-opacity duration-300 min-h-[100px]')}
                      loading="lazy"
                      onError={(e) => {
                        if (p.url && e.currentTarget.src !== p.url) {
                          e.currentTarget.src = p.url;
                        }
                      }}
                    />
                  </div>
                );
              }

              return (
                <div
                  key={p.index}
                  ref={(el) => (pageRefs.current[originalIndex] = el)}
                  className="w-full text-center"
                >
                  <img
                    src={imgSrc}
                    alt={`Page ${pageNum}`}
                    className={getFitModeClasses('rounded-sm shadow-md transition-opacity duration-300 min-h-[300px]')}
                    loading="lazy"
                    onError={(e) => {
                      if (p.url && e.currentTarget.src !== p.url) {
                        e.currentTarget.src = p.url;
                      }
                    }}
                  />
                  <div className="mt-1 text-[10px] text-muted-foreground/70 font-mono">
                    p.{pageNum}
                  </div>
                </div>
              );
            })}
          </div>
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
