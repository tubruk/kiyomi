import { useState, useRef, useEffect, useCallback } from 'react';

export interface UseReaderProgressTrackerOptions {
  effectiveMangaId?: string;
  chapterId: string;
  chapterProviderId?: string;
  pagesCount: number;
  isPaged: boolean;
  isChapterRead: boolean;
  chapterLastReadPage: number;
  searchPage?: number | 'last' | string;
  hasNextChapter: boolean;
  userStatus?: string;
  onScrollToPage?: (page: number) => void;
  onUpdateProgress?: (params: {
    mangaId: string;
    chapterId: string;
    providerId?: string;
    progress: { is_read?: boolean; last_read_page?: number };
  }) => void;
}

export interface UseReaderProgressTrackerResult {
  currentPage: number;
  setCurrentPage: React.Dispatch<React.SetStateAction<number>>;
  showCompletionDialog: boolean;
  setShowCompletionDialog: React.Dispatch<React.SetStateAction<boolean>>;
  markChapterRead: (lastReadPage?: number) => void;
  hasResumed: boolean;
}

export function useReaderProgressTracker({
  effectiveMangaId,
  chapterId,
  chapterProviderId,
  pagesCount,
  isPaged,
  isChapterRead,
  chapterLastReadPage,
  searchPage,
  hasNextChapter,
  userStatus,
  onScrollToPage,
  onUpdateProgress,
}: UseReaderProgressTrackerOptions): UseReaderProgressTrackerResult {
  const [currentPage, setCurrentPage] = useState(1);
  const [showCompletionDialog, setShowCompletionDialog] = useState(false);

  const activeChapterIdRef = useRef<string | null>(null);
  const hasMarkedRead = useRef(false);
  const hasResumed = useRef(false);
  const hasDismissedCompletion = useRef(false);
  const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  // Reset tracking state on chapter switch
  useEffect(() => {
    if (activeChapterIdRef.current !== chapterId) {
      activeChapterIdRef.current = chapterId;
      hasMarkedRead.current = isChapterRead;
      hasResumed.current = false;
      hasDismissedCompletion.current = false;
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
        saveTimeoutRef.current = null;
      }
    }
  }, [chapterId, isChapterRead]);

  // Restore initial reading position
  useEffect(() => {
    if (!chapterId || pagesCount === 0 || hasResumed.current) return;

    let targetResumePage = 1;

    if (searchPage === 'last') {
      targetResumePage = pagesCount;
    } else if (
      searchPage !== undefined &&
      searchPage !== null &&
      searchPage !== '' &&
      !isNaN(Number(searchPage))
    ) {
      const parsedPage = Number(searchPage);
      targetResumePage = Math.min(Math.max(1, parsedPage), pagesCount);
    } else if (!isChapterRead && chapterLastReadPage > 1) {
      targetResumePage = Math.min(Math.max(1, chapterLastReadPage), pagesCount);
    }

    hasResumed.current = true;
    setCurrentPage(targetResumePage);

    if (!isPaged) {
      if (targetResumePage > 1) {
        setTimeout(() => {
          onScrollToPage?.(targetResumePage);
        }, 0);
      } else if (typeof window !== 'undefined') {
        window.scrollTo({ top: 0, behavior: 'instant' as ScrollBehavior });
      }
    }
  }, [chapterId, pagesCount, searchPage, isChapterRead, chapterLastReadPage, isPaged, onScrollToPage]);

  // Debounced auto-save (1.5s idle) & auto mark-as-read on last page
  useEffect(() => {
    if (!effectiveMangaId || !chapterId || pagesCount === 0 || !hasResumed.current) return;

    // Auto mark-as-read when reaching last page
    if (currentPage === pagesCount) {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
        saveTimeoutRef.current = null;
      }

      if (!hasMarkedRead.current) {
        hasMarkedRead.current = true;
        onUpdateProgress?.({
          mangaId: effectiveMangaId,
          chapterId,
          providerId: chapterProviderId,
          progress: { is_read: true, last_read_page: currentPage },
        });

        // Trigger completion prompt if on last chapter and manga is in "reading" status
        if (
          !hasNextChapter &&
          !hasDismissedCompletion.current &&
          userStatus === 'reading'
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
      onUpdateProgress?.({
        mangaId: effectiveMangaId,
        chapterId,
        providerId: chapterProviderId,
        progress: { last_read_page: currentPage },
      });
    }, 1500);

    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, [
    currentPage,
    effectiveMangaId,
    chapterId,
    chapterProviderId,
    pagesCount,
    hasNextChapter,
    userStatus,
    onUpdateProgress,
  ]);

  const markChapterRead = useCallback(
    (lastReadPage?: number) => {
      if (effectiveMangaId && chapterId) {
        hasMarkedRead.current = true;
        onUpdateProgress?.({
          mangaId: effectiveMangaId,
          chapterId,
          providerId: chapterProviderId,
          progress: {
            is_read: true,
            ...(lastReadPage !== undefined ? { last_read_page: lastReadPage } : {}),
          },
        });
      }
    },
    [effectiveMangaId, chapterId, chapterProviderId, onUpdateProgress]
  );

  return {
    currentPage,
    setCurrentPage,
    showCompletionDialog,
    setShowCompletionDialog,
    markChapterRead,
    hasResumed: hasResumed.current,
  };
}
