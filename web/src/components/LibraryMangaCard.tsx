import React, { memo, useMemo } from 'react';
import { Link } from '@tanstack/react-router';
import { Star, Heart, Bookmark, Trash2 } from 'lucide-react';
import { Manga } from '../types/api';
import { Badge } from './ui/badge';
import { getProxyImageUrl } from '../lib/utils';
import { useChapterList } from '../api/hooks';

interface LibraryMangaCardProps {
  manga: Manga;
  onDelete?: (mangaId: string) => void;
  isDeleting?: boolean;
}

export const LibraryMangaCard: React.FC<LibraryMangaCardProps> = memo(({
  manga,
  onDelete,
  isDeleting,
}) => {
  const coverSrc = manga.coverAssetUrl || getProxyImageUrl(manga.coverUrl || manga.cover, manga.url);
  const authorDisplay = (manga.authors || (manga.author ? [manga.author] : [])).slice(0, 2).join(', ');

  const userFavorite = manga.userFavorite || manga.user_favorite || manga.meta?.user_favorite;
  const userStatus = manga.userStatus || manga.user_status || manga.meta?.user_status;
  const userRating = manga.userRating || manga.user_rating || manga.meta?.user_rating;
  const providerId = manga.sourceId || manga.contentProviderId || manga.meta?.content?.provider_id;

  const { data: chaptersData } = useChapterList(manga.id);
  const chapters = chaptersData?.chapters || [];

  const { unreadCount, isAllRead } = useMemo(() => {
    if (chapters.length === 0) return { unreadCount: 0, isAllRead: false };
    const readCount = chapters.filter((c) => Boolean(c.meta?.is_read ?? (c as any).is_read)).length;
    const unread = chapters.length - readCount;
    return {
      unreadCount: unread,
      isAllRead: readCount === chapters.length && chapters.length > 0,
    };
  }, [chapters]);

  return (
    <div className="group relative flex flex-col overflow-hidden rounded-lg border border-border bg-card transition-all duration-200 hover:-translate-y-1 hover:border-primary/50 hover:shadow-md">
      <Link to="/manga/$mangaId" params={{ mangaId: manga.id }} className="flex flex-col flex-1">
        <div className="relative aspect-[2/3] w-full overflow-hidden bg-muted">
          <img
            src={coverSrc}
            alt={manga.title}
            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
            loading="lazy"
            onError={(e) => {
              const fallback = manga.coverUrl || manga.cover;
              if (fallback && e.currentTarget.src !== fallback) {
                e.currentTarget.src = fallback;
              }
            }}
          />
          {providerId && (
            <div className="absolute top-2 left-2 z-10">
              <Badge
                variant="secondary"
                className="bg-background/80 text-[10px] uppercase tracking-wider backdrop-blur-xs font-semibold"
              >
                {providerId}
              </Badge>
            </div>
          )}
          {chapters.length > 0 && (
            <div className="absolute bottom-2 left-2 z-10">
              {isAllRead ? (
                <Badge
                  variant="secondary"
                  className="bg-emerald-600 text-white text-[10px] font-bold uppercase tracking-wider backdrop-blur-xs shadow-xs"
                >
                  Completed
                </Badge>
              ) : unreadCount > 0 ? (
                <Badge
                  variant="default"
                  className="bg-primary/95 text-primary-foreground text-[10px] font-bold backdrop-blur-xs shadow-xs"
                >
                  {unreadCount} unread
                </Badge>
              ) : null}
            </div>
          )}
          {userFavorite && (
            <div className={`absolute top-2 ${providerId ? 'right-2' : 'left-2'} z-10 rounded-full bg-black/60 p-1 text-rose-500`} title="Favorite">
              <Heart className="size-3.5 fill-rose-500" />
            </div>
          )}
        </div>
        <div className="p-3 flex flex-col flex-1 justify-between gap-1.5">
          <h3 className="line-clamp-2 text-sm font-semibold text-foreground group-hover:text-primary transition-colors">
            {manga.title}
          </h3>
          {authorDisplay && (
            <p className="line-clamp-1 text-xs text-muted-foreground">{authorDisplay}</p>
          )}
          <div className="flex flex-wrap items-center gap-1">
            {userStatus && (
              <Badge variant="secondary" className="text-[9px] uppercase px-1.5 py-0">
                {userStatus.replace(/_/g, ' ')}
              </Badge>
            )}
            {userRating && userRating > 0 ? (
              <span className="inline-flex items-center gap-0.5 text-[10px] font-semibold text-amber-500 bg-amber-500/10 rounded px-1.5 py-0">
                <Star className="size-2.5 fill-amber-500" />
                {userRating}
              </span>
            ) : null}
            {manga.shelves && manga.shelves.length > 0 && (
              <Badge variant="outline" className="text-[9px] px-1.5 py-0 border-primary/30 text-primary">
                <Bookmark className="size-2.5 mr-0.5" />
                {manga.shelves[0]}
              </Badge>
            )}
          </div>
        </div>
      </Link>

      {onDelete && (
        <button
          type="button"
          className="absolute top-2 right-2 z-20 flex size-7 items-center justify-center rounded-full bg-black/60 text-white opacity-0 transition-opacity hover:bg-destructive group-hover:opacity-100 cursor-pointer"
          title="Remove from Library"
          aria-label="Remove from Library"
          disabled={isDeleting}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            if (confirm(`Remove "${manga.title}" from library?`)) {
              onDelete(manga.id);
            }
          }}
        >
          <Trash2 className="size-3.5" aria-hidden />
        </button>
      )}
    </div>
  );
});

LibraryMangaCard.displayName = 'LibraryMangaCard';
