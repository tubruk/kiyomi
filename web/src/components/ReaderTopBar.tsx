import React from 'react';
import { useNavigate } from '@tanstack/react-router';
import { ChevronLeft, ChevronRight, X, List } from 'lucide-react';
import { Button } from './ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from './ui/sheet';
import { Chapter } from '../types/api';
import { formatChapterTitleWithPage } from '../lib/utils';

interface ReaderTopBarProps {
  mangaTitle?: string;
  chapterTitle?: string;
  currentPage?: number;
  totalPages?: number;
  chapterNumber?: number;
  mangaId?: string;
  providerId?: string;
  remoteId?: string;
  chapters?: Chapter[];
  currentChapterId?: string;
  hasPrevChapter: boolean;
  hasNextChapter: boolean;
  onPrevChapter: () => void;
  onNextChapter: () => void;
  onSelectChapter: (chapterId: string) => void;
}

export const ReaderTopBar: React.FC<ReaderTopBarProps> = ({
  mangaTitle = 'Manga',
  chapterTitle = 'Chapter',
  currentPage,
  totalPages,
  chapterNumber,
  mangaId,
  providerId,
  remoteId,
  chapters = [],
  currentChapterId,
  hasPrevChapter,
  hasNextChapter,
  onPrevChapter,
  onNextChapter,
  onSelectChapter,
}) => {
  const navigate = useNavigate();

  const formattedChapterTitle = formatChapterTitleWithPage(
    chapterTitle,
    currentPage,
    totalPages,
    chapterNumber
  );


  const handleClose = () => {
    if (mangaId) {
      navigate({ to: '/manga/$mangaId', params: { mangaId } });
    } else if (providerId && remoteId) {
      navigate({ to: '/providers/$providerId/manga/$remoteId', params: { providerId, remoteId } });
    } else {
      navigate({ to: '/' });
    }
  };

  return (
    <header className="sticky top-0 z-50 w-full border-b border-border bg-background/90 backdrop-blur-md">
      <div className="mx-auto flex h-14 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6">
        {/* Back / Close button */}
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClose}
            className="gap-1.5 text-xs cursor-pointer"
          >
            <X className="size-4" aria-hidden />
            <span className="hidden sm:inline">Close</span>
          </Button>

          {/* Quick Chapter Drawer Trigger */}
          {chapters.length > 0 && (
            <Sheet>
              <SheetTrigger
                render={
                  <Button variant="outline" size="sm" className="gap-1.5 text-xs cursor-pointer">
                    <List className="size-4" aria-hidden />
                    <span className="hidden md:inline">Chapters</span>
                  </Button>
                }
              />
              <SheetContent side="left" className="w-[300px] sm:w-[380px]">
                <SheetHeader>
                  <SheetTitle className="text-left text-base font-bold">Chapters ({chapters.length})</SheetTitle>
                </SheetHeader>
                <div className="mt-4 flex flex-col gap-1.5 max-h-[calc(100vh-100px)] overflow-y-auto pr-1">
                  {chapters.map((c) => {
                    const isCurrent = c.id === currentChapterId;
                    const dateVal = c.uploadDate || c.uploadedAt;
                    return (
                      <button
                        key={c.id}
                        type="button"
                        onClick={() => onSelectChapter(c.id)}
                        className={`flex items-center justify-between rounded-md px-3 py-2 text-left text-xs transition-colors cursor-pointer ${
                          isCurrent
                            ? 'bg-primary text-primary-foreground font-semibold'
                            : 'hover:bg-accent text-foreground'
                        }`}
                      >
                        <span className="truncate">{c.title || c.name || `Chapter ${c.number}`}</span>
                        {dateVal && (
                          <span className="text-[10px] opacity-70 ml-2 font-mono">
                            {new Date(dateVal).toLocaleDateString()}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
              </SheetContent>
            </Sheet>
          )}
        </div>

        {/* Title & Info Center */}
        <div className="flex flex-1 items-center justify-center gap-2 overflow-hidden text-center text-xs sm:text-sm">
          <span className="truncate font-semibold text-foreground">{mangaTitle}</span>
          <span className="text-muted-foreground hidden sm:inline">•</span>
          <span className="truncate text-muted-foreground hidden sm:inline">{formattedChapterTitle}</span>
        </div>

        {/* Controls Right: Prev / Next Chapter Buttons */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!hasPrevChapter}
            onClick={onPrevChapter}
            title="Previous Chapter"
            className="h-8 px-2 sm:px-3 text-xs cursor-pointer"
          >
            <ChevronLeft className="size-4" aria-hidden />
            <span className="hidden md:inline">Prev</span>
          </Button>

          <Button
            variant="outline"
            size="sm"
            disabled={!hasNextChapter}
            onClick={onNextChapter}
            title="Next Chapter"
            className="h-8 px-2 sm:px-3 text-xs cursor-pointer"
          >
            <span className="hidden md:inline">Next</span>
            <ChevronRight className="size-4" aria-hidden />
          </Button>
        </div>
      </div>
    </header>
  );
};
