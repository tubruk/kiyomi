import React from 'react';
import { BookOpen, CheckCircle2 } from 'lucide-react';
import { Chapter } from '../../types/api';

export interface ChapterBoundaryCardProps {
  type: 'prev' | 'next';
  hasChapter: boolean;
  targetChapter?: Chapter;
  onNavigate?: () => void;
  className?: string;
}

export const ChapterBoundaryCard: React.FC<ChapterBoundaryCardProps> = ({
  type,
  hasChapter,
  targetChapter,
  onNavigate,
  className = '',
}) => {
  const isPrev = type === 'prev';
  const chapterIdent = targetChapter
    ? targetChapter.number !== undefined && targetChapter.number !== null
      ? targetChapter.number
      : targetChapter.name || targetChapter.title || ''
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
      className={`flex flex-col items-center justify-center p-8 mx-4 max-w-sm w-full rounded-2xl border border-border/70 bg-card/90 backdrop-blur-lg shadow-xl text-center select-none ${
        onNavigate ? 'cursor-pointer hover:border-primary/50 transition-colors' : ''
      } ${className}`.trim()}
      onClick={onNavigate}
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
