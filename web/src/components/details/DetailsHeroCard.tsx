import React, { useState } from 'react';
import { ChevronDown, ChevronUp, ExternalLink } from 'lucide-react';
import { Manga } from '../../types/api';
import { Card } from '../ui/card';
import { Skeleton } from '../ui/skeleton';
import { GenrePill } from '../GenrePill';
import { getProxyImageUrl } from '../../lib/utils';

export const READING_MODE_LABELS: Record<string, string> = {
  rtl: 'Right to Left (Manga)',
  ltr: 'Left to Right (Comic)',
  vertical: 'Vertical (Gapped)',
  longstrip: 'Longstrip (Webtoon)',
};

export interface DetailsHeroCardProps {
  manga?: Manga;
  isMangaLoading?: boolean;
  contentProviderName?: string;
  userMetadataSlot?: React.ReactNode;
  headerActionSlot?: React.ReactNode;
}

export const DetailsHeroCard: React.FC<DetailsHeroCardProps> = ({
  manga,
  isMangaLoading = false,
  contentProviderName,
  userMetadataSlot,
  headerActionSlot,
}) => {
  const [showDetailedMetadata, setShowDetailedMetadata] = useState(false);

  const coverSrc =
    manga?.coverAssetUrl || getProxyImageUrl(manga?.coverUrl || manga?.cover, manga?.url);

  const filteredAliases = (manga?.aliases || manga?.meta?.aliases || []).filter(
    (alias) => alias.toLowerCase().trim() !== (manga?.title || '').toLowerCase().trim()
  );

  const authorsList =
    manga?.authors && manga.authors.length > 0
      ? manga.authors
      : manga?.author
      ? [manga.author]
      : manga?.meta?.authors || [];
  const artistsList =
    manga?.artists && manga.artists.length > 0
      ? manga.artists
      : manga?.artist
      ? [manga.artist]
      : manga?.meta?.artists || [];

  const authorsJoined = authorsList.join(', ');
  const artistsJoined = artistsList.join(', ');
  const areAuthorsAndArtistsSame =
    Boolean(authorsJoined) &&
    Boolean(artistsJoined) &&
    authorsJoined.toLowerCase().trim() === artistsJoined.toLowerCase().trim();

  const readingModeKey = (
    manga?.content?.reading_mode ||
    manga?.meta?.content?.reading_mode ||
    manga?.readingMode ||
    manga?.reading_mode ||
    manga?.readingDirection ||
    manga?.meta?.reading_direction ||
    'rtl'
  ).toLowerCase();

  const extLinks =
    manga?.externalLinks && manga.externalLinks.length > 0
      ? manga.externalLinks
      : manga?.meta?.external_links || [];

  return (
    <Card className="grid grid-cols-1 gap-6 p-6 md:grid-cols-[240px_1fr]">
      {/* Left Column: Cover & User Metadata */}
      <div className="flex flex-col gap-4">
        <div className="overflow-hidden rounded-lg bg-muted shadow-md">
          {isMangaLoading ? (
            <Skeleton className="aspect-[2/3] w-full" />
          ) : (
            <img
              src={coverSrc}
              alt={manga?.title || 'Manga Cover'}
              className="aspect-[2/3] w-full object-cover"
              onError={(e) => {
                const proxied = getProxyImageUrl(manga?.coverUrl || manga?.cover, manga?.url);
                if (
                  manga?.coverAssetUrl &&
                  e.currentTarget.src.includes(manga.coverAssetUrl) &&
                  proxied &&
                  proxied !== '/placeholder.jpg'
                ) {
                  e.currentTarget.src = proxied;
                } else if (!e.currentTarget.src.endsWith('/placeholder.jpg')) {
                  e.currentTarget.src = '/placeholder.jpg';
                }
              }}
            />
          )}
        </div>

        {userMetadataSlot}
      </div>

      {/* Right Column: Title, Metadata, Description */}
      <div className="flex flex-col gap-4">
        <div className="flex items-start justify-between gap-4">
          <div className="flex flex-col gap-1">
            <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl">
              {isMangaLoading ? 'Loading Manga...' : manga?.title || 'Untitled Manga'}
            </h1>
            {contentProviderName && (
              <span className="text-xs text-muted-foreground">
                Provider: <span className="font-medium text-foreground">{contentProviderName}</span>
              </span>
            )}
          </div>

          {headerActionSlot}
        </div>

        {filteredAliases.length > 0 && (
          <p className="text-xs text-muted-foreground">
            <span className="font-semibold text-foreground">Aliases:</span>{' '}
            {filteredAliases.join(', ')}
          </p>
        )}

        {/* Essential Info: merged Author/Artist if same */}
        <div className="flex flex-col gap-1 text-sm border-t border-b border-border/40 py-3 mt-1">
          {areAuthorsAndArtistsSame ? (
            <div>
              <span className="text-muted-foreground font-medium">Author / Artist: </span>
              <span className="text-foreground font-semibold">{authorsJoined}</span>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-1">
              {authorsJoined && (
                <div>
                  <span className="text-muted-foreground font-medium">Author: </span>
                  <span className="text-foreground font-semibold">{authorsJoined}</span>
                </div>
              )}
              {artistsJoined && (
                <div>
                  <span className="text-muted-foreground font-medium">Artist: </span>
                  <span className="text-foreground font-semibold">{artistsJoined}</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Collapsible Detailed Metadata */}
        <div className="border-b border-border/40 pb-3 mt-1">
          <button
            type="button"
            onClick={() => setShowDetailedMetadata(!showDetailedMetadata)}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground font-semibold transition-colors focus:outline-hidden cursor-pointer"
          >
            {showDetailedMetadata ? (
              <>
                <ChevronUp className="size-3.5" />
                Hide Details
              </>
            ) : (
              <>
                <ChevronDown className="size-3.5" />
                Show Detailed Metadata
              </>
            )}
          </button>

          {showDetailedMetadata && (
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mt-3 text-xs bg-muted/40 p-3 rounded-lg border border-border/30">
              <div>
                <span className="text-muted-foreground block text-[10px] uppercase font-bold">
                  Reading Mode
                </span>
                <span className="font-medium text-foreground uppercase">
                  {READING_MODE_LABELS[readingModeKey] || readingModeKey}
                </span>
              </div>
              <div>
                <span className="text-muted-foreground block text-[10px] uppercase font-bold">
                  Content Rating
                </span>
                <span className="font-medium text-foreground capitalize">
                  {manga?.contentRating || manga?.meta?.content_rating || 'safe'}
                </span>
              </div>
              {manga?.publisher && (
                <div>
                  <span className="text-muted-foreground block text-[10px] uppercase font-bold">
                    Publisher
                  </span>
                  <span className="font-medium text-foreground">{manga.publisher}</span>
                </div>
              )}
              {manga?.releaseYear && (
                <div>
                  <span className="text-muted-foreground block text-[10px] uppercase font-bold">
                    Release Year
                  </span>
                  <span className="font-medium text-foreground">{manga.releaseYear}</span>
                </div>
              )}
              {manga?.country && (
                <div>
                  <span className="text-muted-foreground block text-[10px] uppercase font-bold">
                    Country
                  </span>
                  <span className="font-medium text-foreground uppercase">{manga.country}</span>
                </div>
              )}
              {extLinks.length > 0 && (
                <div className="col-span-2 sm:col-span-3 pt-1 border-t border-border/20">
                  <span className="text-muted-foreground block text-[10px] uppercase font-bold mb-1.5">
                    External Links
                  </span>
                  <div className="flex flex-wrap gap-1.5">
                    {extLinks.map((link, idx) => (
                      <a
                        key={`${link.url}-${idx}`}
                        href={link.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs text-primary hover:underline bg-primary/10 rounded px-2 py-0.5 font-medium transition-colors"
                      >
                        <ExternalLink className="size-3" />
                        <span>{link.label || link.provider || link.url}</span>
                      </a>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Genre / Tag Pills */}
        {((manga?.tags && manga.tags.length > 0) || (manga?.genres && manga.genres.length > 0)) && (
          <div className="flex flex-wrap gap-1.5">
            {(manga?.tags || manga?.genres || manga?.meta?.tags || []).map((genre) => (
              <GenrePill key={genre} genre={genre} />
            ))}
          </div>
        )}

        {/* Description */}
        <div className="text-sm leading-relaxed text-muted-foreground whitespace-pre-wrap">
          {manga?.description ||
            manga?.meta?.description ||
            'No description available for this series.'}
        </div>
      </div>
    </Card>
  );
};
