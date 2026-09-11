import React, { useState } from 'react';
import { Link } from '@tanstack/react-router';
import {
  ArrowLeft,
  BookOpen,
  Plus,
  Play,
  MoreVertical,
  Download,
  Edit3,
  Trash2,
  GitMerge,
} from 'lucide-react';
import { Button } from '../ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import { MergeMangaDialog } from '@/components/merge-manga/MergeMangaDialog';
import { Manga } from '../../types/api';

export interface DetailsActionBarProps {
  isRemoteRoute: boolean;
  providerIdParam?: string;
  isInLibrary: boolean;
  targetMangaId?: string;
  manga?: Manga;
  hasZeroChapters?: boolean;
  isUnavailable?: boolean;
  hasChapters?: boolean;
  readingCta?: { label: string; chapterId: string; page: number } | null;
  isAddingToLibrary?: boolean;
  onAddToLibrary?: () => void;
  onPrimaryCta?: () => void;
  onOpenImportMetadata?: () => void;
  onOpenEditMetadata?: () => void;
  onRemoveFromLibrary?: () => void;
}

export const DetailsActionBar: React.FC<DetailsActionBarProps> = ({
  isRemoteRoute,
  providerIdParam,
  isInLibrary,
  targetMangaId,
  manga,
  hasZeroChapters = false,
  isUnavailable = false,
  hasChapters = false,
  readingCta,
  isAddingToLibrary = false,
  onAddToLibrary,
  onPrimaryCta,
  onOpenImportMetadata,
  onOpenEditMetadata,
  onRemoveFromLibrary,
}) => {
  const [isMergeOpen, setIsMergeOpen] = useState(false);
  const keepMangaId = targetMangaId || manga?.id || '';

  return (
    <div className="flex items-center justify-between gap-4">
      {/* Back link */}
      {isRemoteRoute ? (
        <Link
          to="/providers/$providerId"
          params={{ providerId: providerIdParam || 'mangafox' }}
          className="inline-flex items-center gap-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4" aria-hidden />
          Back to Explore
        </Link>
      ) : (
        <Link
          to="/"
          className="inline-flex items-center gap-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4" aria-hidden />
          Back to Library
        </Link>
      )}

      {/* Action Buttons */}
      <div className="flex flex-wrap items-center gap-2">
        {isRemoteRoute ? (
          isInLibrary ? (
            <Link
              to="/manga/$mangaId"
              params={{ mangaId: targetMangaId || '' }}
              className="inline-flex h-9 items-center gap-2 rounded-md border border-emerald-500/40 bg-emerald-500/10 px-3.5 text-xs font-semibold text-emerald-500 hover:bg-emerald-500/20 transition-colors"
              title="Series is saved in your local library. Click to view library entry."
            >
              <BookOpen className="size-4" aria-hidden />
              View in Library
            </Link>
          ) : (
            <Button
              variant="default"
              onClick={onAddToLibrary}
              disabled={isAddingToLibrary || !manga || hasZeroChapters || isUnavailable}
              className="gap-2 cursor-pointer"
            >
              <Plus className="size-4" aria-hidden />
              {isAddingToLibrary ? 'Adding...' : 'Add to Library'}
            </Button>
          )
        ) : (
          <>
            {manga && !isUnavailable && hasChapters && readingCta && onPrimaryCta && (
              <Button
                variant="default"
                onClick={onPrimaryCta}
                className="font-semibold cursor-pointer gap-2"
              >
                <Play className="size-4 fill-current" aria-hidden />
                {readingCta.label}
              </Button>
            )}

            {isInLibrary && manga && onRemoveFromLibrary && (
              <DropdownMenu>
                <DropdownMenuTrigger
                  className="inline-flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-card text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer"
                  aria-label="Metadata options"
                  title="Metadata options"
                >
                  <MoreVertical className="size-4" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                  {onOpenImportMetadata && (
                    <DropdownMenuItem
                      onClick={onOpenImportMetadata}
                      className="text-xs cursor-pointer gap-2"
                    >
                      <Download className="size-4" />
                      Import metadata
                    </DropdownMenuItem>
                  )}
                  {onOpenEditMetadata && (
                    <DropdownMenuItem
                      onClick={onOpenEditMetadata}
                      className="text-xs cursor-pointer gap-2"
                    >
                      <Edit3 className="size-4" />
                      Edit Metadata
                    </DropdownMenuItem>
                  )}
                  {!isRemoteRoute && keepMangaId && (
                    <DropdownMenuItem
                      onClick={() => setIsMergeOpen(true)}
                      disabled={isAddingToLibrary}
                      className="text-xs cursor-pointer gap-2"
                    >
                      <GitMerge className="size-4" />
                      Merge with...
                    </DropdownMenuItem>
                  )}
                  <DropdownMenuItem
                    onClick={onRemoveFromLibrary}
                    className="text-destructive focus:text-destructive focus:bg-destructive/10 text-xs cursor-pointer gap-2"
                  >
                    <Trash2 className="size-4" />
                    Remove from Library
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </>
        )}
      </div>

      <MergeMangaDialog
        open={isMergeOpen}
        onOpenChange={setIsMergeOpen}
        keepMangaId={keepMangaId}
      />
    </div>
  );
};
