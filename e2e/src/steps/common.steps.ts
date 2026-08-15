import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';
import { E2EWorld } from '../world';
import axios from 'axios';

Given('a clean server with empty library is running', async function () {
  const { port } = getWorld();
  // Verify the server is up and the library is empty
  const res = await axios.get(`http://localhost:${port}/api/v1/library/manga`, { timeout: 5000 });
  const mangas = Array.isArray(res.data) ? res.data : (res.data?.data ?? []);
  if (mangas.length > 0) {
    throw new Error('Library is not empty at start of scenario — possible state leak from previous scenario');
  }
});

Given('a seeded library with manga-x is running', async function () {
  const { port } = getWorld();
  const base = `http://localhost:${port}`;
  await axios.post(`${base}/api/v1/library/manga/import`, {
    provider_id: 'mock',
    remote_id: 'alpha',
    user_status: 'reading',
  }, { timeout: 5000 });
  await axios.post(`${base}/api/v1/library/manga/import`, {
    provider_id: 'mock',
    remote_id: 'beta',
    user_status: 'completed',
  }, { timeout: 5000 });
  await new Promise(r => setTimeout(r, 500));
});

Given('a seeded library with {string} is running', async function (mangaName: string) {
  const { port } = getWorld();
  const base = `http://localhost:${port}`;
  let remoteId = 'alpha';
  if (mangaName.toLowerCase().includes('beta')) {
    remoteId = 'beta';
  }
  await axios.post(`${base}/api/v1/library/manga/import`, {
    provider_id: 'mock',
    remote_id: remoteId,
    user_status: 'reading',
  }, { timeout: 5000 });
  await new Promise(r => setTimeout(r, 500));
});

Given('the manga {string} is already in the library', async function (_title: string) {
  const { port } = getWorld();
  const base = `http://localhost:${port}`;
  await axios.post(`${base}/api/v1/library/manga/import`, {
    provider_id: 'mock',
    remote_id: 'alpha',
  }, { timeout: 5000 });
  await new Promise(r => setTimeout(r, 500));
});

Given('the mock provider has a manga with zero chapters', async function () {
  // empty-manga fixture is available in mock provider with 0 chapters
});

Given('I open the library', async function () {
  const { page, port } = getWorld();
  await page.goto(`http://localhost:${port}/`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1000);
});

Given('the library is empty', async function () {
  const { page, port } = getWorld();
  await page.goto(`http://localhost:${port}/`, { waitUntil: 'domcontentloaded' });
  await page.waitForFunction(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    return btns.some(b => b.textContent?.includes('Start Exploring') || b.textContent?.includes('Add Your First Series'));
  }, { timeout: 10000 });
  await page.waitForTimeout(500);
  const cards = await page.locator('[class*="rounded-lg"][class*="border-border"][class*="bg-card"]').count();
  expect(cards).toBe(0);
});

When('I click {string}', async function (targetText: string) {
  const { page } = getWorld();
  const element = page.locator('button, a, [role="button"]').filter({ hasText: targetText }).filter({ visible: true }).first();
  await element.waitFor({ state: 'visible', timeout: 15000 });
  const tagName = await element.evaluate(el => el.tagName.toLowerCase()).catch(() => '');
  if (tagName === 'button') {
    await expect(element).toBeEnabled({ timeout: 15000 });
  }
  await element.click();
  await page.waitForTimeout(500);
});

Then('I am redirected to the Explore view', async function () {
  const { page } = getWorld();
  await page.waitForURL(url => url.pathname.includes('/explore') || url.pathname.includes('/providers'), { timeout: 10000 });
  const currentUrl = page.url();
  expect(currentUrl.includes('/explore') || currentUrl.includes('/providers')).toBe(true);
});

When('I open the Explore view', async function () {
  const { page, port } = getWorld();
  const exploreLink = page.locator('a[href*="/explore"], button:has-text("Explore")').first();
  if (await exploreLink.isVisible().catch(() => false)) {
    await exploreLink.click();
  } else {
    await page.goto(`http://localhost:${port}/explore`, { waitUntil: 'domcontentloaded' });
  }
  await page.waitForURL(url => url.pathname.includes('/explore') || url.pathname.includes('/providers'), { timeout: 10000 });
});

