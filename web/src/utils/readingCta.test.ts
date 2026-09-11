import { describe, it, expect } from 'vitest';
import { getReadingCtaInfo } from './readingCta';
import { Chapter, Manga } from '../types/api';

describe('getReadingCtaInfo', () => {
  it('returns null when chapters is null, undefined, or empty', () => {
    expect(getReadingCtaInfo(null)).toBeNull();
    expect(getReadingCtaInfo(undefined)).toBeNull();
    expect(getReadingCtaInfo([])).toBeNull();
  });

  it('returns "Start Reading" for chapter 1 when no chapters are read', () => {
    const chapters: Chapter[] = [
      { id: 'ch-2', name: 'Chapter 2', number: 2, is_read: false },
      { id: 'ch-1', name: 'Chapter 1', number: 1, is_read: false },
    ];
    const result = getReadingCtaInfo(chapters);
    expect(result).toEqual({
      type: 'start',
      label: 'Start Reading',
      chapterId: 'ch-1',
      page: 1,
      isResume: false,
      isCompleted: false,
    });
  });

  it('returns "Resume Ch. X (p. Y)" when a chapter is in progress via lastReadChapterId', () => {
    const chapters: Chapter[] = [
      { id: 'ch-1', name: 'Chapter 1', number: 1, is_read: true },
      { id: 'ch-2', name: 'Chapter 2', number: 2, is_read: false, meta: { last_read_page: 15 } },
      { id: 'ch-3', name: 'Chapter 3', number: 3, is_read: false },
    ];
    const manga: Manga = {
      id: 'm-1',
      title: 'Manga',
      lastReadChapterId: 'ch-2',
      metadata: { title: 'Manga', aliases: [], description: '', authors: [], artists: [], tags: [], collections: [], publishers: [] },
      user_state: { status: 'reading', rating: 0, favorite: false, notes: '' },
      bindings: { providers: [] },
    };
    const result = getReadingCtaInfo(chapters, manga);
    expect(result).toEqual({
      type: 'resume',
      label: 'Resume Ch. 2 (p. 15)',
      chapterId: 'ch-2',
      page: 15,
      isResume: true,
      isCompleted: false,
    });
  });

  it('returns "Resume" when an in-progress chapter exists in chapters list even without lastReadChapterId', () => {
    const chapters: Chapter[] = [
      { id: 'ch-1', name: 'Chapter 1', number: 1, is_read: true },
      { id: 'ch-2', name: 'Chapter 2', number: 2, is_read: false, last_read_page: 7 },
    ];
    const result = getReadingCtaInfo(chapters);
    expect(result).toEqual({
      type: 'resume',
      label: 'Resume Ch. 2 (p. 7)',
      chapterId: 'ch-2',
      page: 7,
      isResume: true,
      isCompleted: false,
    });
  });

  it('returns "Re-read Ch. 1" and isCompleted: true when all chapters are read', () => {
    const chapters: Chapter[] = [
      { id: 'ch-1', name: 'Chapter 1', number: 1, is_read: true },
      { id: 'ch-2', name: 'Chapter 2', number: 2, is_read: true },
    ];
    const result = getReadingCtaInfo(chapters);
    expect(result).toEqual({
      type: 'reread',
      label: 'Re-read Ch. 1',
      chapterId: 'ch-1',
      page: 1,
      isResume: false,
      isCompleted: true,
    });
  });

  it('returns "Read Ch. X+1" for next unread chapter after last completed chapter', () => {
    const chapters: Chapter[] = [
      { id: 'ch-1', name: 'Chapter 1', number: 1, is_read: true },
      { id: 'ch-2', name: 'Chapter 2', number: 2, is_read: false },
      { id: 'ch-3', name: 'Chapter 3', number: 3, is_read: false },
    ];
    const manga: Manga = {
      id: 'm-1',
      title: 'Manga',
      user_state: { status: 'reading', rating: 0, favorite: false, notes: '', last_read_chapter_id: 'ch-1' },
      metadata: { title: 'Manga', aliases: [], description: '', authors: [], artists: [], tags: [], collections: [], publishers: [] },
      bindings: { providers: [] },
    };
    const result = getReadingCtaInfo(chapters, manga);
    expect(result).toEqual({
      type: 'next',
      label: 'Read Ch. 2',
      chapterId: 'ch-2',
      page: 1,
      isResume: false,
      isCompleted: false,
    });
  });

  it('handles lastReadChapterId string argument directly', () => {
    const chapters: Chapter[] = [
      { id: 'ch-1', name: 'Chapter 1', number: 1, is_read: true },
      { id: 'ch-2', name: 'Chapter 2', number: 2, is_read: false },
    ];
    const result = getReadingCtaInfo(chapters, 'ch-1');
    expect(result).toEqual({
      type: 'next',
      label: 'Read Ch. 2',
      chapterId: 'ch-2',
      page: 1,
      isResume: false,
      isCompleted: false,
    });
  });

  it('handles fallback gracefully when last read chapter has no subsequent unread in sequence', () => {
    const chapters: Chapter[] = [
      { id: 'ch-1', name: 'Chapter 1', number: 1, is_read: false },
      { id: 'ch-2', name: 'Chapter 2', number: 2, is_read: true },
    ];
    const result = getReadingCtaInfo(chapters, 'ch-2');
    expect(result).toEqual({
      type: 'next',
      label: 'Read Ch. 1',
      chapterId: 'ch-1',
      page: 1,
      isResume: false,
      isCompleted: false,
    });
  });
});
