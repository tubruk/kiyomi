import * as fs from 'fs';
import * as path from 'path';
import sqlite3 from 'better-sqlite3';

export async function libraryContains(home: string, mangaTitle: string): Promise<boolean> {
  const libraryPath = path.join(home, 'library');
  if (!fs.existsSync(libraryPath)) {
    return false;
  }
  const entries = fs.readdirSync(libraryPath, { withFileTypes: true });
  return entries.some((entry) => entry.isDirectory() && entry.name === mangaTitle);
}

export function fileExists(filePath: string): boolean {
  return fs.existsSync(filePath);
}

export async function chapterDownloaded(
  home: string,
  mangaTitle: string,
  chapterNumber: number
): Promise<boolean> {
  const mangaPath = path.join(home, 'library', mangaTitle);
  if (!fs.existsSync(mangaPath)) {
    return false;
  }

  const chapters = fs.readdirSync(mangaPath, { withFileTypes: true });
  const chapterDir = chapters.find((entry) => {
    const match = entry.name.match(new RegExp(`^ch?[-.]?${chapterNumber}(\\.|$)`, 'i'));
    return entry.isDirectory() && match !== null;
  });

  if (!chapterDir) {
    return false;
  }

  const chapterPath = path.join(mangaPath, chapterDir.name);
  const files = fs.readdirSync(chapterPath);
  return files.length > 0;
}

export async function dbHasDownloadJobs(home: string, count: number): Promise<boolean> {
  const dbPath = path.join(home, 'kiyomi.db');
  if (!fs.existsSync(dbPath)) {
    return count === 0;
  }

  const db = sqlite3(dbPath);
  try {
    const stmt = db.prepare('SELECT COUNT(*) as cnt FROM download_jobs WHERE status = ?');
    const result = stmt.get('completed') as { cnt: number };
    return result.cnt === count;
  } finally {
    db.close();
  }
}