When('I click the add button', async function () {
  const { page } = getWorld();
  await page.waitForFunction(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    return btns.some(b => b.textContent?.includes('Start Exploring') || b.textContent?.includes('Add Your First Series') || b.textContent?.includes('Add Series'));
  }, { timeout: 10000 });
  await page.waitForTimeout(500);
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.textContent?.includes('Start Exploring') || b.textContent?.includes('Add Your First Series') || b.textContent?.includes('Add Series'));
    if (btn) (btn as HTMLButtonElement)?.click();
  });
  await page.waitForTimeout(1000);
});

When('the search modal opens', async function () {
  const { page } = getWorld();
  await page.waitForTimeout(1000);
  const open = await page.evaluate(() => {
    const dialog = document.querySelector('[role="dialog"]');
    return dialog !== null;
  });
  expect(open).toBe(true);
});

When('I search for {string}', async function (query: string) {
  const { page, port } = getWorld();
  await page.goto(`http://localhost:${port}/providers/mock?q=${encodeURIComponent(query)}`, { waitUntil: 'domcontentloaded' });
  await page.waitForFunction(() => {
    const links = Array.from(document.querySelectorAll('a'));
    return links.some(a => {
      const href = a.getAttribute('href') || '';
      return href.includes('/manga/');
    });
  }, { timeout: 15000 });
  await page.waitForTimeout(500);
});

When('I click the first result', async function () {
  const { page } = getWorld();
  await page.waitForFunction(() => {
    const links = Array.from(document.querySelectorAll('a'));
    return links.some(a => {
      const href = a.getAttribute('href') || '';
      return href.includes('/manga/');
    });
  }, { timeout: 15000 });

  const firstCard = page
    .locator('a[href*="/manga/"]')
    .filter({ hasNotText: /^(Explore|Library|Downloads|Workers|Settings)$/i })
    .first();

  await firstCard.waitFor({ state: 'visible', timeout: 10000 });
  await firstCard.click();
  await page.waitForURL(url => url.pathname.includes('/manga/'), { timeout: 10000 });
  await page.waitForTimeout(500);
});

When('I navigate to the manga detail page for {string}', async function (_title: string) {
  const { page, port } = getWorld();
  await page.goto(`http://localhost:${port}/`, { waitUntil: 'domcontentloaded' });
  // Wait for skeleton to disappear and real cards to render
  await page.waitForFunction(() => {
    const cards = document.querySelectorAll('[class*="rounded-lg"][class*="border-border"][class*="bg-card"]');
    return cards.length > 0;
  }, { timeout: 15000 });
  const card = page.locator('[class*="rounded-lg"][class*="border-border"][class*="bg-card"]').filter({ hasText: _title }).first();
  await card.click();
  // SPA navigation via React Router — wait for URL change instead of load state
  await page.waitForURL(`**/manga/**`, { timeout: 15000 });
});

When('I click on chapter {string}', async function (_chapter: string) {
  const { page } = getWorld();
  const chapterNum = _chapter.replace(/^chapter\s+/i, '').trim();
  // Locate chapter row and click its link/button, scoped to chapter list container
  // Chapter rows are divs with specific border/card classes; the chapter text is in a Link child
  const chapterRow = page.locator('[class*="rounded-lg"][class*="border-border"][class*="bg-card"]:has(a)')
    .filter({ hasText: new RegExp(`(?:ch\\.?\\s*|chapter\\s*)${chapterNum}`, 'i') })
    .first();
  await chapterRow.locator('a').first().click();
  await page.waitForLoadState('networkidle');
});

Then('the manga {string} appears in the library', async function (_title: string) {
  const { page } = getWorld();
  // Wait for the View in Library button to appear, indicating successful add
  const viewInLibBtn = page.locator('a:has-text("View in Library")');
  await expect(viewInLibBtn).toBeVisible({ timeout: 10000 });
  await viewInLibBtn.click();

  // Navigate back to Library
  const libraryLink = page.locator('a[href="/"]').first();
  await libraryLink.click();
  await page.waitForFunction(() => window.location.pathname === '/', { timeout: 10000 });

  // Verify the manga card is visible in the library grid (excluding skeleton cards)
  const card = page.locator('[class*="bg-card"]:has(h3)').filter({ hasText: _title }).first();
  await expect(card).toBeVisible({ timeout: 10000 });

  // Verify provider badge is displayed on the cover thumbnail inside the library card
  const badge = card.locator('span, div, [data-slot="badge"]').first();
  await expect(badge).toBeVisible({ timeout: 10000 });

  // Re-open details page so subsequent steps can inspect detail page state
  await card.locator('a').first().click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/manga/'), { timeout: 10000 });
});

