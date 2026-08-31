import { useEffect, useRef, useCallback } from 'react';

export interface UseReaderScrollTrackerOptions {
  isPaged: boolean;
  pagesCount: number;
  pageRefs: React.MutableRefObject<(HTMLElement | null)[]>;
  onPageVisible: (page: number) => void;
  initialPage?: number;
}

export interface UseReaderScrollTrackerResult {
  scrollToPage: (page: number, smooth?: boolean) => void;
  setReportedPage: (page: number) => void;
}

export function useReaderScrollTracker({
  isPaged,
  pagesCount,
  pageRefs,
  onPageVisible,
  initialPage = 1,
}: UseReaderScrollTrackerOptions): UseReaderScrollTrackerResult {
  const lastReportedPage = useRef(initialPage);
  const rafPending = useRef(false);

  const setReportedPage = useCallback((page: number) => {
    lastReportedPage.current = page;
  }, []);

  const handleScroll = useCallback(() => {
    if (isPaged || rafPending.current) return;
    rafPending.current = true;

    requestAnimationFrame(() => {
      rafPending.current = false;
      if (pagesCount === 0) return;

      let visiblePage = 1;
      const viewportMid = typeof window !== 'undefined' ? window.innerHeight * 0.5 : 400;

      pageRefs.current.forEach((el, idx) => {
        if (el) {
          const rect = el.getBoundingClientRect();
          if (rect.top <= viewportMid) {
            visiblePage = idx + 1;
          }
        }
      });

      if (visiblePage !== lastReportedPage.current) {
        lastReportedPage.current = visiblePage;
        onPageVisible(visiblePage);
      }
    });
  }, [isPaged, pagesCount, pageRefs, onPageVisible]);

  useEffect(() => {
    if (isPaged) return;
    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [isPaged, handleScroll]);

  const scrollToPage = useCallback(
    (page: number, smooth = false) => {
      const targetEl = pageRefs.current[page - 1];
      if (targetEl) {
        targetEl.scrollIntoView({
          behavior: smooth ? 'smooth' : ('instant' as ScrollBehavior),
          block: 'start',
        });
      }
    },
    [pageRefs]
  );

  return {
    scrollToPage,
    setReportedPage,
  };
}
