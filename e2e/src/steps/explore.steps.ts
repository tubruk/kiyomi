import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';

Then('I see the catalog header {string}', async function (headerText: string) {
  const { page } = getWorld();
  const heading = page.locator('h1').filter({ hasText: headerText }).first();
  await expect(heading).toBeVisible({ timeout: 10000 });
});

When('I select the provider {string}', async function (providerName: string) {
  const { page } = getWorld();
  const selectTrigger = page.locator('[data-slot="select-trigger"], button[role="combobox"]').filter({ visible: true }).first();
  await selectTrigger.waitFor({ state: 'visible', timeout: 15000 });
  await selectTrigger.click();

  const option = page.locator('[data-slot="select-item"], [role="option"]').filter({ hasText: providerName }).filter({ visible: true }).first();
  await option.waitFor({ state: 'visible', timeout: 15000 });
  await option.click();
  await page.waitForTimeout(500);
});

Then('the catalog for {string} is displayed', async function (providerName: string) {
  const { page } = getWorld();
  await page.waitForFunction((pName) => {
    const trigger = document.querySelector('[data-slot="select-trigger"], [role="combobox"]');
    return trigger?.textContent?.includes(pName) || false;
  }, providerName, { timeout: 10000 });
});

Then('I see manga titles from the provider', async function () {
  const { page } = getWorld();
  const card = page.locator('a[href*="/providers/"], a[href*="/manga/"]').filter({ visible: true }).first();
  await card.waitFor({ state: 'visible', timeout: 15000 });
});

When('I search for {string} in the catalog', async function (query: string) {
  const { page } = getWorld();
  const searchInput = page.locator('input[placeholder*="Search catalog titles"]').first();
  await searchInput.waitFor({ state: 'visible', timeout: 15000 });
  await searchInput.fill(query);
  await searchInput.press('Enter');
  await page.waitForTimeout(600);
});

Then('I see the manga {string} in search results', async function (title: string) {
  const { page } = getWorld();
  const card = page
    .locator('a[href*="/providers/"], a[href*="/manga/"], [class*="bg-card"]')
    .filter({ hasText: title })
    .filter({ visible: true })
    .first();
  await card.waitFor({ state: 'visible', timeout: 15000 });
  await expect(card).toBeVisible({ timeout: 10000 });
});

When('I click the manga {string}', async function (title: string) {
  const { page } = getWorld();
  const card = page.locator('a[href*="/providers/"], a[href*="/manga/"]').filter({ hasText: title }).filter({ visible: true }).first();
  await card.waitFor({ state: 'visible', timeout: 15000 });
  await card.click();
  await page.waitForURL(url => url.pathname.includes('/manga/'), { timeout: 15000 });
});

Then('I am on the remote manga details page for {string}', async function (title: string) {
  const { page } = getWorld();
  await page.waitForURL(url => url.pathname.includes('/manga/'), { timeout: 15000 });
  const heading = page.locator('h1, h2').filter({ hasText: title }).filter({ visible: true }).first();
  await expect(heading).toBeVisible({ timeout: 15000 });
});

Then('I see the manga details including synopsis and chapter list', async function () {
  const { page } = getWorld();
  const chaptersSection = page.locator('div, section').filter({ hasText: 'Chapters' }).filter({ visible: true }).first();
  await expect(chaptersSection).toBeVisible({ timeout: 15000 });
});

Given('the provider has a manga {string} explicitly marked as unavailable', async function (_title: string) {
  // manga-unavailable-manga fixture is available in mock provider with availability: unavailable
});

Then('the manga card for {string} displays an {string} badge', async function (title: string, badgeText: string) {
  const { page } = getWorld();
  const card = page
    .locator('a[href*="/providers/"], a[href*="/manga/"], [class*="bg-card"]')
    .filter({ hasText: title })
    .filter({ visible: true })
    .first();
  await card.waitFor({ state: 'visible', timeout: 15000 });
  const badge = card
    .locator('[data-slot="badge"], [class*="badge"], span, div')
    .filter({ hasText: badgeText })
    .first();
  await expect(badge).toBeVisible({ timeout: 10000 });
});

Then('I see the manga details including synopsis and metadata', async function () {
  const { page } = getWorld();
  const heading = page.locator('h1, h2').filter({ visible: true }).first();
  await expect(heading).toBeVisible({ timeout: 15000 });
  const metaOrSynopsis = page.locator('div, p').filter({ hasText: /synopsis|mock author|mock artist|author|description/i }).first();
  await expect(metaOrSynopsis).toBeVisible({ timeout: 15000 });
});

Then('the chapter list section displays {string}', async function (expectedText: string) {
  const { page } = getWorld();
  const section = page.locator('div, section').filter({ hasText: expectedText }).filter({ visible: true }).first();
  await expect(section).toBeVisible({ timeout: 15000 });
});

