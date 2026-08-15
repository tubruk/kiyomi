import { When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';

When('I toggle detailed metadata', async function () {
  const { page } = getWorld();
  const toggleBtn = page
    .locator('button')
    .filter({ hasText: /Show Detailed Metadata|Hide Details|Detailed Metadata/i })
    .first();
  await toggleBtn.waitFor({ state: 'visible', timeout: 10000 });
  await toggleBtn.click();
  await page.waitForTimeout(300);
});

Then('I see the manga reading mode is {string}', async function (expectedMode: string) {
  const { page } = getWorld();
  const section = page
    .locator('div')
    .filter({ hasText: /Reading Mode/i })
    .first();
  await expect(section).toBeVisible({ timeout: 10000 });
  await expect(section).toContainText(expectedMode, { ignoreCase: true });
});

When('I open the edit metadata dialog', async function () {
  const { page } = getWorld();
  const menuTrigger = page
    .locator('button[aria-label="Metadata options"], button[title="Metadata options"]')
    .first();
  if (await menuTrigger.isVisible().catch(() => false)) {
    await menuTrigger.click();
    const menuItem = page
      .locator('[role="menuitem"], button')
      .filter({ hasText: /Edit Metadata/i })
      .first();
    await menuItem.waitFor({ state: 'visible', timeout: 10000 });
    await menuItem.click();
  } else {
    const editBtn = page.locator('button').filter({ hasText: /Edit Metadata/i }).first();
    await editBtn.waitFor({ state: 'visible', timeout: 10000 });
    await editBtn.click();
  }

  await page.locator('[role="dialog"]').waitFor({ state: 'visible', timeout: 10000 });
});

When('I select the reading mode option {string}', async function (modeOption: string) {
  const { page } = getWorld();
  const dialog = page.locator('[role="dialog"]').first();
  const trigger = dialog
    .locator('div:has(label:has-text("Reading Mode")) [data-slot="select-trigger"], [data-slot="select-trigger"]')
    .first();
  await trigger.waitFor({ state: 'visible', timeout: 10000 });
  await trigger.click();

  const option = page
    .locator('[data-slot="select-item"], [role="option"]')
    .filter({ hasText: modeOption })
    .first();
  await option.waitFor({ state: 'visible', timeout: 10000 });
  await option.click();
  await page.waitForTimeout(300);
});

When('I save the metadata changes', async function () {
  const { page } = getWorld();
  const dialog = page.locator('[role="dialog"]').first();
  const saveBtn = dialog.locator('button').filter({ hasText: /Save Changes/i }).first();
  await saveBtn.waitFor({ state: 'visible', timeout: 10000 });
  await saveBtn.click();
  await dialog.waitFor({ state: 'detached', timeout: 15000 });
  await page.waitForTimeout(500);
});

Then('the reader layout is rendered as continuous longstrip scroll', async function () {
  const { page } = getWorld();
  const readerContent = page.locator('[data-testid="reader-content"]').first();
  await expect(readerContent).toBeVisible({ timeout: 10000 });
  await expect(readerContent).toHaveClass(/gap-0/);
});

Then('the reader layout is rendered as vertical gapped scroll', async function () {
  const { page } = getWorld();
  const readerContent = page.locator('[data-testid="reader-content"]').first();
  await expect(readerContent).toBeVisible({ timeout: 10000 });
  await expect(readerContent).toHaveClass(/gap-6/);
});

Then('the reader layout is rendered as right-to-left', async function () {
  const { page } = getWorld();
  const pagedContainer = page.locator('[data-testid="reader-paged-container"]').first();
  await expect(pagedContainer).toBeVisible({ timeout: 10000 });
  const leftZone = page.locator('[data-testid="reader-zone-left"]').first();
  await expect(leftZone).toHaveAttribute('aria-label', 'Next Page', { timeout: 10000 });
  const rightZone = page.locator('[data-testid="reader-zone-right"]').first();
  await expect(rightZone).toHaveAttribute('aria-label', 'Previous Page', { timeout: 10000 });
});

Then('the reader layout is rendered as left-to-right', async function () {
  const { page } = getWorld();
  const pagedContainer = page.locator('[data-testid="reader-paged-container"]').first();
  await expect(pagedContainer).toBeVisible({ timeout: 10000 });
  const leftZone = page.locator('[data-testid="reader-zone-left"]').first();
  await expect(leftZone).toHaveAttribute('aria-label', 'Previous Page', { timeout: 10000 });
  const rightZone = page.locator('[data-testid="reader-zone-right"]').first();
  await expect(rightZone).toHaveAttribute('aria-label', 'Next Page', { timeout: 10000 });
});

Then('the reader falls back to the default reading mode layout', async function () {
  const { page } = getWorld();
  const pagedContainer = page.locator('[data-testid="reader-paged-container"]').first();
  await expect(pagedContainer).toBeVisible({ timeout: 10000 });
  const leftZone = page.locator('[data-testid="reader-zone-left"]').first();
  await expect(leftZone).toHaveAttribute('aria-label', 'Next Page', { timeout: 10000 });
});

Then('I am in the reader for {string}', async function (mangaTitle: string) {
  const { page } = getWorld();
  await page.waitForURL(url => url.pathname.includes('/chapter/') || url.pathname.includes('/reader/'), { timeout: 15000 });
  await page.waitForSelector('[data-testid="reader-content"], [data-testid="reader-paged-container"], [data-testid="page-indicator"]', { state: 'visible', timeout: 15000 });
  const titleEl = page.locator('header, div').filter({ hasText: mangaTitle }).first();
  await expect(titleEl).toBeVisible({ timeout: 10000 });
});

When('I swipe {word} on the reader canvas', async function (direction: string) {
  const { page } = getWorld();
  const dir = direction.toLowerCase();

  const container = page.locator('[data-testid="reader-paged-container"], [data-testid="reader-content"]').first();
  await container.waitFor({ state: 'visible', timeout: 10000 });

  await page.evaluate(`(async (dirStr) => {
    const el = document.querySelector('[data-testid="reader-paged-container"]') || document.querySelector('[data-testid="reader-content"]') || document.body;
    const rect = el.getBoundingClientRect();
    const startX = rect.left + rect.width / 2;
    const startY = rect.top + rect.height / 2;
    const distance = 250;
    const endX = dirStr === 'right' ? startX + distance : startX - distance;

    const mkTouch = (x, y) => {
      if (typeof Touch !== 'undefined') {
        return new Touch({
          identifier: 1,
          target: el,
          clientX: x,
          clientY: y,
          screenX: x,
          screenY: y,
          pageX: x,
          pageY: y,
        });
      }
      return {
        identifier: 1,
        target: el,
        clientX: x,
        clientY: y,
        screenX: x,
        screenY: y,
        pageX: x,
        pageY: y,
      };
    };

    const tStart = mkTouch(startX, startY);
    el.dispatchEvent(new TouchEvent('touchstart', {
      touches: [tStart],
      targetTouches: [tStart],
      changedTouches: [tStart],
      bubbles: true,
      cancelable: true,
    }));

    const steps = 6;
    for (let i = 1; i <= steps; i++) {
      const curX = startX + (endX - startX) * (i / steps);
      const tMove = mkTouch(curX, startY);
      el.dispatchEvent(new TouchEvent('touchmove', {
        touches: [tMove],
        targetTouches: [tMove],
        changedTouches: [tMove],
        bubbles: true,
        cancelable: true,
      }));
      await new Promise(r => setTimeout(r, 20));
    }

    const tEnd = mkTouch(endX, startY);
    el.dispatchEvent(new TouchEvent('touchend', {
      touches: [],
      targetTouches: [],
      changedTouches: [tEnd],
      bubbles: true,
      cancelable: true,
    }));
  })("${dir}")`);

  await page.waitForTimeout(600);
});

Then('I am on the last page of chapter {int}', async function (chapterNum: number) {
  const { page } = getWorld();
  await page.waitForSelector('[data-testid="page-indicator"]', { state: 'visible', timeout: 15000 });
  const indicator = page.locator('[data-testid="page-indicator"]').first();
  await expect(indicator).toBeVisible({ timeout: 10000 });

  await page.waitForFunction(() => {
    const el = document.querySelector('[data-testid="page-indicator"]');
    if (!el || !el.textContent) return false;
    const match = el.textContent.match(/(\d+)\s*\/\s*(\d+)/);
    if (!match) return false;
    const curr = parseInt(match[1], 10);
    const total = parseInt(match[2], 10);
    return curr === total && total > 0;
  }, { timeout: 10000 });

  const header = page.locator('header').first();
  await expect(header).toContainText(new RegExp(`Chapter\\s*${chapterNum}|Ch\\.?\\s*${chapterNum}`, 'i'), { timeout: 10000 });
});
