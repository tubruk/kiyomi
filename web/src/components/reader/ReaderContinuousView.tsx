import React from 'react';
import { Chapter, Page } from '../../types/api';
import { FitMode } from '../../hooks/useReaderFitMode';
import { getProxyImageUrl, getPageImageUrl } from '../../lib/utils';
import { ChapterBoundaryCard } from './ChapterBoundaryCard';
import { getFitModeClasses } from './readerUtils';

export interface ReaderContinuousViewProps {
  readingMode: string;
  pages: Page[];
  effectiveMangaId?: string;
  chapterId: string;
  chapterProviderId?: string;
  mangaUrl?: string;
  fitMode: FitMode;
  pageRefs: React.MutableRefObject<(HTMLDivElement | null)[]>;
  onToggleOverlays: () => void;
  hasPrevChapter?: boolean;
  hasNextChapter?: boolean;
  prevChapter?: Chapter;
  nextChapter?: Chapter;
  onPrevChapter?: () => void;
  onNextChapter?: () => void;
}

export const ReaderContinuousView: React.FC<ReaderContinuousViewProps> = ({
  readingMode,
  pages,
  effectiveMangaId,
  chapterId,
  chapterProviderId,
  mangaUrl,
  fitMode,
  pageRefs,
  onToggleOverlays,
  hasPrevChapter = false,
  hasNextChapter = false,
  prevChapter,
  nextChapter,
  onPrevChapter,
  onNextChapter,
}) => {
  const isLongstrip = readingMode === 'longstrip';

  return (
    <div
      data-testid="reader-content"
      className={
        isLongstrip
          ? 'flex flex-col gap-0 items-center w-full max-w-3xl mx-auto cursor-pointer'
          : 'flex flex-col gap-6 items-center w-full max-w-3xl mx-auto my-4 cursor-pointer'
      }
      onClick={onToggleOverlays}
    >
      {/* Previous chapter boundary card at top */}
      <div
        className={isLongstrip ? 'py-8' : 'py-4'}
        onClick={(e) => {
          if (onPrevChapter && hasPrevChapter) {
            e.stopPropagation();
            onPrevChapter();
          }
        }}
      >
        <ChapterBoundaryCard
          type="prev"
          hasChapter={hasPrevChapter}
          targetChapter={prevChapter}
          onNavigate={hasPrevChapter ? onPrevChapter : undefined}
        />
      </div>

      {pages.map((p) => {
        const originalIndex = p.index;
        const pageNum = originalIndex + 1;
        const imgSrc = getPageImageUrl(p, effectiveMangaId, chapterId, chapterProviderId, mangaUrl);

        if (isLongstrip) {
          return (
            <div
              key={p.index}
              ref={(el) => {
                pageRefs.current[originalIndex] = el;
              }}
              className="w-full text-center leading-none"
            >
              <img
                src={imgSrc}
                alt={`Page ${pageNum}`}
                className={getFitModeClasses(fitMode, 'block transition-opacity duration-300 min-h-[100px]')}
                loading="lazy"
                onError={(e) => {
                  const fallbackUrl = getProxyImageUrl(p.url, mangaUrl);
                  if (fallbackUrl && e.currentTarget.src !== fallbackUrl) {
                    e.currentTarget.src = fallbackUrl;
                  }
                }}
              />
            </div>
          );
        }

        return (
          <div
            key={p.index}
            ref={(el) => {
              pageRefs.current[originalIndex] = el;
            }}
            className="w-full text-center"
          >
            <img
              src={imgSrc}
              alt={`Page ${pageNum}`}
              className={getFitModeClasses(fitMode, 'rounded-sm shadow-md transition-opacity duration-300 min-h-[300px]')}
              loading="lazy"
              onError={(e) => {
                const fallbackUrl = getProxyImageUrl(p.url, mangaUrl);
                if (fallbackUrl && e.currentTarget.src !== fallbackUrl) {
                  e.currentTarget.src = fallbackUrl;
                }
              }}
            />
            <div className="mt-1 text-[10px] text-muted-foreground/70 font-mono">
              p.{pageNum}
            </div>
          </div>
        );
      })}

      {/* Next chapter boundary card at bottom */}
      <div
        className={isLongstrip ? 'py-12' : 'py-6'}
        onClick={(e) => {
          if (onNextChapter && hasNextChapter) {
            e.stopPropagation();
            onNextChapter();
          }
        }}
      >
        <ChapterBoundaryCard
          type="next"
          hasChapter={hasNextChapter}
          targetChapter={nextChapter}
          onNavigate={hasNextChapter ? onNextChapter : undefined}
        />
      </div>
    </div>
  );
};
