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

export function formatBytes(bytes?: number, decimals = 2): string {
  if (bytes === undefined || isNaN(bytes) || bytes <= 0) return '0 B';
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const idx = Math.min(i, sizes.length - 1);
  return `${parseFloat((bytes / Math.pow(k, idx)).toFixed(dm))} ${sizes[idx]}`;
}

