import React, { useState, useEffect } from 'react';
import { Star, Heart } from 'lucide-react';
import { Manga, UserStatus } from '../../types/api';
import { Button } from '../ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { cn } from '../../lib/utils';

export const STATUS_OPTIONS: { value: UserStatus; label: string }[] = [
  { value: 'reading', label: 'Reading' },
  { value: 'plan_to_read', label: 'Plan to Read' },
  { value: 'completed', label: 'Completed' },
  { value: 'on_hold', label: 'On Hold' },
  { value: 'dropped', label: 'Dropped' },
  { value: 'unread', label: 'Unread' },
];

export const formatStatus = (status?: string) => {
  if (!status) return 'Unread';
  const found = STATUS_OPTIONS.find((s) => s.value === status);
  if (found) return found.label;
  return status
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
};

export interface DetailsUserMetadataProps {
  manga: Manga;
  isUpdating?: boolean;
  onUpdate: (fields: Partial<Manga>) => void;
}

export const DetailsUserMetadata: React.FC<DetailsUserMetadataProps> = ({
  manga,
  isUpdating = false,
  onUpdate,
}) => {
  const [localUserNotes, setLocalUserNotes] = useState('');
  const [isEditingNotes, setIsEditingNotes] = useState(false);
  const [hoverRating, setHoverRating] = useState<number | null>(null);

  const isFavorite = Boolean(
    manga.userFavorite || manga.user_favorite || manga.meta?.user_favorite
  );
  const userStatus = manga.userStatus || manga.meta?.user_status || 'reading';
  const userRating = manga.userRating || manga.meta?.user_rating || 0;
  const userNotes = manga.userNotes || manga.meta?.user_notes || '';

  useEffect(() => {
    if (!isEditingNotes) {
      setLocalUserNotes(userNotes);
    }
  }, [userNotes, isEditingNotes]);

  return (
    <div
      className={cn(
        'flex flex-col gap-3 p-3.5 rounded-lg border border-border bg-card shadow-xs transition-opacity duration-200',
        isUpdating && 'opacity-60 pointer-events-none'
      )}
    >
      {/* Reading Status Dropdown */}
      <div className="flex flex-col gap-1">
        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
          Reading Status
        </span>
        <Select
          value={userStatus}
          onValueChange={(val) => onUpdate({ user_status: (val || undefined) as UserStatus })}
          disabled={isUpdating}
        >
          <SelectTrigger className="h-9 w-full text-xs bg-background border-border">
            <SelectValue placeholder="Select Status">
              {formatStatus(userStatus)}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {STATUS_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value} className="text-xs">
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Rating and Favorite row */}
      <div className="flex items-center justify-between border-t border-border/40 pt-3 mt-0.5">
        {/* 5-star rating with score indicator */}
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-1.5">
            <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
              Rating
            </span>
            <span className="text-[10px] font-semibold text-foreground/80">
              {(hoverRating !== null ? hoverRating : userRating) > 0
                ? `${hoverRating !== null ? hoverRating : userRating}/10`
                : 'Unrated'}
            </span>
            {userRating > 0 && (
              <button
                type="button"
                onClick={() => onUpdate({ user_rating: 0 })}
                disabled={isUpdating}
                className="text-[10px] text-muted-foreground hover:text-destructive cursor-pointer"
                title="Clear Rating"
                aria-label="Clear Rating"
              >
                ×
              </button>
            )}
          </div>
          <div
            className="flex items-center gap-0.5"
            onMouseLeave={() => setHoverRating(null)}
          >
            {Array.from({ length: 5 }).map((_, idx) => {
              const starValue = (idx + 1) * 2;
              const activeValue = hoverRating !== null ? hoverRating : userRating;
              const isFilled = activeValue >= starValue;
              return (
                <button
                  key={idx}
                  type="button"
                  disabled={isUpdating}
                  onMouseEnter={() => setHoverRating(starValue)}
                  onClick={() => {
                    const newRating = userRating === starValue ? 0 : starValue;
                    onUpdate({ user_rating: newRating });
                  }}
                  className="text-amber-400 hover:scale-110 active:scale-95 transition-transform focus:outline-hidden disabled:opacity-50 cursor-pointer"
                  title={`Rate ${starValue}/10 (${idx + 1} Stars)`}
                  aria-label={`Rate ${starValue}/10 (${idx + 1} Stars)`}
                >
                  <Star
                    className={cn(
                      'size-4',
                      isFilled ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground/30'
                    )}
                  />
                </button>
              );
            })}
          </div>
        </div>

        {/* Love / Favorite button */}
        <div className="flex flex-col gap-1 items-end">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Favorite
          </span>
          <button
            type="button"
            disabled={isUpdating}
            onClick={() => onUpdate({ user_favorite: !isFavorite })}
            className={cn(
              'flex items-center justify-center size-8 rounded-full border transition-all active:scale-95 hover:bg-muted/50 cursor-pointer disabled:opacity-50',
              isFavorite
                ? 'bg-rose-500/10 border-rose-500/30 text-rose-500 hover:bg-rose-500/20'
                : 'bg-background border-border text-muted-foreground hover:text-foreground'
            )}
            title={isFavorite ? 'Remove from Favorites' : 'Add to Favorites'}
            aria-label={isFavorite ? 'Remove from Favorites' : 'Add to Favorites'}
          >
            <Heart className="size-4" fill={isFavorite ? 'currentColor' : 'none'} />
          </button>
        </div>
      </div>

      {/* Personal Notes */}
      <div className="flex flex-col gap-1.5 border-t border-border/40 pt-3 mt-0.5">
        <div className="flex items-center justify-between">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Personal Notes
          </span>
          {!isEditingNotes && (
            <button
              type="button"
              onClick={() => setIsEditingNotes(true)}
              className="text-[10px] font-semibold text-primary hover:underline cursor-pointer"
            >
              {userNotes ? 'Edit' : '+ Add Note'}
            </button>
          )}
        </div>

        {isEditingNotes ? (
          <div className="flex flex-col gap-2">
            <textarea
              value={localUserNotes}
              onChange={(e) => setLocalUserNotes(e.target.value)}
              disabled={isUpdating}
              placeholder="Add private personal notes..."
              autoFocus
              className="w-full min-h-16 h-20 max-h-32 rounded-md border border-border bg-background px-3 py-2 text-xs placeholder:text-muted-foreground focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-ring disabled:opacity-50"
            />
            <div className="flex items-center justify-end gap-1.5">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  setLocalUserNotes(userNotes);
                  setIsEditingNotes(false);
                }}
                disabled={isUpdating}
                className="h-7 text-[11px] px-2.5 cursor-pointer"
              >
                Cancel
              </Button>
              <Button
                type="button"
                size="sm"
                onClick={() => {
                  onUpdate({ user_notes: localUserNotes || undefined });
                  setIsEditingNotes(false);
                }}
                disabled={isUpdating}
                className="h-7 text-[11px] px-2.5 bg-primary hover:bg-primary/90 text-primary-foreground cursor-pointer"
              >
                {isUpdating ? 'Saving...' : 'Save'}
              </Button>
            </div>
          </div>
        ) : (
          <div className="text-xs text-foreground bg-muted/40 border border-border/20 rounded-md p-2.5 min-h-12 break-words whitespace-pre-wrap">
            {userNotes ? (
              userNotes
            ) : (
              <span className="text-muted-foreground italic text-[11px]">No notes added yet.</span>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
