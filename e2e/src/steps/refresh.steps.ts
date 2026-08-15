import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';
import axios from 'axios';
import * as path from 'path';
import * as fs from 'fs';

Given('the manga has {string} chapters', async function (countStr: string) {
  const count = parseInt(countStr, 10);
  const { port, fixturesDir, page } = getWorld();
  const base = `http://localhost:${port}`;

  // 1. Find manga ID for alpha
  const listRes = await axios.get(`${base}/api/v1/library/manga`);
  const mangaList = Array.isArray(listRes.data) ? listRes.data : (listRes.data?.data ?? []);
  const manga = mangaList.find((m: any) => m.title === 'Alpha Manga');
  if (!manga) throw new Error('Alpha Manga not found in library');
  const mangaId = manga.id;

  // 2. Modify the provider fixture to temporarily only return `count` chapters
  const filePath = path.join(fixturesDir, 'providers', 'manga-alpha.json');
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  const allChapters = [...data.chapters];
  data.chapters = allChapters.slice(0, count);
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2));

  // 3. Get chapters from DB and delete the extra chapters
  const chapRes = await axios.get(`${base}/api/v1/library/manga/${mangaId}/chapters`);
  const chapters = chapRes.data.chapters;
  chapters.sort((a: any, b: any) => a.number - b.number);
  if (chapters.length > count) {
    const toDelete = chapters.slice(count);
    for (const ch of toDelete) {
      await axios.delete(`${base}/api/v1/library/manga/${mangaId}/chapters/${ch.id}`);
    }
  }

  // 4. Restore the provider fixture to have all chapters
  data.chapters = allChapters;
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2));

  // 5. Reload page to reflect starting database state of 3 chapters
  await page.reload();
  await page.waitForTimeout(1000);
});

When('I click the refresh button', async function () {
  const { page } = getWorld();
  const btn = page.locator('button:has-text("Refresh"), button[aria-label="Refresh chapter list"]').first();
  await btn.click();
  await page.waitForTimeout(1000);
});

Then('the chapter list updates', async function () {
  const { page } = getWorld();
  // Wait until "Refreshing..." goes back to "Refresh"
  const refreshBtn = page.locator('button:has-text("Refresh")').first();
  await refreshBtn.waitFor({ state: 'visible', timeout: 15000 });
});

Then('new chapters are added to the list', async function () {
  const { page } = getWorld();
  const chapters = await page.locator('[class*="rounded-lg"][class*="border-border"][class*="bg-card"]:has(a)').count();
  expect(chapters).toBe(5);
});

Then('I am notified about the new chapters', async function () {
  const { page } = getWorld();
  const toast = page.locator('[role="status"], [class*="toast"], :has-text("Refresh complete")').first();
  await expect(toast).toBeVisible({ timeout: 5000 });
  await expect(toast).toContainText('Refresh complete');
});

Given('the manga is up to date with the provider', async function () {
  // No-op: Alpha Manga starts with all 5 chapters, mock provider has all 5 chapters.
});

Then('I see a message {string}', async function (msg: string) {
  const { page } = getWorld();
  const toast = page.locator(`[role="status"], [class*="toast"], :has-text("${msg}")`).first();
  await expect(toast).toBeVisible({ timeout: 5000 });
  await expect(toast).toContainText(msg);
});

Then('the chapter list is unchanged', async function () {
  const { page } = getWorld();
  const chapters = await page.locator('[class*="rounded-lg"][class*="border-border"][class*="bg-card"]:has(a)').count();
  expect(chapters).toBe(5);
});

Given('the provider has removed some chapters', async function () {
  const { fixturesDir } = getWorld();
  const filePath = path.join(fixturesDir, 'providers', 'manga-alpha.json');
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  // Remove the last chapter (Chapter 5)
  data.chapters = data.chapters.slice(0, 4);
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2));
});

Then('missing chapters are marked as orphans', async function () {
  const { page } = getWorld();
  const orphanMarker = page.locator('[aria-label*="Orphan:"]').first();
  await expect(orphanMarker).toBeVisible({ timeout: 5000 });
});

Then('the orphan count is displayed', async function () {
  const { page } = getWorld();
  const btn = page.locator('button:has-text("Remove missing (1)")').first();
  await expect(btn).toBeVisible({ timeout: 5000 });
});
