import React from 'react';
import { Link } from '@tanstack/react-router';
import { Manga } from '../types/api';
import { Badge } from './ui/badge';
import { CoverImage } from './CoverImage';
import { cn, getProxyImageUrl } from '../lib/utils';

interface MangaCardProps {
  manga: Manga;
  isExplore?: boolean;
}

export const MangaCard: React.FC<MangaCardProps> = ({ manga }) => {
  const proxied = getProxyImageUrl(manga.coverUrl || manga.cover, manga.url);
  const coverSrc = manga.coverAssetUrl || proxied;

  const isRemote = Boolean(manga.sourceId || manga.contentProviderId);
  const providerId = manga.sourceId || manga.contentProviderId || '';
  const remoteId = manga.contentRemoteId || manga.id || '';

  const imgClassName = cn(
    'h-full w-full object-cover transition-transform duration-300 group-hover:scale-105',
    manga.availability === 'unavailable' && 'opacity-70 grayscale-[35%]'
  );

  if (isRemote && providerId && remoteId) {
    return (
      <Link
        to="/providers/$providerId/manga/$remoteId"
        params={{ providerId, remoteId }}
        className="group relative flex flex-col overflow-hidden rounded-lg border border-border bg-card transition-all duration-200 hover:-translate-y-1 hover:border-primary/50 hover:shadow-md"
      >
        {/* Cover + overlays */}
        <div className="relative aspect-[2/3] w-full overflow-hidden bg-muted">
          <CoverImage
            src={coverSrc}
            fallbackSrc={manga.coverAssetUrl ? proxied : undefined}
            alt={manga.title}
            shape="auto"
            className={imgClassName}
          />

          {/* Source ID Overlay Badge */}
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

          {/* Status & Availability Badges */}
          <div className="absolute top-2 right-2 z-10 flex flex-col items-end gap-1">
            {manga.availability === 'unavailable' && (
              <Badge
                variant="secondary"
                className="bg-background/80 text-muted-foreground border border-border/50 text-[10px] uppercase font-semibold backdrop-blur-xs shadow-xs"
              >
                Unavailable
              </Badge>
            )}
            {manga.status && (
              <Badge
                variant={manga.status.toLowerCase() === 'ongoing' ? 'default' : 'secondary'}
                className="text-[10px] uppercase font-semibold"
              >
                {manga.status}
              </Badge>
            )}
          </div>
        </div>

        {/* Content Container */}
        <div className="flex flex-col flex-1 p-3">
          <h3
            className="line-clamp-2 text-sm font-semibold leading-snug text-foreground group-hover:text-primary transition-colors"
            title={manga.title}
          >
            {manga.title}
          </h3>
          {(manga.author || (manga.authors && manga.authors.length > 0)) && (
            <p className="mt-1 line-clamp-1 text-xs text-muted-foreground">
              {manga.author || manga.authors?.join(', ')}
            </p>
          )}
        </div>
      </Link>
    );
  }

  return (
    <Link
      to="/manga/$mangaId"
      params={{ mangaId: manga.id }}
      className="group relative flex flex-col overflow-hidden rounded-lg border border-border bg-card transition-all duration-200 hover:-translate-y-1 hover:border-primary/50 hover:shadow-md"
    >
      {/* Cover + overlays */}
      <div className="relative aspect-[2/3] w-full overflow-hidden bg-muted">
        <CoverImage
          src={coverSrc}
          fallbackSrc={manga.coverAssetUrl ? proxied : undefined}
          alt={manga.title}
          shape="auto"
          className={imgClassName}
        />

        {/* Status & Availability Badges */}
        <div className="absolute top-2 right-2 z-10 flex flex-col items-end gap-1">
          {manga.availability === 'unavailable' && (
            <Badge
              variant="secondary"
              className="bg-background/80 text-muted-foreground border border-border/50 text-[10px] uppercase font-semibold backdrop-blur-xs shadow-xs"
            >
              Unavailable
            </Badge>
          )}
          {manga.status && (
            <Badge
              variant={manga.status.toLowerCase() === 'ongoing' ? 'default' : 'secondary'}
              className="text-[10px] uppercase font-semibold"
            >
              {manga.status}
            </Badge>
          )}
        </div>
      </div>

      {/* Content Container */}
      <div className="flex flex-col flex-1 p-3">
        <h3
          className="line-clamp-2 text-sm font-semibold leading-snug text-foreground group-hover:text-primary transition-colors"
          title={manga.title}
        >
          {manga.title}
        </h3>
        {(manga.author || (manga.authors && manga.authors.length > 0)) && (
          <p className="mt-1 line-clamp-1 text-xs text-muted-foreground">
            {manga.author || manga.authors?.join(', ')}
          </p>
        )}
      </div>
    </Link>
  );
};