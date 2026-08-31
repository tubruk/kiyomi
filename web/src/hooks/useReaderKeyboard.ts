import { useEffect } from 'react';

export interface UseReaderKeyboardOptions {
  readingMode: string;
  isPaged?: boolean;
  enabled?: boolean;
  onNextPage?: () => void;
  onPrevPage?: () => void;
  onFirstPage?: () => void;
  onLastPage?: () => void;
  onToggleOverlays?: () => void;
  onToggleFullscreen?: () => void;
}

export function useReaderKeyboard({
  readingMode,
  isPaged = true,
  enabled = true,
  onNextPage,
  onPrevPage,
  onFirstPage,
  onLastPage,
  onToggleOverlays,
  onToggleFullscreen,
}: UseReaderKeyboardOptions): void {
  useEffect(() => {
    if (!enabled) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.isContentEditable ||
          Boolean(target.closest?.('[role="dialog"]')))
      ) {
        return;
      }

      if (e.ctrlKey || e.metaKey || e.altKey) {
        return;
      }

      const isRTL = readingMode === 'rtl';

      // Fullscreen toggle ('f' or 'F')
      if (e.key === 'f' || e.key === 'F') {
        e.preventDefault();
        if (onToggleFullscreen) {
          onToggleFullscreen();
        } else if (typeof document !== 'undefined') {
          if (!document.fullscreenElement) {
            document.documentElement.requestFullscreen?.().catch(() => {});
          } else {
            document.exitFullscreen?.().catch(() => {});
          }
        }
        return;
      }

      // Menu / Overlays toggle ('m' or 'M')
      if (e.key === 'm' || e.key === 'M') {
        e.preventDefault();
        onToggleOverlays?.();
        return;
      }

      // Home & End
      if (e.key === 'Home') {
        e.preventDefault();
        onFirstPage?.();
        return;
      }

      if (e.key === 'End') {
        e.preventDefault();
        onLastPage?.();
        return;
      }

      // PageUp & PageDown
      if (e.key === 'PageUp') {
        e.preventDefault();
        onPrevPage?.();
        return;
      }

      if (e.key === 'PageDown') {
        e.preventDefault();
        onNextPage?.();
        return;
      }

      // Spacebar
      if (e.key === ' ' || e.code === 'Space') {
        e.preventDefault();
        if (e.shiftKey) {
          onPrevPage?.();
        } else {
          onNextPage?.();
        }
        return;
      }

      // Arrow keys (Paged mode primarily, or continuous mode)
      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        if (isRTL) {
          onNextPage?.();
        } else {
          onPrevPage?.();
        }
        return;
      }

      if (e.key === 'ArrowRight') {
        e.preventDefault();
        if (isRTL) {
          onPrevPage?.();
        } else {
          onNextPage?.();
        }
        return;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [
    readingMode,
    isPaged,
    enabled,
    onNextPage,
    onPrevPage,
    onFirstPage,
    onLastPage,
    onToggleOverlays,
    onToggleFullscreen,
  ]);
}
