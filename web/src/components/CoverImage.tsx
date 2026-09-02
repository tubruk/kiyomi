import React, { useRef, useState } from 'react';
import { BookOpen, Image as ImageIcon } from 'lucide-react';
import { cn } from '../lib/utils';

type CoverIcon = 'book' | 'image';

export interface CoverImageProps {
  /**
   * Image source URL. Pass `undefined` or `''` to render only the placeholder.
   */
  src?: string;
  alt?: string;
  /**
   * Optional secondary fallback URL tried once if `src` fails to load.
   * Useful for coverAssetUrl → coverUrl (proxied) cascade.
   */
  fallbackSrc?: string;
  /**
   * Which icon to show as the placeholder. Use 'book' for manga/library covers,
   * 'image' for generic image tiles (e.g. cover-compare).
   */
  icon?: CoverIcon;
  /**
   * Tailwind size class for the placeholder icon (e.g. 'size-5', 'size-12').
   * Defaults to 'size-10'.
   */
  iconSize?: string;
  /**
   * Tailwind classes applied to the inner <img>. The wrapper always provides
   * `bg-muted` + an aspect ratio via the chosen `shape`.
   */
  className?: string;
  /**
   * Tailwind classes applied to the outer wrapper. Use this to override the
   * aspect ratio, sizing, or rounded corners per-call-site.
   */
  containerClassName?: string;
  /**
   * Visual shape of the container.
   * - `portrait`: aspect-[2/3] (default, manga covers)
   * - `square`: aspect-square (search-result tiles)
   * - `auto`: no aspect enforced; size comes from containerClassName
   */
  shape?: 'portrait' | 'square' | 'auto';
  /**
   * Disable lazy loading (e.g. for above-the-fold hero images).
   */
  eager?: boolean;
  /**
   * If true, never show the placeholder pulse shimmer (e.g. when this is the
   * placeholder itself).
   */
  noShimmer?: boolean;
  /**
   * Forwarded click handler (e.g. when the cover is wrapped in a button).
   */
  onClick?: React.MouseEventHandler<HTMLDivElement>;
  /**
   * Forwarded to the inner <img> onError for advanced consumers.
   */
  onError?: React.ReactEventHandler<HTMLImageElement>;
}

/**
 * Unified cover image with a subtle icon placeholder.
 *
 * States:
 * - Empty src (or never loaded yet): shows bg-muted + BookOpen icon + optional pulse shimmer.
 * - Loaded: img fades in over the placeholder.
 * - Error: img is hidden; placeholder remains visible permanently.
 *
 * One source of truth for the cover look — used by every card, search tile,
 * compare tile, and add-provider row in the app.
 */
export const CoverImage = React.forwardRef<HTMLDivElement, CoverImageProps>(
  (
    {
      src,
      alt = '',
      fallbackSrc,
      icon = 'book',
      iconSize = 'size-10',
      className,
      containerClassName,
      shape = 'portrait',
      eager = false,
      noShimmer = false,
      onClick,
      onError,
    },
    ref,
  ) => {
    const [loaded, setLoaded] = useState(false);
    const [errored, setErrored] = useState(false);
    const [currentSrc, setCurrentSrc] = useState(src);
    // Tracks which src we have already attempted (and failed) for. Prevents
    // re-firing the fallback cascade if React re-renders with the same src.
    const attemptedRef = useRef<string | null>(null);

    // Reset state when src prop changes (e.g. row swap in a list).
    React.useEffect(() => {
      setLoaded(false);
      setErrored(false);
      setCurrentSrc(src);
      attemptedRef.current = null;
    }, [src]);

    const showShimmer = Boolean(currentSrc) && !loaded && !errored && !noShimmer;

    const handleError: React.ReactEventHandler<HTMLImageElement> = (e) => {
      // If we have a fallback and we have not already tried it for this initial src,
      // swap to it once. Otherwise mark errored permanently.
      if (
        fallbackSrc &&
        attemptedRef.current !== fallbackSrc &&
        e.currentTarget.src !== fallbackSrc
      ) {
        attemptedRef.current = fallbackSrc;
        setCurrentSrc(fallbackSrc);
        return;
      }
      setErrored(true);
      onError?.(e);
    };

    const Icon = icon === 'book' ? BookOpen : ImageIcon;
    const shapeClass =
      shape === 'portrait'
        ? 'aspect-[2/3]'
        : shape === 'square'
        ? 'aspect-square'
        : '';

    return (
      <div
        ref={ref}
        onClick={onClick}
        data-testid="cover-image-root"
        className={cn(
          'relative overflow-hidden bg-muted',
          shapeClass,
          containerClassName
        )}
      >
        {/* Placeholder layer — always present underneath the image. */}
        <div
          aria-hidden
          className={cn(
            'absolute inset-0 flex items-center justify-center text-muted-foreground/60',
            showShimmer && 'animate-pulse'
          )}
        >
          <Icon className={iconSize} />
        </div>

        {/* Image layer — fades in once loaded. Hidden permanently on error. */}
        {currentSrc && (
          <img
            key={currentSrc}
            src={currentSrc}
            alt={alt}
            loading={eager ? 'eager' : 'lazy'}
            onLoad={() => setLoaded(true)}
            onError={handleError}
            className={cn(
              'relative h-full w-full object-cover transition-opacity duration-300',
              loaded ? 'opacity-100' : 'opacity-0',
              errored && 'hidden',
              className
            )}
            style={errored ? { display: 'none' } : undefined}
          />
        )}
      </div>
    );
  }
);

CoverImage.displayName = 'CoverImage';