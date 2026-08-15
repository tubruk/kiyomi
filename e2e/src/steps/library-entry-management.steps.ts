import { When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';

When('I select the reading status {string}', async function (statusText: string) {
  const { page } = getWorld();
  const section = page.locator('div').filter({ hasText: /^Reading Status/i }).first();
  const trigger = section.locator('[data-slot="select-trigger"], button[role="combobox"]').first();
  await trigger.waitFor({ state: 'visible', timeout: 10000 });
  await trigger.click();

  const option = page
    .locator('[role="option"]')
    .filter({ hasText: new RegExp(`^${statusText}$`, 'i') })
    .first();
  await option.waitFor({ state: 'visible', timeout: 10000 });
  await option.click();

  await expect(trigger).toContainText(statusText, { timeout: 10000 });
});

When('I toggle the favorite button', async function () {
  const { page } = getWorld();
  const favBtn = page.locator('button[aria-label*="Favorite" i]').first();
  await favBtn.waitFor({ state: 'visible', timeout: 10000 });
  const prevLabel = await favBtn.getAttribute('aria-label');
  await favBtn.click();
  await expect(favBtn).not.toHaveAttribute('aria-label', prevLabel || '', { timeout: 10000 });
});

Then('the manga is not marked as favorite', async function () {
  const { page } = getWorld();
  const addFavBtn = page.locator('button[aria-label="Add to Favorites"]').first();
  await expect(addFavBtn).toBeVisible({ timeout: 10000 });
});

Then('the manga is marked as favorite', async function () {
  const { page } = getWorld();
  const removeFavBtn = page.locator('button[aria-label="Remove from Favorites"]').first();
  await expect(removeFavBtn).toBeVisible({ timeout: 10000 });
});

When('I set the rating to {string} stars', async function (starsCount: string) {
  const { page } = getWorld();
  const starBtn = page.locator(`button[title*="${starsCount} Stars"]`).first();
  await starBtn.waitFor({ state: 'visible', timeout: 10000 });
  await starBtn.click();
  const expectedScore = `${parseInt(starsCount, 10) * 2}/10`;
  const ratingEl = page.locator('span').filter({ hasText: expectedScore }).first();
  await expect(ratingEl).toBeVisible({ timeout: 10000 });
});

Then('I see the manga rating is {string}', async function (ratingText: string) {
  const { page } = getWorld();
  const ratingEl = page
    .locator('span')
    .filter({ hasText: new RegExp(`^${ratingText}$`, 'i') })
    .first();
  await expect(ratingEl).toBeVisible({ timeout: 10000 });
});

Then('I see no personal notes added', async function () {
  const { page } = getWorld();
  const emptyNotes = page.locator('span, div, p').filter({ hasText: 'No notes added yet.' }).first();
  await expect(emptyNotes).toBeVisible({ timeout: 10000 });
});

When('I set the personal notes to {string}', async function (noteText: string) {
  const { page } = getWorld();
  const editBtn = page.locator('button').filter({ hasText: /^(\+ Add Note|Edit)$/i }).first();
  await editBtn.waitFor({ state: 'visible', timeout: 10000 });
  await editBtn.click();

  const textarea = page.locator('textarea[placeholder="Add private personal notes..."]').first();
  await textarea.waitFor({ state: 'visible', timeout: 10000 });
  await textarea.fill(noteText);

  const saveBtn = page.locator('button').filter({ hasText: /^Save$/i }).first();
  await saveBtn.click();

  await textarea.waitFor({ state: 'detached', timeout: 10000 });
});

Then('I see the manga personal notes contain {string}', async function (noteText: string) {
  const { page } = getWorld();
  const noteEl = page.locator('div, p').filter({ hasText: noteText }).first();
  await expect(noteEl).toBeVisible({ timeout: 10000 });
});

Then('I am on the library page', async function () {
  const { page } = getWorld();
  await page.waitForFunction(() => window.location.pathname === '/', { timeout: 15000 });
});
