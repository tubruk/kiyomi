import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getProxyImageUrl(url?: string, referer?: string): string {
  if (!url) return '';
  if (url.startsWith('/') || url.startsWith('data:') || url.startsWith('blob:')) {
    return url;
  }
  let proxyUrl = `/api/v1/proxy/image?url=${encodeURIComponent(url)}`;
  if (referer) {
    proxyUrl += `&referer=${encodeURIComponent(referer)}`;
  }
  return proxyUrl;
}

export function getPageImageUrl(
  page?: { index: number; url?: string; assetUrl?: string },
  mangaId?: string,
  chapterId?: string,
  providerId?: string,
  referer?: string
): string {
  if (!page) return '';
  if (page.assetUrl) {
    return page.assetUrl;
  }
  if (mangaId && chapterId && page.index > 0) {
    let pageUrl = `/api/v1/library/manga/${encodeURIComponent(mangaId)}/chapters/${encodeURIComponent(chapterId)}/pages/${page.index}`;
    const params = new URLSearchParams();
    if (page.url) {
      params.set('url', page.url);
    }
    if (providerId) {
      params.set('provider_id', providerId);
    }
    const query = params.toString();
    return query ? `${pageUrl}?${query}` : pageUrl;
  }
  return getProxyImageUrl(page.url, referer);
}

const CHAPTER_TITLE_REGEX = /^((?:Vol\.\s*\d+\s+)?(?:Chapter|Ch\.)\s*[\d.]+|\d+)(.*)$/i;

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
  const chapterMatch = baseTitle.match(CHAPTER_TITLE_REGEX);
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

