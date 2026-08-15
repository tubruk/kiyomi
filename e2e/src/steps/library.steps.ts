import { When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { E2EWorld } from '../world';
import { LibraryPage } from '../pages/LibraryPage';

When('I add manga {string} from search', async function (this: E2EWorld, title: string) {
  const libraryPage = new LibraryPage(this.page!);
  await libraryPage.goto();
  await libraryPage.openAddModal();
  await libraryPage.searchFor(title);
  await libraryPage.addMangaFromResults(title);
});

Then('manga {string} is in library', async function (this: E2EWorld, title: string) {
  const libraryPage = new LibraryPage(this.page!);
  await libraryPage.goto();
  const card = await libraryPage.getMangaCard(title);
  await expect(card).toBeVisible();
});

Then('I see empty library', async function (this: E2EWorld) {
  const libraryPage = new LibraryPage(this.page!);
  await libraryPage.goto();
  const cards = await libraryPage.getMangaCards();
  expect(cards.length).toBe(0);
});
