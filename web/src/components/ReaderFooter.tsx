import React from 'react';
import { ArrowUp, Palette } from 'lucide-react';
import { Button } from './ui/button';
import { Slider } from './ui/slider';

export type ReaderTheme = 'oled' | 'dark' | 'sepia' | 'light';

interface ReaderFooterProps {
  currentPage: number;
  totalPages: number;
  readingMode?: string;
  readerTheme: ReaderTheme;
  onThemeChange: (theme: ReaderTheme) => void;
  onPageChange: (page: number) => void;
  onScrollTop: () => void;
}

export const ReaderFooter: React.FC<ReaderFooterProps> = ({
  currentPage,
  totalPages,
  readingMode,
  readerTheme,
  onThemeChange,
  onPageChange,
  onScrollTop,
}) => {
  const isRTL = readingMode === 'rtl';

  return (
    <footer className="fixed bottom-0 left-0 right-0 z-50 border-t border-border bg-background/90 backdrop-blur-md px-4 py-2.5">
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
                onPageChange(val[0]);
              } else if (typeof val === 'number') {
                onPageChange(val);
              }
            }}
            className="cursor-pointer"
          />
        </div>

        {/* Background Canvas Theme Selector */}
        <div className="hidden sm:flex items-center gap-1">
          <Palette className="size-3.5 text-muted-foreground mr-1" aria-hidden />
          <button
            type="button"
            title="OLED Black Theme"
            onClick={() => onThemeChange('oled')}
            className={`size-5 rounded-full border border-border bg-black transition-transform cursor-pointer ${
              readerTheme === 'oled' ? 'scale-125 ring-2 ring-primary' : 'hover:scale-110'
            }`}
          />
          <button
            type="button"
            title="Dark Charcoal Theme"
            onClick={() => onThemeChange('dark')}
            className={`size-5 rounded-full border border-border bg-zinc-900 transition-transform cursor-pointer ${
              readerTheme === 'dark' ? 'scale-125 ring-2 ring-primary' : 'hover:scale-110'
            }`}
          />
          <button
            type="button"
            title="Sepia Paper Theme"
            onClick={() => onThemeChange('sepia')}
            className={`size-5 rounded-full border border-border bg-[#e8e0d0] transition-transform cursor-pointer ${
              readerTheme === 'sepia' ? 'scale-125 ring-2 ring-primary' : 'hover:scale-110'
            }`}
          />
          <button
            type="button"
            title="White Light Theme"
            onClick={() => onThemeChange('light')}
            className={`size-5 rounded-full border border-border bg-white transition-transform cursor-pointer ${
              readerTheme === 'light' ? 'scale-125 ring-2 ring-primary' : 'hover:scale-110'
            }`}
          />
        </div>

        {/* Scroll to Top */}
        <Button
          variant="outline"
          size="icon"
          className="h-8 w-8 cursor-pointer"
          onClick={onScrollTop}
          title="Scroll to Top"
          aria-label="Scroll to Top"
        >
          <ArrowUp className="size-4" aria-hidden />
        </Button>
      </div>
    </footer>
  );
};
