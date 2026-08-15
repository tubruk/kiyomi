import { When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';

When('I click on the manga {string} in the library', async function (title: string) {
  const { page } = getWorld();
  // Wait for real library manga cards to render
  await page.waitForFunction(() => {
    const cards = document.querySelectorAll('[class*="bg-card"]');
    return cards.length > 0;
  }, { timeout: 15000 });

  const card = page
    .locator('[class*="bg-card"]')
    .filter({ hasText: title })
    .first();

  await card.waitFor({ state: 'visible', timeout: 15000 });
  await card.locator('a').first().click();
  await page.waitForURL(url => url.pathname.includes('/manga/'), { timeout: 15000 });
});

Then('I am on the library manga details page for {string}', async function (title: string) {
  const { page } = getWorld();
  await page.waitForURL(url => url.pathname.includes('/manga/'), { timeout: 15000 });
  const heading = page.locator('h1').filter({ hasText: title }).first();
  await expect(heading).toBeVisible({ timeout: 15000 });
});

Then('I see the manga cover image', async function () {
  const { page } = getWorld();
  // Cover image in hero card
  const img = page.locator('.aspect-\\[2\\/3\\] img, img[alt]').first();
  await expect(img).toBeVisible({ timeout: 10000 });
  const src = await img.getAttribute('src');
  expect(src).toBeTruthy();
});

Then('I see the manga title {string}', async function (title: string) {
  const { page } = getWorld();
  const heading = page.locator('h1').filter({ hasText: title }).first();
  await expect(heading).toBeVisible({ timeout: 10000 });
});

Then('I see the manga aliases containing {string}', async function (aliasText: string) {
  const { page } = getWorld();
  const aliasElement = page
    .locator('p, div, span')
    .filter({ hasText: 'Aliases:' })
    .filter({ hasText: aliasText })
    .first();
  await expect(aliasElement).toBeVisible({ timeout: 10000 });
});

Then('I see the merged author and artist {string}', async function (authorArtist: string) {
  const { page } = getWorld();
  const authorArtistElement = page
    .locator('div, span, p')
    .filter({ hasText: 'Author / Artist:' })
    .filter({ hasText: authorArtist })
    .first();
  await expect(authorArtistElement).toBeVisible({ timeout: 10000 });
});

Then('I see the manga tags including {string}', async function (tag: string) {
  const { page } = getWorld();
  const tagBadge = page
    .locator('[data-slot="badge"], [class*="badge"], span, div')
    .filter({ hasText: new RegExp(`^${tag}$`, 'i') })
    .first();
  await expect(tagBadge).toBeVisible({ timeout: 10000 });
});

Then('I see the manga synopsis containing {string}', async function (synopsisText: string) {
  const { page } = getWorld();
  const synopsisEl = page.locator('div, p').filter({ hasText: synopsisText }).first();
  await expect(synopsisEl).toBeVisible({ timeout: 10000 });
});

Then('I see the manga reading status is {string}', async function (statusText: string) {
  const { page } = getWorld();
  const selectTrigger = page
    .locator('[data-slot="select-trigger"], button[role="combobox"]')
    .filter({ hasText: new RegExp(statusText, 'i') })
    .first();
  await expect(selectTrigger).toBeVisible({ timeout: 10000 });
});
