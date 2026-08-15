import { When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { E2EWorld } from '../world';
import { LibraryPage } from '../pages/LibraryPage';
import { MangaDetailPage } from '../pages/MangaDetail';
import { chapterDownloaded } from '../support/fs';

When('I download chapter {int} of {string}', async function (this: E2EWorld, chapterNumber: number, mangaTitle: string) {
  const libraryPage = new LibraryPage(this.page!);
  await libraryPage.goto();
  const card = await libraryPage.getMangaCard(mangaTitle);
  await card.click();

  const detailPage = new MangaDetailPage(this.page!);
  await detailPage.downloadChapter(chapterNumber);
});

Then('chapter {int} is downloaded', async function (this: E2EWorld, chapterNumber: number) {
  const mangaId = (this.page!.url().match(/\/manga\/([^/]+)/) ?? [])[1] ?? '';
  const downloaded = await chapterDownloaded(this.home, mangaId, chapterNumber);
  expect(downloaded).toBe(true);
});
