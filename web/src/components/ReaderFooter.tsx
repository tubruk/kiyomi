import React from 'react';
import { ChevronsLeft, ChevronsRight, ChevronsUp } from 'lucide-react';
import { Button } from './ui/button';
import { Slider } from './ui/slider';
import { ReaderFitModeSelector } from './ReaderFitModeSelector';
import { FitMode } from '../hooks/useReaderFitMode';

interface ReaderFooterProps {
  currentPage: number;
  totalPages: number;
  readingMode: string;
  fitMode: FitMode;
  onPageChange: (page: number) => void;
  onScrollTop: () => void;
  onFitModeChange: (mode: FitMode) => void;
  onReadingModeChange: (mode: string) => void;
}

export const ReaderFooter: React.FC<ReaderFooterProps> = ({
  currentPage,
  totalPages,
  readingMode,
  fitMode,
  onPageChange,
  onScrollTop,
  onFitModeChange,
  onReadingModeChange,
}) => {
  const isRTL = readingMode === 'rtl';
  const isVertical = readingMode === 'vertical' || readingMode === 'longstrip';

  const scrollIcon = isRTL ? (
    <ChevronsRight className="size-4" aria-hidden />
  ) : isVertical ? (
    <ChevronsUp className="size-4" aria-hidden />
  ) : (
    <ChevronsLeft className="size-4" aria-hidden />
  );

  const scrollLabel = isRTL ? 'Scroll to End' : 'Scroll to Top';

  return (
    <footer className="border-t border-border bg-background/90 backdrop-blur-md px-4 py-2.5">
      <div className="mx-auto flex max-w-4xl items-center gap-4">
        {/* Page Counter */}
        <span data-testid="page-indicator" className="min-w-[70px] text-xs font-semibold text-muted-foreground">
          {currentPage} / {totalPages}
        </span>

        {/* Interactive Page Slider */}
        <div className="flex-1" dir={isRTL ? 'rtl' : undefined}>
          <Slider
            value={[currentPage]}
            min={1}
            max={totalPages || 1}
            step={1}
            dir={isRTL ? 'rtl' : undefined}
            onValueChange={(val) => {
              if (Array.isArray(val) && typeof val[0] === 'number') {
                // In RTL, slider is visually mirrored — invert the value so
                // dragging "left" (thumb visually left) = higher page
                const page = isRTL ? totalPages - val[0] + 1 : val[0];
                onPageChange(page);
              } else if (typeof val === 'number') {
                const page = isRTL ? totalPages - val + 1 : val;
                onPageChange(page);
              }
            }}
            className="cursor-pointer"
          />
        </div>

        {/* Reader Settings (fit mode + direction) */}
        <ReaderFitModeSelector
          fitMode={fitMode}
          readingMode={readingMode}
          onFitModeChange={onFitModeChange}
          onReadingModeChange={onReadingModeChange}
        />

        {/* Jump to Start / End */}
        <Button
          variant="outline"
          size="icon"
          className="h-8 w-8 cursor-pointer"
          onClick={onScrollTop}
          title={scrollLabel}
          aria-label={scrollLabel}
        >
          {scrollIcon}
        </Button>
      </div>
    </footer>
  );
};
