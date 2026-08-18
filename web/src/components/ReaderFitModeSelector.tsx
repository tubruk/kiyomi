import React, { useState, useRef, useEffect } from 'react';
import { SlidersHorizontal, Maximize2, ArrowUpDown, Image } from 'lucide-react';
import { FitMode } from '../hooks/useReaderFitMode';

interface ReaderFitModeSelectorProps {
  fitMode: FitMode;
  readingMode: string;
  onFitModeChange: (mode: FitMode) => void;
  onReadingModeChange: (mode: string) => void;
}

const READING_MODES = ['rtl', 'ltr', 'vertical', 'longstrip'] as const;
const READING_MODE_LABELS: Record<string, string> = {
  rtl: 'Right-to-Left',
  ltr: 'Left-to-Right',
  vertical: 'Vertical',
  longstrip: 'Long Strip',
};

const FIT_MODE_OPTIONS: { value: FitMode; label: string; icon: React.ReactNode }[] = [
  { value: 'fit-width', label: 'Fit Width', icon: <Maximize2 className="size-4" aria-hidden /> },
  { value: 'fit-height', label: 'Fit Height', icon: <ArrowUpDown className="size-4" aria-hidden /> },
  { value: 'fit-original', label: 'Original', icon: <Image className="size-4" aria-hidden /> },
];

export const ReaderFitModeSelector: React.FC<ReaderFitModeSelectorProps> = ({
  fitMode,
  readingMode,
  onFitModeChange,
  onReadingModeChange,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  return (
    <div className="relative" ref={menuRef}>
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex items-center gap-1.5 px-2 py-1.5 rounded-md text-xs font-medium text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
        title="Reader Settings"
        aria-label="Reader Settings"
        aria-expanded={isOpen}
        aria-haspopup="menu"
      >
        <SlidersHorizontal className="size-4" />
      </button>

      {isOpen && (
        <div
          role="menu"
          className="absolute bottom-full mb-1 right-0 min-w-[160px] rounded-md border border-border bg-background/95 backdrop-blur-md shadow-lg z-50"
        >
          {/* Fit Mode section */}
          <div className="px-3 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
              Fit Mode
            </p>
            {FIT_MODE_OPTIONS.map((option) => (
              <button
                key={option.value}
                type="button"
                role="menuitem"
                onClick={() => {
                  onFitModeChange(option.value);
                }}
                className={`w-full flex items-center gap-2 px-2 py-1.5 text-xs rounded cursor-pointer transition-colors ${
                  fitMode === option.value
                    ? 'text-foreground bg-muted/50'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/30'
                }`}
              >
                {option.icon}
                <span>{option.label}</span>
              </button>
            ))}
          </div>

          <div className="h-px bg-border mx-2" />

          {/* Reading Direction section */}
          <div className="px-3 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
              Direction
            </p>
            {READING_MODES.map((mode) => (
              <button
                key={mode}
                type="button"
                role="menuitem"
                onClick={() => {
                  onReadingModeChange(mode);
                }}
                className={`w-full flex items-center gap-2 px-2 py-1.5 text-xs rounded cursor-pointer transition-colors ${
                  readingMode === mode
                    ? 'text-foreground bg-muted/50'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted/30'
                }`}
              >
                <span className="w-5 text-center font-mono text-[10px]">
                  {mode === 'rtl' ? 'RTL' : mode === 'ltr' ? 'LTR' : mode === 'vertical' ? 'V' : 'LS'}
                </span>
                <span>{READING_MODE_LABELS[mode]}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
