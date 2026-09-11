import React, { useEffect, useMemo, useState } from 'react';
import { Search } from 'lucide-react';
import { Manga } from '../../types/api';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { CoverImage } from '../CoverImage';
import { cn } from '../../lib/utils';
import { useMangaDetails } from '../../api/hooks';
import { ImportMetadataDialog } from '../import-metadata/ImportMetadataDialog';
import { useMergeFlow } from './hooks/useMergeFlow';

interface MergeMangaDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  keepMangaId: string;
}

type DialogPhase = 'pick' | 'compare';

const SourcesStep: React.FC<{
  availableSources: Manga[];
  sourceMangaId: string | null;
  setSourceMangaId: (id: string | null) => void;
  keepMangaTitle: string;
  keepMangaCover: string;
}> = ({
  availableSources,
  sourceMangaId,
  setSourceMangaId,
  keepMangaTitle,
  keepMangaCover,
}) => {
  const [query, setQuery] = useState('');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return availableSources;
    return availableSources.filter((m) => (m.title || '').toLowerCase().includes(q));
  }, [availableSources, query]);

  return (
    <div className="space-y-3">
      <DialogHeader>
        <DialogTitle>Merge with other manga</DialogTitle>
        <DialogDescription>
          Pick one library entry to merge into{' '}
          <span className="font-semibold text-foreground">{keepMangaTitle}</span>. The selected
          manga will be removed after the merge.
        </DialogDescription>
      </DialogHeader>

      <div className="flex items-center gap-2 rounded-lg border border-border bg-card/40 p-2">
        <CoverImage
          src={keepMangaCover}
          alt=""
          shape="auto"
          containerClassName="size-10 rounded shrink-0"
          iconSize="size-4"
        />
        <div className="min-w-0 flex-1">
          <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Keep</p>
          <p className="text-sm font-semibold truncate">{keepMangaTitle}</p>
        </div>
      </div>

      <div className="relative">
        <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search library..."
          className="h-8 pl-8 text-xs"
        />
      </div>

      <div
        className="max-h-72 overflow-y-auto rounded-lg border border-border divide-y divide-border"
        role="radiogroup"
        aria-label="Source manga"
      >
        {filtered.length === 0 ? (
          <p className="py-6 text-center text-xs text-muted-foreground">
            No library entries to merge.
          </p>
        ) : (
          filtered.map((manga) => {
            const isSelected = sourceMangaId === manga.id;
            return (
              <label
                key={manga.id}
                className={cn(
                  'flex cursor-pointer items-center gap-3 p-2 transition-colors hover:bg-muted/40',
                  isSelected && 'bg-primary/5'
                )}
              >
                <input
                  type="radio"
                  name="merge-source"
                  className="size-4 accent-primary"
                  checked={isSelected}
                  onChange={() => setSourceMangaId(manga.id)}
                  aria-label={`Select ${manga.title}`}
                />
                <CoverImage
                  src={manga.coverUrl || manga.cover || manga.metadata.cover_url}
                  alt=""
                  shape="auto"
                  containerClassName="size-10 rounded shrink-0"
                  iconSize="size-4"
                />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{manga.title}</p>
                  <p className="truncate text-[11px] text-muted-foreground">
                    {manga.id}
                  </p>
                </div>
              </label>
            );
          })
        )}
      </div>
    </div>
  );
};

export const MergeMangaDialog: React.FC<MergeMangaDialogProps> = ({
  open,
  onOpenChange,
  keepMangaId,
}) => {
  const flow = useMergeFlow({ keepMangaId });
  const [phaseState, setPhaseState] = useState<DialogPhase>('pick');
  const keepDetails = useMangaDetails(keepMangaId, { enabled: Boolean(keepMangaId) });
  const keepManga = keepDetails.data;

  // Reset local state when the dialog closes so reopens start on the picker.
  useEffect(() => {
    if (!open) {
      setPhaseState('pick');
      flow.setSourceMangaId(null);
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  const keepMangaTitle = keepManga?.title ?? 'this manga';
  const keepMangaCover =
    keepManga?.coverUrl || keepManga?.cover || keepManga?.metadata?.cover_url || '';

  const handleClose = () => {
    setPhaseState('pick');
    flow.setSourceMangaId(null);
    onOpenChange(false);
  };

  const handleCompare = () => {
    if (!flow.sourceMangaId) return;
    setPhaseState('compare');
  };

  // Once the user confirms a source pick, swap the body to the embedded
  // ImportMetadataDialog (mode='merge') which handles the comparison step.
  if (phaseState === 'compare' && flow.sourceManga && keepManga) {
    return (
      <ImportMetadataDialog
        manga={keepManga}
        sources={[]}
        open={open}
        onOpenChange={handleClose}
        mode="merge"
        incomingManga={flow.sourceManga}
        sourceMangaIds={[flow.sourceManga.id]}
      />
    );
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-2xl">
        <SourcesStep
          availableSources={flow.availableSources}
          sourceMangaId={flow.sourceMangaId}
          setSourceMangaId={flow.setSourceMangaId}
          keepMangaTitle={keepMangaTitle}
          keepMangaCover={keepMangaCover}
        />
        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleClose}
            className="cursor-pointer"
          >
            Cancel
          </Button>
          <Button
            onClick={handleCompare}
            disabled={!flow.sourceMangaId}
            className="cursor-pointer"
          >
            Compare
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};