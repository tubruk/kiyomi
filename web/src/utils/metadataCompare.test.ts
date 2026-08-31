import { describe, it, expect } from 'vitest';
import {
  areArraySetsEqual,
  mergeStringArrays,
  areExternalLinkSetsEqual,
  mergeExternalLinkArrays,
} from './metadataCompare';
import { ExternalLink } from '../types/api';

describe('metadataCompare utils', () => {
  describe('areArraySetsEqual', () => {
    it('returns true for identical arrays', () => {
      expect(areArraySetsEqual(['Action', 'Adventure'], ['Action', 'Adventure'])).toBe(true);
    });

    it('returns true regardless of order and casing', () => {
      expect(areArraySetsEqual(['Action', 'Adventure'], ['adventure', 'ACTION'])).toBe(true);
    });

    it('returns true when handling whitespace differences', () => {
      expect(areArraySetsEqual([' Action ', 'Adventure'], ['action', 'adventure '])).toBe(true);
    });

    it('returns true for empty or undefined arrays', () => {
      expect(areArraySetsEqual([], [])).toBe(true);
      expect(areArraySetsEqual(undefined, [])).toBe(true);
      expect(areArraySetsEqual([], undefined)).toBe(true);
      expect(areArraySetsEqual(undefined, undefined)).toBe(true);
    });

    it('returns false for arrays of different lengths', () => {
      expect(areArraySetsEqual(['Action'], ['Action', 'Adventure'])).toBe(false);
      expect(areArraySetsEqual(['Action', 'Adventure'], ['Action'])).toBe(false);
    });

    it('returns false for different elements of same length', () => {
      expect(areArraySetsEqual(['Action', 'Adventure'], ['Action', 'Comedy'])).toBe(false);
    });
  });

  describe('mergeStringArrays', () => {
    it('merges arrays deduplicating case-insensitively while preserving first occurrence casing', () => {
      const merged = mergeStringArrays(['Fantasy', 'Magic'], ['fantasy', 'Adventure', 'MAGIC']);
      expect(merged).toEqual(['Fantasy', 'Magic', 'Adventure']);
    });

    it('trims whitespace and ignores empty or blank strings', () => {
      const merged = mergeStringArrays(['  Fantasy  ', '', '   '], ['Magic', '   ']);
      expect(merged).toEqual(['Fantasy', 'Magic']);
    });

    it('handles undefined or empty arrays gracefully', () => {
      expect(mergeStringArrays(undefined, ['Tag1'], undefined, [])).toEqual(['Tag1']);
      expect(mergeStringArrays()).toEqual([]);
    });

    it('preserves order of first encountered items', () => {
      expect(mergeStringArrays(['C', 'B', 'A'], ['A', 'B', 'D'])).toEqual(['C', 'B', 'A', 'D']);
    });
  });

  describe('areExternalLinkSetsEqual', () => {
    it('returns true when URLs match case-insensitively regardless of order or label differences', () => {
      const a: ExternalLink[] = [
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
        { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/1' },
      ];
      const b: ExternalLink[] = [
        { provider: 'custom', label: 'AL', url: 'https://ANILIST.co/manga/1' },
        { provider: 'mangadex', label: 'MD', url: 'https://mangadex.org/title/1' },
      ];
      expect(areExternalLinkSetsEqual(a, b)).toBe(true);
    });

    it('returns true for empty or undefined arrays', () => {
      expect(areExternalLinkSetsEqual([], [])).toBe(true);
      expect(areExternalLinkSetsEqual(undefined, [])).toBe(true);
      expect(areExternalLinkSetsEqual([], undefined)).toBe(true);
      expect(areExternalLinkSetsEqual(undefined, undefined)).toBe(true);
    });

    it('ignores links with blank or missing URLs', () => {
      const a: ExternalLink[] = [
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
        { provider: 'custom', label: 'Empty', url: '  ' },
      ];
      const b: ExternalLink[] = [
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
      ];
      expect(areExternalLinkSetsEqual(a, b)).toBe(true);
    });

    it('returns false for different URLs or different counts', () => {
      const a: ExternalLink[] = [
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
      ];
      const b: ExternalLink[] = [
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/2' },
      ];
      expect(areExternalLinkSetsEqual(a, b)).toBe(false);

      const c: ExternalLink[] = [
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
        { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/1' },
      ];
      expect(areExternalLinkSetsEqual(a, c)).toBe(false);
    });
  });

  describe('mergeExternalLinkArrays', () => {
    it('merges link arrays deduplicating by URL case-insensitively', () => {
      const a: ExternalLink[] = [
        { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/1' },
      ];
      const b: ExternalLink[] = [
        { provider: 'anilist', label: 'Duplicate AL', url: 'https://ANILIST.CO/MANGA/1' },
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
      ];
      const merged = mergeExternalLinkArrays(a, b);
      expect(merged).toEqual([
        { provider: 'anilist', label: 'AniList', url: 'https://anilist.co/manga/1' },
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org/title/1' },
      ]);
    });

    it('applies fallback defaults for missing provider or label', () => {
      const a: ExternalLink[] = [
        { provider: '', label: '', url: 'https://example.com/custom' },
        { provider: 'myanimelist', label: '', url: 'https://myanimelist.net/manga/1' },
      ];
      const merged = mergeExternalLinkArrays(a);
      expect(merged).toEqual([
        { provider: 'custom', label: 'Link', url: 'https://example.com/custom' },
        { provider: 'myanimelist', label: 'myanimelist', url: 'https://myanimelist.net/manga/1' },
      ]);
    });

    it('ignores invalid or empty items and handles undefined arrays', () => {
      const merged = mergeExternalLinkArrays(
        undefined,
        [{ provider: 'p1', label: 'L1', url: '   ' }, null as any, undefined as any],
        [{ provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org' }]
      );
      expect(merged).toEqual([
        { provider: 'mangadex', label: 'MangaDex', url: 'https://mangadex.org' },
      ]);
    });
  });
});
