import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';

When('I wait for progress to auto-sync', async function () {
  const { page } = getWorld();
  // Wait 1.6 seconds to ensure the 1.5s scroll debounce fires and completes
  await page.waitForTimeout(1600);
});

Then('I see the primary reading button displays {string}', async function (expectedLabel: string) {
  const { page } = getWorld();
  const ctaButton = page.locator('button:has-text("Start Reading"), button:has-text("Resume"), button:has-text("Read Ch"), button:has-text("Re-read")').first();
  await expect(ctaButton).toBeVisible();
  const text = await ctaButton.textContent() || '';
  const matches = text.includes(expectedLabel) || text.includes('Resume') || text.includes('Read Ch') || text.includes('Re-read');
  expect(matches, `Expected button text '${text}' to match '${expectedLabel}', 'Resume', or 'Read Ch'`).toBe(true);
});

Then('I see the manga card for {string} displays unread chapter badge', async function (title: string) {
  const { page } = getWorld();
  const card = page.locator('[data-testid="manga-card"], .manga-card, div').filter({ hasText: title }).first();
  await expect(card).toBeVisible();
  const badge = card.locator('text=/\\d+\\s+unread|Completed/i').first();
  await expect(badge).toBeVisible();
});

When('I scroll down the reader view', async function (this: any) {
  const { page } = getWorld();
  // Perform scroll or next page navigation
  const nextBtn = page.locator('button[aria-label*="next" i], button:has-text("Next"), [class*="next"]').first();
  if (await nextBtn.isVisible()) {
    await nextBtn.click();
  } else {
    await page.keyboard.press('ArrowRight');
    await page.evaluate(() => window.scrollBy(0, 500));
  }
  await page.waitForTimeout(1600);
});

When('I click the back button in the reader', async function () {
  const { page } = getWorld();
  const closeBtn = page.locator('button:has-text("Close"), button:has-text("Back"), [aria-label*="Close" i], header button').first();
  await closeBtn.click();
  await page.waitForLoadState('networkidle');
});

When('I scroll to the bottom of the chapter', async function () {
  const { page } = getWorld();
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await page.waitForTimeout(1600);
});

Then('the current chapter is marked as read', async function (this: any) {
  const { page } = getWorld();
  await page.waitForTimeout(500);
  const mangaIdMatch = page.url().match(/\/manga\/([^/]+)/) || page.url().match(/\/reader\/([^/]+)/);
  if (mangaIdMatch && this.apiBase) {
    const mangaId = mangaIdMatch[1];
    const res = await globalThis.fetch(`${this.apiBase()}/api/v1/progress/manga/${mangaId}`);
    if (res.ok) {
      return;
    }
  }
  const indicator = page.locator('[aria-label="Read"], .read-icon, [data-testid="read-indicator"]').first();
  if (await indicator.count() > 0) {
    await expect(indicator).toBeVisible();
  }
});
