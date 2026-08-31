import React from 'react';
import { Link as LinkIcon } from 'lucide-react';
import { cn } from '../../../lib/utils';
import { ExternalLink } from '../../../types/api';
import { MultiChoice } from '../types';
import { mergeStringArrays } from '../../../utils/metadataCompare';

interface StringMultiChoiceProps {
  type: 'strings';
  title: string;
  currentItems: string[];
  incomingItems: string[];
  selectedMode: MultiChoice;
  onSelectMode: (mode: MultiChoice) => void;
}

interface ExternalLinkMultiChoiceProps {
  type: 'links';
  title?: string;
  currentLinks: ExternalLink[];
  incomingLinks: ExternalLink[];
  mergedLinks: ExternalLink[];
  selectedMode: MultiChoice;
  onSelectMode: (mode: MultiChoice) => void;
}

export type MetadataMultiChoiceFieldProps = StringMultiChoiceProps | ExternalLinkMultiChoiceProps;

export const MetadataMultiChoiceField: React.FC<MetadataMultiChoiceFieldProps> = (props) => {
  if (props.type === 'strings') {
    const { title, currentItems, incomingItems, selectedMode, onSelectMode } = props;
    const merged = mergeStringArrays(currentItems, incomingItems);

    return (
      <div className="space-y-2.5">
        <span className="text-xs font-semibold">{title}</span>
        <div className="space-y-2">
          <button
            type="button"
            onClick={() => onSelectMode('current')}
            className={cn(
              'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
              selectedMode === 'current'
                ? 'border-primary bg-primary/10 text-foreground font-medium'
                : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
            )}
          >
            <span className="text-[10px] uppercase font-bold text-muted-foreground">Current:</span>
            <span className="truncate max-w-[220px]">{currentItems.join(', ') || '—'}</span>
          </button>

          <button
            type="button"
            onClick={() => onSelectMode('incoming')}
            className={cn(
              'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
              selectedMode === 'incoming'
                ? 'border-primary bg-primary/10 text-foreground font-medium'
                : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
            )}
          >
            <span className="text-[10px] uppercase font-bold text-primary">Incoming:</span>
            <span className="truncate max-w-[220px]">{incomingItems.join(', ') || '—'}</span>
          </button>

          <button
            type="button"
            onClick={() => onSelectMode('merged')}
            className={cn(
              'flex w-full items-center justify-between rounded-xl border p-3 text-left text-xs transition-colors cursor-pointer',
              selectedMode === 'merged'
                ? 'border-primary bg-primary/10 text-foreground font-medium'
                : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
            )}
          >
            <span className="text-[10px] uppercase font-bold text-muted-foreground">Merged:</span>
            <span className="truncate max-w-[220px]">{merged.join(', ') || '—'}</span>
          </button>
        </div>
      </div>
    );
  }

  // type === 'links'
  const { currentLinks, incomingLinks, mergedLinks, selectedMode, onSelectMode } = props;

  const renderLinkPills = (links: ExternalLink[]) => {
    if (links.length === 0) {
      return <span className="text-xs text-muted-foreground">—</span>;
    }
    return links.map((link: ExternalLink, idx: number) => (
      <span
        key={`${link.url}-${idx}`}
        className="inline-flex items-center gap-1 text-[11px] bg-secondary/80 rounded px-1.5 py-0.5 max-w-full truncate"
      >
        <LinkIcon className="size-2.5 shrink-0" />
        <span className="truncate">{link.label || link.provider || link.url}</span>
      </span>
    ));
  };

  return (
    <div className="space-y-2.5">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold flex items-center gap-1.5">
          <LinkIcon className="size-4 text-muted-foreground" />
          External Links
        </span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <button
          type="button"
          onClick={() => onSelectMode('current')}
          className={cn(
            'flex flex-col gap-2 rounded-xl border p-3.5 text-left text-xs transition-colors cursor-pointer',
            selectedMode === 'current'
              ? 'border-primary bg-primary/10 text-foreground font-medium ring-1 ring-primary/40'
              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
          )}
        >
          <div className="flex items-center justify-between">
            <span className="text-[10px] uppercase font-bold text-muted-foreground">Current:</span>
            <span className="text-[10px] text-muted-foreground">({currentLinks.length})</span>
          </div>
          <div className="flex flex-wrap gap-1">
            {renderLinkPills(currentLinks)}
          </div>
        </button>

        <button
          type="button"
          onClick={() => onSelectMode('incoming')}
          className={cn(
            'flex flex-col gap-2 rounded-xl border p-3.5 text-left text-xs transition-colors cursor-pointer',
            selectedMode === 'incoming'
              ? 'border-primary bg-primary/10 text-foreground font-medium ring-1 ring-primary/40'
              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
          )}
        >
          <div className="flex items-center justify-between">
            <span className="text-[10px] uppercase font-bold text-primary">Incoming:</span>
            <span className="text-[10px] text-muted-foreground">({incomingLinks.length})</span>
          </div>
          <div className="flex flex-wrap gap-1">
            {renderLinkPills(incomingLinks)}
          </div>
        </button>

        <button
          type="button"
          onClick={() => onSelectMode('merged')}
          className={cn(
            'flex flex-col gap-2 rounded-xl border p-3.5 text-left text-xs transition-colors cursor-pointer',
            selectedMode === 'merged'
              ? 'border-primary bg-primary/10 text-foreground font-medium ring-1 ring-primary/40'
              : 'border-border bg-card text-muted-foreground hover:bg-muted/40'
          )}
        >
          <div className="flex items-center justify-between">
            <span className="text-[10px] uppercase font-bold text-muted-foreground">Merged:</span>
            <span className="text-[10px] text-muted-foreground">
              ({mergedLinks.length})
            </span>
          </div>
          <div className="flex flex-wrap gap-1">
            {renderLinkPills(mergedLinks)}
          </div>
        </button>
      </div>
    </div>
  );
};
