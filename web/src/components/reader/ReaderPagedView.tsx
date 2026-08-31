import React from 'react';
import { Chapter, Page } from '../../types/api';
import { FitMode } from '../../hooks/useReaderFitMode';
import { getProxyImageUrl, getPageImageUrl } from '../../lib/utils';
import { ChapterBoundaryCard } from './ChapterBoundaryCard';
import { getFitModeClasses } from './readerUtils';

export interface ReaderPagedViewProps {
  readingMode: string;
  currentPage: number;
  pages: Page[];
  effectiveMangaId?: string;
  chapterId: string;
  chapterProviderId?: string;
  mangaUrl?: string;
  fitMode: FitMode;
  hasPrevChapter: boolean;
  hasNextChapter: boolean;
  prevChapter?: Chapter;
  nextChapter?: Chapter;
  dragOffset: number;
  isDragging: boolean;
  isAnimating: boolean;
  didDragRef?: React.MutableRefObject<boolean>;
  containerRef?: React.Ref<HTMLDivElement>;
  gestureHandlers?: {
    onTouchStart?: (e: React.TouchEvent) => void;
    onTouchMove?: (e: React.TouchEvent) => void;
    onTouchEnd?: (e: React.TouchEvent) => void;
    onTouchCancel?: (e: React.TouchEvent) => void;
    onMouseDown?: (e: React.MouseEvent) => void;
    onMouseMove?: (e: React.MouseEvent) => void;
    onMouseUp?: (e: React.MouseEvent) => void;
    onMouseLeave?: (e: React.MouseEvent) => void;
  };
  onNextPage: () => void;
  onPrevPage: () => void;
  onToggleOverlays: () => void;
  onPrevChapter?: () => void;
  onNextChapter?: () => void;
}

export const ReaderPagedView: React.FC<ReaderPagedViewProps> = ({
  readingMode,
  currentPage,
  pages,
  effectiveMangaId,
  chapterId,
  chapterProviderId,
  mangaUrl,
  fitMode,
  hasPrevChapter,
  hasNextChapter,
  prevChapter,
  nextChapter,
  dragOffset,
  isDragging,
  isAnimating,
  didDragRef,
  containerRef,
  gestureHandlers,
  onNextPage,
  onPrevPage,
  onToggleOverlays,
  onPrevChapter,
  onNextChapter,
}) => {
  const isRTL = readingMode === 'rtl';

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
            src={getPageImageUrl(prevPageData, effectiveMangaId, chapterId, chapterProviderId, mangaUrl)}
            alt={`Page ${currentPage - 1}`}
            className={getFitModeClasses(fitMode, 'rounded-sm shadow-md select-none pointer-events-none')}
            draggable={false}
            onError={(e) => {
              const fallbackUrl = getProxyImageUrl(prevPageData.url, mangaUrl);
              if (fallbackUrl && e.currentTarget.src !== fallbackUrl) {
                e.currentTarget.src = fallbackUrl;
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
          onNavigate={onPrevChapter}
        />
      );
    } else {
      if (currentPage < pages.length) {
        const nextPageData = pages[currentPage];
        if (!nextPageData) return null;
        return (
          <img
            key={`page-${currentPage + 1}`}
            src={getPageImageUrl(nextPageData, effectiveMangaId, chapterId, chapterProviderId, mangaUrl)}
            alt={`Page ${currentPage + 1}`}
            className={getFitModeClasses(fitMode, 'rounded-sm shadow-md select-none pointer-events-none')}
            draggable={false}
            onError={(e) => {
              const fallbackUrl = getProxyImageUrl(nextPageData.url, mangaUrl);
              if (fallbackUrl && e.currentTarget.src !== fallbackUrl) {
                e.currentTarget.src = fallbackUrl;
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
          onNavigate={onNextChapter}
        />
      );
    }
  };

  const beforeSlotContent = renderPageOrBoundary('prev');
  const centerSlotContent = currentPageData ? (
    <img
      key={`page-${currentPage}`}
      src={getPageImageUrl(currentPageData, effectiveMangaId, chapterId, chapterProviderId, mangaUrl)}
      alt={`Page ${currentPage}`}
      className={getFitModeClasses(fitMode, 'rounded-sm shadow-md select-none pointer-events-none')}
      draggable={false}
      onError={(e) => {
        const fallbackUrl = getProxyImageUrl(currentPageData.url, mangaUrl);
        if (fallbackUrl && e.currentTarget.src !== fallbackUrl) {
          e.currentTarget.src = fallbackUrl;
        }
      }}
    />
  ) : null;
  const afterSlotContent = renderPageOrBoundary('next');

  const leftSlotContent = isRTL ? afterSlotContent : beforeSlotContent;
  const rightSlotContent = isRTL ? beforeSlotContent : afterSlotContent;

  return (
    <div
      data-testid="reader-content"
      className="relative flex flex-col items-center justify-center w-full min-h-[calc(100vh-7.5rem)] select-none overflow-hidden"
    >
      <div
        ref={containerRef}
        data-testid="reader-paged-container"
        className="relative flex items-center justify-center w-full min-h-[calc(100vh-7.5rem)] max-h-[calc(100vh-7.5rem)] h-[calc(100vh-7.5rem)] overflow-hidden touch-pan-y select-none"
        {...gestureHandlers}
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
            aria-label={isRTL ? 'Next Page' : 'Previous Page'}
            className="w-[30%] h-full cursor-pointer touch-manipulation"
            onTouchEnd={(e) => {
              e.stopPropagation();
              if (didDragRef?.current || isAnimating) return;
              if (isRTL) {
                onNextPage();
              } else {
                onPrevPage();
              }
            }}
            onClick={() => {
              if (didDragRef?.current || isAnimating) return;
              if (isRTL) {
                onNextPage();
              } else {
                onPrevPage();
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
              if (didDragRef?.current || isAnimating) return;
              onToggleOverlays();
            }}
            onClick={() => {
              if (didDragRef?.current || isAnimating) return;
              onToggleOverlays();
            }}
          />

          {/* Right 30% Zone */}
          <div
            data-testid="reader-zone-right"
            role="button"
            tabIndex={-1}
            aria-label={isRTL ? 'Previous Page' : 'Next Page'}
            className="w-[30%] h-full cursor-pointer touch-manipulation"
            onTouchEnd={(e) => {
              e.stopPropagation();
              if (didDragRef?.current || isAnimating) return;
              if (isRTL) {
                onPrevPage();
              } else {
                onNextPage();
              }
            }}
            onClick={() => {
              if (didDragRef?.current || isAnimating) return;
              if (isRTL) {
                onPrevPage();
              } else {
                onNextPage();
              }
            }}
          />
        </div>
      </div>
    </div>
  );
};