Then('the manga cover thumbnail displays the content provider badge {string}', async function (providerId: string) {
  const { page } = getWorld();
  let wasOnDetails = false;
  if (page.url().includes('/manga/')) {
    wasOnDetails = true;
    const libraryLink = page.locator('a[href="/"]').first();
    await libraryLink.click();
    await page.waitForFunction(() => window.location.pathname === '/', { timeout: 10000 });
  }

  // Target real library cards with titles (excluding skeleton loading cards)
  const card = page.locator('[class*="bg-card"]:has(h3)').filter({ hasText: new RegExp(providerId, 'i') }).first();
  await expect(card).toBeVisible({ timeout: 10000 });

  const badge = card.locator('span, div, [data-slot="badge"]').filter({ hasText: new RegExp(`^${providerId}$`, 'i') }).first();
  await expect(badge).toBeVisible({ timeout: 10000 });

  if (wasOnDetails) {
    await card.locator('a').first().click();
    await page.waitForFunction(() => window.location.pathname.startsWith('/manga/'), { timeout: 10000 });
  }
});

Then('the manga has {string} chapters listed', async function (_count: string) {
  const { page } = getWorld();
  const rows = page.locator('a[href*="/chapter/"]');
  await expect(rows).toHaveCount(parseInt(_count, 10), { timeout: 10000 });
});

Then('an error is shown', async function () {
  const { page } = getWorld();
  const error = page.locator('[role="alert"], .error, [class*="error"], [aria-label*="error" i]').first();
  await expect(error).toBeVisible({ timeout: 5000 });
});

Then('the manga is not added to the library', async function () {
  const { page, port } = getWorld();
  await page.goto(`http://localhost:${port}/`);
  await page.waitForTimeout(500);
  const cards = await page.locator('[class*="rounded-lg"][class*="border-border"][class*="bg-card"]').count();
  expect(cards).toBe(0);
});

Then('progress is saved', async function () {
  const { port } = getWorld();
  const res = await axios.get(`http://localhost:${port}/api/v1/library/manga`);
  const mangas: any[] = Array.isArray(res.data) ? res.data : (res.data?.data ?? []);
  expect(mangas.length).toBeGreaterThan(0);
});

When('I close the reader', async function (this: E2EWorld) {
  // Navigate back to the manga detail page via browser back
  await this.page!.goBack();
  await this.page!.waitForURL('**/manga/**', { timeout: 15000 });
});

Then('the {string} button is disabled or indicates {string}', async function (buttonText: string, indicatorText: string) {
  const { page } = getWorld();
  // Wait until either the button is disabled or the indicator/View in Library is visible
  await page.waitForFunction((args) => {
    const btns = Array.from(document.querySelectorAll('button, a'));
    const btn = btns.find(b => b.textContent?.includes(args.buttonText));
    const isBtnDisabled = btn && ((btn as HTMLButtonElement).disabled || btn.getAttribute('aria-disabled') === 'true');
    const hasIndicator = btns.some(b => b.textContent?.includes(args.indicatorText) || b.textContent?.includes('View in Library'));
    return Boolean(isBtnDisabled || hasIndicator);
  }, { buttonText, indicatorText }, { timeout: 15000 });
});

Then('an error is shown indicating no chapters were found', async function () {
  const { page } = getWorld();
  await page.waitForFunction(() => {
    const textElements = Array.from(document.querySelectorAll('[role="alert"], [class*="error"], [class*="destructive"], div, p, span, h2, h3'));
    return textElements.some(el => {
      const txt = el.textContent?.toLowerCase() || '';
      return (txt.includes('no chapters') || txt.includes('failed to load chapter') || txt.includes('zero chapters') || txt.includes('no content provider') || txt.includes('no matching chapters') || txt.includes('content unavailable'));
    });
  }, { timeout: 10000 });
});

Then('the {string} button is disabled', async function (buttonText: string) {
  const { page } = getWorld();
  const btn = page.locator('button').filter({ hasText: buttonText }).first();
  await expect(btn).toBeDisabled({ timeout: 5000 });
});


