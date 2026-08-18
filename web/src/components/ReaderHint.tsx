import React, { useEffect } from 'react';
import { ArrowLeft, ArrowRight, ArrowUp, ArrowDown } from 'lucide-react';
import type { ReadingHint } from '../hooks/useReadingHint';

interface ReaderHintProps {
  hint: ReadingHint | null;
  onDismiss: () => void;
}

const MODE_LABELS: Record<string, string> = {
  rtl: 'Right-to-Left',
  ltr: 'Left-to-Right',
  vertical: 'Vertical Scroll',
  longstrip: 'Long Strip',
};

export const ReaderHint: React.FC<ReaderHintProps> = ({ hint, onDismiss }) => {
  useEffect(() => {
    if (!hint?.visible) return;
    const timer = setTimeout(() => {
      onDismiss();
    }, 2000);
    return () => clearTimeout(timer);
  }, [hint?.visible, onDismiss]);

  if (!hint?.visible) return null;

  const modeLabel = MODE_LABELS[hint.readingMode] || hint.readingMode;
  const isPaged = hint.readingMode === 'rtl' || hint.readingMode === 'ltr';

  const isRTL = hint.readingMode === 'rtl';
  const isVertical = hint.readingMode === 'vertical';
  const isLongStrip = hint.readingMode === 'longstrip';

  // For paged: next direction
  const nextIsLeft = isRTL; // RTL: next page is left, LTR: next page is right
  const nextIcon = nextIsLeft ? <ArrowLeft className="size-12 text-white" strokeWidth={2.5} /> : <ArrowRight className="size-12 text-white" strokeWidth={2.5} />;
  const prevIcon = nextIsLeft ? <ArrowRight className="size-5 text-white/40" /> : <ArrowLeft className="size-5 text-white/40" />;

  return (
    <div
      className="fixed inset-0 z-[100] flex flex-col items-center justify-center gap-8 bg-black/80 backdrop-blur-md cursor-pointer"
      onClick={onDismiss}
    >
      {/* Mode label */}
      <div className="text-center">
        <span className="text-xs font-medium uppercase tracking-widest text-white/40">
          Reading Mode
        </span>
        <p className="mt-0.5 text-base font-semibold text-white">
          {modeLabel}
        </p>
      </div>

      {/* Direction visual */}
      {isPaged ? (
        <div className="flex h-20 w-72 items-center justify-center gap-6">
          {/* RTL: prev on right. LTR: prev on left. */}
          {!isRTL && (
            <div className="flex flex-col items-center gap-1 opacity-40">
              {prevIcon}
              <span className="text-[10px] text-white/60">Prev</span>
            </div>
          )}

          {/* Big next arrow */}
          <div className="flex flex-col items-center gap-2">
            {nextIcon}
            <span className="text-xs font-medium text-white/70">Next</span>
          </div>

          {isRTL && (
            <div className="flex flex-col items-center gap-1 opacity-40">
              {prevIcon}
              <span className="text-[10px] text-white/60">Prev</span>
            </div>
          )}
        </div>
      ) : isVertical ? (
        <div className="flex flex-col items-center gap-6">
          {/* Prev — subtle, scroll up */}
          <div className="flex flex-col items-center gap-1 opacity-40">
            <ArrowUp className="size-5 text-white/40" />
            <span className="text-[10px] text-white/60">Prev</span>
          </div>
          {/* Next — big, scroll down */}
          <div className="flex flex-col items-center gap-2">
            <ArrowDown className="size-12 text-white" strokeWidth={2.5} />
            <span className="text-xs font-medium text-white/70">Next</span>
          </div>
        </div>
      ) : isLongStrip ? (
        <div className="flex flex-col items-center gap-6">
          {/* Prev — subtle, scroll up */}
          <div className="flex flex-col items-center gap-1 opacity-40">
            <ArrowUp className="size-5 text-white/40" />
            <span className="text-[10px] text-white/60">Prev</span>
          </div>
          {/* Next — big, scroll down */}
          <div className="flex flex-col items-center gap-2">
            <ArrowDown className="size-12 text-white" strokeWidth={2.5} />
            <span className="text-xs font-medium text-white/70">Next</span>
          </div>
        </div>
      ) : null}

      {/* Instruction text */}
      <p className="text-xs text-white/40">{hint.message}</p>
    </div>
  );
};
