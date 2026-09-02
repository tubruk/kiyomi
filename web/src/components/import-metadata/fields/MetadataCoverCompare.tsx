import React from 'react';
import { Image as ImageIcon } from 'lucide-react';
import { cn, getProxyImageUrl } from '../../../lib/utils';
import { CoverImage } from '../../CoverImage';
import { Choice } from '../types';

interface MetadataCoverCompareProps {
  currentCoverUrl: string;
  currentCoverAssetUrl?: string;
  mangaUrl?: string;
  incomingCoverUrl: string;
  remoteMangaUrl?: string;
  selectedCover: Choice;
  onSelectCover: (choice: Choice) => void;
}

export const MetadataCoverCompare: React.FC<MetadataCoverCompareProps> = ({
  currentCoverUrl,
  currentCoverAssetUrl,
  mangaUrl,
  incomingCoverUrl,
  remoteMangaUrl,
  selectedCover,
  onSelectCover,
}) => {
  return (
    <div className="space-y-2.5">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold flex items-center gap-1.5">
          <ImageIcon className="size-4 text-muted-foreground" />
          Cover Image
        </span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <button
          type="button"
          onClick={() => onSelectCover('current')}
          className={cn(
            'flex items-center gap-4 rounded-xl border p-3.5 text-left transition-all cursor-pointer',
            selectedCover === 'current'
              ? 'border-primary bg-primary/5 ring-1 ring-primary'
              : 'border-border bg-card hover:bg-muted/40'
          )}
        >
          <CoverImage
            src={currentCoverAssetUrl || (currentCoverUrl ? getProxyImageUrl(currentCoverUrl, mangaUrl) : undefined)}
            alt=""
            icon="image"
            shape="auto"
            containerClassName="w-16 h-24 rounded-md shrink-0 shadow-xs"
            iconSize="size-6"
          />
          <div className="min-w-0 space-y-1">
            <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
              Keep Current
            </span>
            <p className="text-sm font-medium truncate">Current Cover</p>
          </div>
        </button>

        <button
          type="button"
          onClick={() => onSelectCover('incoming')}
          className={cn(
            'flex items-center gap-4 rounded-xl border p-3.5 text-left transition-all cursor-pointer',
            selectedCover === 'incoming'
              ? 'border-primary bg-primary/5 ring-1 ring-primary'
              : 'border-border bg-card hover:bg-muted/40'
          )}
        >
          <CoverImage
            src={incomingCoverUrl ? getProxyImageUrl(incomingCoverUrl, remoteMangaUrl) : undefined}
            alt=""
            icon="image"
            shape="auto"
            containerClassName="w-16 h-24 rounded-md shrink-0 shadow-xs"
            iconSize="size-6"
          />
          <div className="min-w-0 space-y-1">
            <span className="text-[10px] font-bold uppercase tracking-wider text-primary">
              Use Incoming
            </span>
            <p className="text-sm font-medium truncate">Incoming Cover</p>
          </div>
        </button>
      </div>
    </div>
  );
};
