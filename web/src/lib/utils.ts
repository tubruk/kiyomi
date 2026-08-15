import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getProxyImageUrl(url?: string, referer?: string): string {
  if (!url) return '/placeholder.jpg';
  if (url.startsWith('/') || url.startsWith('data:') || url.startsWith('blob:')) {
    return url;
  }
  let proxyUrl = `/api/v1/proxy/image?url=${encodeURIComponent(url)}`;
  if (referer) {
    proxyUrl += `&referer=${encodeURIComponent(referer)}`;
  }
  return proxyUrl;
}

export function formatChapterTitleWithPage(
  chapterTitle?: string,
  currentPage?: number,
  totalPages?: number,
  chapterNumber?: number
): string {
  if (!chapterTitle && chapterNumber === undefined) return '';

  const baseTitle = chapterTitle || (chapterNumber !== undefined ? `Chapter ${chapterNumber}` : '');
  if (!currentPage || currentPage < 1) return baseTitle;

  const pageStr = totalPages && totalPages > 0 ? `(${currentPage}/${totalPages})` : `(p. ${currentPage})`;

  // If baseTitle starts with "Chapter X", "Ch. X", "Vol. Y Ch. X", or digit "X"
  const chapterMatch = baseTitle.match(/^((?:Vol\.\s*\d+\s+)?(?:Chapter|Ch\.)\s*[\d.]+|\d+)(.*)$/i);
  if (chapterMatch) {
    const chapterPart = chapterMatch[1];
    const restPart = chapterMatch[2];
    return `${chapterPart} ${pageStr}${restPart}`;
  }

  // If chapterNumber is explicitly known and not already in baseTitle
  if (chapterNumber !== undefined && !baseTitle.toLowerCase().includes(`chapter ${chapterNumber}`)) {
    return `Chapter ${chapterNumber} ${pageStr}: ${baseTitle}`;
  }

  // Fallback: append pageStr to baseTitle
  return `${baseTitle} ${pageStr}`;
}

