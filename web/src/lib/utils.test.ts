import { describe, it, expect } from 'vitest';
import {
  cn,
  getProxyImageUrl,
  getPageImageUrl,
  formatChapterTitleWithPage,
  formatBytes,
} from './utils';

describe('cn', () => {
  it('merges class names correctly', () => {
    expect(cn('px-2 py-1', 'bg-blue-500')).toBe('px-2 py-1 bg-blue-500');
    expect(cn('p-4', 'p-2')).toBe('p-2');
    expect(cn('text-sm', false && 'text-lg', null, undefined, 'font-bold')).toBe('text-sm font-bold');
  });
});

describe('getProxyImageUrl', () => {
  it('returns /placeholder.jpg for undefined or empty url', () => {
    expect(getProxyImageUrl(undefined)).toBe('/placeholder.jpg');
    expect(getProxyImageUrl('')).toBe('/placeholder.jpg');
  });

  it('returns direct url if url starts with /, data:, or blob:', () => {
    expect(getProxyImageUrl('/images/cover.jpg')).toBe('/images/cover.jpg');
    expect(getProxyImageUrl('data:image/png;base64,...')).toBe('data:image/png;base64,...');
    expect(getProxyImageUrl('blob:http://localhost/123')).toBe('blob:http://localhost/123');
  });

  it('returns proxy url with encoded parameters for remote urls', () => {
    const rawUrl = 'https://example.com/image.jpg?foo=bar';
    expect(getProxyImageUrl(rawUrl)).toBe(
      `/api/v1/proxy/image?url=${encodeURIComponent(rawUrl)}`
    );
  });

  it('appends referer when provided', () => {
    const rawUrl = 'https://example.com/image.jpg';
    const referer = 'https://example.com/';
    expect(getProxyImageUrl(rawUrl, referer)).toBe(
      `/api/v1/proxy/image?url=${encodeURIComponent(rawUrl)}&referer=${encodeURIComponent(referer)}`
    );
  });
});

describe('getPageImageUrl', () => {
  it('returns /placeholder.jpg if page is undefined', () => {
    expect(getPageImageUrl(undefined)).toBe('/placeholder.jpg');
  });

  it('returns assetUrl directly if page has assetUrl', () => {
    expect(getPageImageUrl({ index: 1, assetUrl: '/cached/page-1.jpg' })).toBe(
      '/cached/page-1.jpg'
    );
  });

  it('returns library page url when mangaId, chapterId, and index > 0 are provided', () => {
    const page = { index: 1, url: 'https://example.com/1.jpg' };
    const mangaId = 'manga-123';
    const chapterId = 'chap-456';
    const providerId = 'mangadex';

    const url = getPageImageUrl(page, mangaId, chapterId, providerId);
    expect(url).toContain('/api/v1/library/manga/manga-123/chapters/chap-456/pages/1');
    expect(url).toContain(`url=${encodeURIComponent('https://example.com/1.jpg')}`);
    expect(url).toContain('provider_id=mangadex');
  });

  it('falls back to getProxyImageUrl when mangaId is not provided', () => {
    const page = { index: 1, url: 'https://example.com/page1.jpg' };
    expect(getPageImageUrl(page)).toBe(getProxyImageUrl('https://example.com/page1.jpg'));
  });
});

describe('formatChapterTitleWithPage', () => {
  it('returns empty string if no title and no chapterNumber', () => {
    expect(formatChapterTitleWithPage()).toBe('');
  });

  it('returns base title when currentPage is missing or < 1', () => {
    expect(formatChapterTitleWithPage('Chapter 10')).toBe('Chapter 10');
    expect(formatChapterTitleWithPage('Chapter 10', 0)).toBe('Chapter 10');
    expect(formatChapterTitleWithPage(undefined, undefined, undefined, 5)).toBe('Chapter 5');
  });

  it('formats chapter with page and total pages', () => {
    expect(formatChapterTitleWithPage('Chapter 12', 3, 20)).toBe('Chapter 12 (3/20)');
    expect(formatChapterTitleWithPage('Chapter 12 - The Beginning', 3, 20)).toBe(
      'Chapter 12 (3/20) - The Beginning'
    );
  });

  it('formats chapter with page only when totalPages is not provided', () => {
    expect(formatChapterTitleWithPage('Chapter 12', 3)).toBe('Chapter 12 (p. 3)');
  });

  it('handles titles matching Vol. and Ch. regex', () => {
    expect(formatChapterTitleWithPage('Vol. 2 Ch. 15', 5, 25)).toBe('Vol. 2 Ch. 15 (5/25)');
    expect(formatChapterTitleWithPage('15 - Final', 1, 10)).toBe('15 (1/10) - Final');
  });

  it('prepends chapterNumber when not in baseTitle', () => {
    expect(formatChapterTitleWithPage('Special Epilogue', 2, 8, 42)).toBe(
      'Chapter 42 (2/8): Special Epilogue'
    );
  });
});

describe('formatBytes', () => {
  it('returns 0 B for invalid, negative, or 0 inputs', () => {
    expect(formatBytes(undefined)).toBe('0 B');
    expect(formatBytes(NaN)).toBe('0 B');
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(-100)).toBe('0 B');
  });

  it('formats byte units correctly', () => {
    expect(formatBytes(500)).toBe('500 B');
    expect(formatBytes(1024)).toBe('1 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(1048576)).toBe('1 MB');
    expect(formatBytes(1073741824)).toBe('1 GB');
    expect(formatBytes(1099511627776)).toBe('1 TB');
  });

  it('respects decimal places parameter', () => {
    expect(formatBytes(1536, 0)).toBe('2 KB');
    expect(formatBytes(1536, 3)).toBe('1.5 KB');
  });
});
