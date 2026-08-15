import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';
import { E2EWorld } from '../world';
import { LibraryPage } from '../pages/LibraryPage';
import { MangaDetailPage } from '../pages/MangaDetail';
import { ReaderPage } from '../pages/Reader';

When('I read chapter {int} of {string}', async function (this: E2EWorld, chapterNumber: number, mangaTitle: string) {
  const libraryPage = new LibraryPage(this.page!);
  await libraryPage.goto();
  const card = libraryPage.getMangaCard(mangaTitle);
  await card.click();

  const detailPage = new MangaDetailPage(this.page!);
  await detailPage.goto((this.page!.url().match(/\/manga\/([^/]+)/) ?? [])[1] ?? '');

  const chapterList = await detailPage.getChapterList();
  const chapter = chapterList[chapterNumber - 1];
  await chapter.locator('a, button').first().click();

  const readerPage = new ReaderPage(this.page!);
  await this.page!.waitForLoadState('networkidle');
});

Then('the chapter progress is saved at page {int}', async function (this: E2EWorld, pageNumber: number) {
  const mangaId = (this.page!.url().match(/\/reader\/([^/]+)/) ?? [])[1] ?? '';
  const url = `${this.apiBase()}/api/v1/progress/manga/${mangaId}`;
  const response = await globalThis.fetch(url);
  const data = await response.json() as { mangaId: string; progress?: { lastReadPage: number }; chapterStates?: Array<{ chapterId: string; status: string; lastPage: number }> };
  // Progress may be at manga level or chapter level
  const savedPage = data.progress?.lastReadPage;
  expect(savedPage).toBe(pageNumber);
});

Then('I am on page {int} of chapter {int}', async function (pageVal: number, _chapter: number) {
  const { page } = getWorld();
  const readerPage = new ReaderPage(page);
  const currentPage = await readerPage.getCurrentPage();
  expect(currentPage).toBe(pageVal);
});

// ─── New steps for 02-read-manga.feature ─────────────────────────────────────

Given('I see the chapter list with at least {int} chapters', async function (count: number) {
  const { page } = getWorld();
  const detailPage = new MangaDetailPage(page);
  const locators = await detailPage.getChapterList();
  expect(locators.length).toBeGreaterThanOrEqual(count);
});

Then('the reader opens', async function () {
  const { page } = getWorld();
  await page.waitForURL((url) => url.pathname.includes('/chapter/'), { timeout: 15000 });
  await page.waitForSelector('[data-testid="reader-content"], [data-testid="page-indicator"], .page-indicator, [class*="reader"]', { state: 'visible', timeout: 10000 });
});



When('I navigate to page {int}', async function (this: E2EWorld, targetPage: number) {
  const readerPage = new ReaderPage(this.page!);
  const current = await readerPage.getCurrentPage();
  for (let i = current; i < targetPage; i++) {
    await readerPage.navigateNext();
  }
});

Given('I read chapter {int} to page {int}', async function (this: E2EWorld, chapterNumber: number, pageNum: number) {
  const mangaId = (this.page!.url().match(/\/manga\/([^/]+)/) ?? [])[1] ?? '';
  const apiBase = this.apiBase();

  // Get chapter list to find the chapterId
  const chaptersRes = await globalThis.fetch(`${apiBase}/api/v1/library/manga/${mangaId}/chapters`);
  const chaptersData = await chaptersRes.json() as { chapters: Array<{ id: string; number: number }> };
  const chapter = chaptersData.chapters.find(c => c.number === chapterNumber);
  if (!chapter) throw new Error(`Chapter ${chapterNumber} not found for manga ${mangaId}`);

  const chapterId = chapter.id;

  // PUT /api/v1/progress/manga/:mangaId sets both manga-level and chapter-level progress in one call
  await globalThis.fetch(`${apiBase}/api/v1/progress/manga/${mangaId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ chapterId, chapterNumber, page: pageNum, totalPages: 8 }),
  });
});

When('I close and reopen the manga', async function (this: E2EWorld) {
  // Navigate back to library home
  await this.page!.goto(`${this.apiBase()}`);
  await this.page!.waitForLoadState('networkidle');

  // Click on Alpha Manga card
  const libraryPage = new LibraryPage(this.page!);
  const card = libraryPage.getMangaCard('Alpha Manga');
  await card.click();
  await this.page!.waitForLoadState('networkidle');
});

Then('chapter {int} shows {string} status', async function (this: E2EWorld, chapterNumber: number, status: string) {
  const detailPage = new MangaDetailPage(this.page!);
  const locators = await detailPage.getChapterList();
  const chapterRow = locators[chapterNumber - 1];

  if (status === 'in progress') {
    // Look for amber/yellow in-progress indicator: dot, badge, or page text
    const hasAmberDot = await chapterRow.locator('span.rounded-full.bg-amber-500, [class*="bg-amber"]').count() > 0;
    const hasAmberText = await chapterRow.locator('span.text-amber-500, [class*="text-amber"]').count() > 0;
    const hasInProgressText = await chapterRow.getByText(/in\s*progress/i).count() > 0;
    const hasPageProgress = await chapterRow.getByText(/p\.\s*\d+/i).count() > 0;

    const found = hasAmberDot || hasAmberText || hasInProgressText || hasPageProgress;
    expect(found, `Chapter ${chapterNumber} should show in-progress status`).toBe(true);
  } else if (status === 'completed') {
    const hasCompleted = await chapterRow.getByText(/completed|✓|done/i).count() > 0;
    expect(hasCompleted, `Chapter ${chapterNumber} should show completed status`).toBe(true);
  } else {
    throw new Error(`Unknown status: ${status}`);
  }
});

Then('continuing from chapter {int} starts at page {int}', async function (this: E2EWorld, chapterNumber: number, expectedPage: number) {
  const detailPage = new MangaDetailPage(this.page!);
  const locators = await detailPage.getChapterList();
  const chapterRow = locators[chapterNumber - 1];

  // Click on the chapter row (should be a link)
  await chapterRow.locator('a, button').first().click();
  const page = this.page!;
  await page.waitForURL((url) => url.pathname.includes('/chapter/'), { timeout: 15000 });

  const readerPage = new ReaderPage(this.page!);
  const currentPage = await readerPage.getCurrentPage();
  expect(currentPage).toBe(expectedPage);
});
