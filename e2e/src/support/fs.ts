import * as fs from 'fs';
import * as path from 'path';

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
  chapterTitle: string
): Promise<boolean> {
  const libraryPath = path.join(home, 'library');
  if (!fs.existsSync(libraryPath)) {
    return false;
  }

  // Find manga folder matching mangaTitle
  const mangaEntries = fs.readdirSync(libraryPath, { withFileTypes: true });
  const mangaDir = mangaEntries.find(
    (entry) => entry.isDirectory() && entry.name === mangaTitle
  );

  if (!mangaDir) {
    return false;
  }

  // Find chapter folder containing pages
  const mangaPath = path.join(libraryPath, mangaDir.name);
  const chapterEntries = fs.readdirSync(mangaPath, { withFileTypes: true });
  const chapterDir = chapterEntries.find(
    (entry) => entry.isDirectory() && entry.name === chapterTitle
  );

  if (!chapterDir) {
    return false;
  }

  const chapterPath = path.join(mangaPath, chapterDir.name);
  const files = fs.readdirSync(chapterPath);
  return files.length > 0;
}
