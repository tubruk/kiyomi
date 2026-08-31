import { Chapter, Manga } from '../types/api';

export interface ReadingCtaInfo {
  label: string;
  chapterId: string;
  page: number;
  isResume: boolean;
  isCompleted: boolean;
  type: 'start' | 'resume' | 'next' | 'reread';
}

const getIsRead = (c: Chapter): boolean => {
  return Boolean(c.is_read || c.isRead || c.meta?.is_read);
};

const getLastPage = (c: Chapter): number => {
  return c.last_read_page ?? c.lastReadPage ?? c.meta?.last_read_page ?? 0;
};

export const getReadingCtaInfo = (
  chapters?: Chapter[] | null,
  lastReadChapterIdOrManga?: string | Manga | null
): ReadingCtaInfo | null => {
  if (!chapters || chapters.length === 0) return null;

  const sorted = [...chapters].sort((a, b) => (a.number ?? 0) - (b.number ?? 0));
  const firstChapter = sorted[0];

  let lastReadChapterId: string | undefined;
  if (typeof lastReadChapterIdOrManga === 'string') {
    lastReadChapterId = lastReadChapterIdOrManga;
  } else if (lastReadChapterIdOrManga) {
    lastReadChapterId =
      lastReadChapterIdOrManga.lastReadChapterId ||
      lastReadChapterIdOrManga.last_read_chapter_id ||
      lastReadChapterIdOrManga.meta?.last_read_chapter_id;
  }

  const lastReadChapter = lastReadChapterId
    ? chapters.find((c) => c.id === lastReadChapterId)
    : undefined;

  // 1. Check in-progress chapter (!is_read && last_read_page > 1)
  let inProgressChapter: Chapter | undefined;
  if (lastReadChapter) {
    const isRead = getIsRead(lastReadChapter);
    const lastPage = getLastPage(lastReadChapter);
    if (!isRead && lastPage > 1) {
      inProgressChapter = lastReadChapter;
    }
  }

  if (!inProgressChapter) {
    const inProgressList = sorted.filter((c) => {
      const isRead = getIsRead(c);
      const lastPage = getLastPage(c);
      return !isRead && lastPage > 1;
    });
    if (inProgressList.length > 0) {
      inProgressChapter = inProgressList[inProgressList.length - 1];
    }
  }

  if (inProgressChapter) {
    const lastPage = getLastPage(inProgressChapter) || 1;
    const chNum = inProgressChapter.number ?? 1;
    return {
      type: 'resume',
      label: `Resume Ch. ${chNum} (p. ${lastPage})`,
      chapterId: inProgressChapter.id,
      page: lastPage,
      isResume: true,
      isCompleted: false,
    };
  }

  // 2. Check read chapters
  const readChapters = sorted.filter((c) => getIsRead(c));

  if (readChapters.length === 0) {
    return {
      type: 'start',
      label: 'Start Reading',
      chapterId: firstChapter.id,
      page: 1,
      isResume: false,
      isCompleted: false,
    };
  }

  if (readChapters.length === sorted.length) {
    const chNum = firstChapter.number ?? 1;
    return {
      type: 'reread',
      label: `Re-read Ch. ${chNum}`,
      chapterId: firstChapter.id,
      page: 1,
      isResume: false,
      isCompleted: true,
    };
  }

  // 3. If Ch X completed: Read Ch. X+1
  let nextChapter: Chapter | undefined;
  if (lastReadChapter && getIsRead(lastReadChapter)) {
    const lastIdx = sorted.findIndex((c) => c.id === lastReadChapter.id);
    if (lastIdx >= 0) {
      nextChapter = sorted.slice(lastIdx + 1).find((c) => !getIsRead(c));
    }
  }

  if (!nextChapter) {
    const lastReadCh = readChapters[readChapters.length - 1];
    const lastReadIdx = sorted.findIndex((c) => c.id === lastReadCh.id);
    if (lastReadIdx >= 0) {
      nextChapter = sorted.slice(lastReadIdx + 1).find((c) => !getIsRead(c));
    }
  }

  if (!nextChapter) {
    nextChapter = sorted.find((c) => !getIsRead(c));
  }

  if (nextChapter) {
    const chNum = nextChapter.number ?? 1;
    return {
      type: 'next',
      label: `Read Ch. ${chNum}`,
      chapterId: nextChapter.id,
      page: 1,
      isResume: false,
      isCompleted: false,
    };
  }

  return {
    type: 'start',
    label: 'Start Reading',
    chapterId: firstChapter.id,
    page: 1,
    isResume: false,
    isCompleted: false,
  };
};
