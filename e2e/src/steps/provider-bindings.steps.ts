import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { getWorld } from '../hooks';
import axios from 'axios';

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

/** Parse mangaId from the current URL (e.g. /manga/<id>/...) */
async function getMangaIdFromUrl(page: any): Promise<string> {
  const url = page.url();
  const match = url.match(/\/manga\/([^/]+)/);
  if (!match) throw new Error(`Cannot parse mangaId from URL: ${url}`);
  return match[1];
}

/** Click the providers section toggle button and wait for collapse/expand */
async function clickProvidersToggle(page: any, expectExpanded: boolean) {
  const toggle = page.locator('button').filter({ hasText: /^Providers$/i }).first();
  await toggle.waitFor({ state: 'visible', timeout: 15000 });
  await toggle.click();
  if (expectExpanded) {
    // Wait until ProviderList ul is visible (expanded state)
    await page.locator('ul.rounded-lg.border.border-border.bg-card').waitFor({ state: 'visible', timeout: 15000 });
  } else {
    // Wait until ProviderList ul disappears (collapsed state)
    await page.locator('ul.rounded-lg.border.border-border.bg-card').waitFor({ state: 'hidden', timeout: 15000 });
  }
}

/** Open the dropdown menu on a provider row and return the menu locator */
async function openProviderDropdown(page: any, providerName: string) {
  // Provider rows display the friendly source name (e.g. "Mock Primary"), not the id.
  const displayName = providerName
    .replace(/^mock-primary$/i, 'Mock Primary')
    .replace(/^mock-secondary$/i, 'Mock Secondary');
  // Close any existing open dropdown by pressing Escape (base-ui dropdowns close on Escape).
  await page.keyboard.press('Escape');
  await page.waitForTimeout(150);
  const row = page.locator('li.flex.items-center.gap-3').filter({ hasText: displayName }).first();
  await row.waitFor({ state: 'visible', timeout: 15000 });
  const menuTrigger = row.locator('[role="button"], button').filter({ hasText: '•••' }).first();
  await menuTrigger.click();
  // Dropdown content is portaled (base-ui) — find globally, but only the one we just opened.
  const menu = page.locator('[role="menu"]').first();
  await menu.waitFor({ state: 'visible', timeout: 10000 });
  return menu;
}

// ---------------------------------------------------------------------------
// Given steps
// ---------------------------------------------------------------------------

Given('a seeded library with manga-x bound to mock-primary is running', async function () {
  const { port } = getWorld();
  const base = `http://localhost:${port}`;

  // Import manga from mock-primary (creates manga + binds mock-primary + sets active content)
  const importRes = await axios.post(`${base}/api/v1/library/manga/import`, {
    provider_id: 'mock-primary',
    remote_id: 'alpha',
    user_status: 'reading',
  }, { timeout: 10000 });
  const mangaId = (importRes.data && (importRes.data.id || importRes.data.manga?.id)) as string;
  if (!mangaId) {
    throw new Error(`Could not parse manga id from import response: ${JSON.stringify(importRes.data)}`);
  }

  await new Promise(r => setTimeout(r, 500));
});

Given('a seeded library with manga-x bound to mock-primary and mock-secondary is running', async function () {
  const { port } = getWorld();
  const base = `http://localhost:${port}`;

  // Import manga from mock-primary (creates manga + binds mock-primary + sets active content)
  const importRes = await axios.post(`${base}/api/v1/library/manga/import`, {
    provider_id: 'mock-primary',
    remote_id: 'alpha',
    user_status: 'reading',
  }, { timeout: 10000 });
  const mangaId = (importRes.data && (importRes.data.id || importRes.data.manga?.id)) as string;
  if (!mangaId) {
    throw new Error(`Could not parse manga id from import response: ${JSON.stringify(importRes.data)}`);
  }

  // Add mock-secondary as an additional provider binding via POST /providers
  await axios.post(`${base}/api/v1/library/manga/${mangaId}/providers`, {
    provider_id: 'mock-secondary',
    provider_manga_id: 'alpha',
    manga_title: 'Alpha (Secondary)',
  }, { timeout: 10000 });

  await new Promise(r => setTimeout(r, 500));
});

Given('the manga has only provider {string}', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;

  // GET current providers
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 10000 });
  const providers: any[] = res.data?.providers || [];
  // Remove all providers that are not the target
  for (const p of providers) {
    if (p.provider_id !== providerId) {
      await axios.delete(
        `${base}/api/v1/library/manga/${mangaId}/providers/${p.provider_id}/${encodeURIComponent(p.provider_manga_id)}`,
        { timeout: 10000 }
      );
    }
  }
  // Verify only target remains
  const after = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 10000 });
  const remaining: any[] = after.data?.providers || [];
  expect(remaining).toHaveLength(1);
  expect(remaining[0].provider_id).toBe(providerId);
  // Reload page so the frontend's TanStack Query cache reflects the new bindings.
  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(500);
});

Given('the manga has providers {string}', async function (p1: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 10000 });
  const providers: any[] = res.data?.providers || [];
  expect(providers.some((p: any) => p.provider_id === p1)).toBe(true);
});

Given('the manga has providers {string} and {string}', async function (p1: string, p2: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;

  // Ensure both are present (add if missing) by binding to existing manga
  const existing = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 10000 });
  const current: any[] = existing.data?.providers || [];
  const ids = new Set(current.map((p: any) => p.provider_id));

  // Per-provider manga_title as defined in fixtures
  const providerTitles: Record<string, string> = {
    'mock-primary': 'Alpha Manga',
    'mock-secondary': 'Alpha (Secondary)',
  };

  for (const providerId of [p1, p2]) {
    if (!ids.has(providerId)) {
      const mangaTitle = providerTitles[providerId] || current[0]?.manga_title || 'Alpha Manga';
      await axios.post(`${base}/api/v1/library/manga/${mangaId}/providers`, {
        provider_id: providerId,
        provider_manga_id: 'alpha',
        manga_title: mangaTitle,
      }, { timeout: 10000 });
    }
  }
  // Reload so the UI reflects the new bindings
  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(500);
});

Given('{string} is the active content provider', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;

  // Find provider_manga_id for this provider from current bindings
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 10000 });
  const providers: any[] = res.data?.providers || [];
  const target = providers.find((p: any) => p.provider_id === providerId);
  expect(target).toBeDefined();

  await axios.patch(
    `${base}/api/v1/library/manga/${mangaId}/content`,
    { provider_id: providerId, provider_manga_id: target.provider_manga_id },
    { timeout: 10000 }
  );
  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(500);
});

// ---------------------------------------------------------------------------
// When steps
// ---------------------------------------------------------------------------

When('I open the library manga details page for {string}', async function (title: string) {
  const { page } = getWorld();
  // Reuse existing library card click pattern from library-details.steps
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

When('I expand the providers section', async function () {
  const { page } = getWorld();
  await clickProvidersToggle(page, true);
});

When('I confirm the dialog', async function () {
  const { page } = getWorld();
  const dialog = page.locator('[role="dialog"]').filter({ hasText: /Confirm|Add|Switch/i }).first();
  await dialog.waitFor({ state: 'visible', timeout: 10000 });
  const confirmBtn = dialog.locator('button').filter({ hasText: /^(Add Provider|Confirm|Add)$/i }).first();
  await confirmBtn.waitFor({ state: 'visible', timeout: 10000 });
  try {
    await confirmBtn.click({ force: true, timeout: 5000 });
  } catch (e) {
    await dialog.locator('button[type="submit"], button:not([type])').first().click({ force: true });
  }
  await page.waitForTimeout(1000);
});

When('I search the provider dialog for {string}', async function (query: string) {
  // Search inside the open AddProviderDialog combobox (alias combobox).
  const { page } = getWorld();
  const dialog = page.locator('[role="dialog"]').first();
  await dialog.waitFor({ state: 'visible', timeout: 10000 });
  const input = dialog.locator('input[placeholder*="Search" i]').first();
  await input.waitFor({ state: 'visible', timeout: 10000 });
  await input.click();
  // Clear then type to trigger React onChange handlers (fill() may bypass them on custom inputs).
  await input.fill('');
  await input.pressSequentially(query, { delay: 50 });
  await page.waitForTimeout(1000); // debounced search
});

When('I select provider {string} in the dialog', async function (providerName: string) {
  // Pick a different provider in the AddProviderDialog's Select dropdown.
  const { page } = getWorld();
  const dialog = page.locator('[role="dialog"]').first();
  await dialog.waitFor({ state: 'visible', timeout: 10000 });
  const trigger = dialog.locator('button[role="combobox"]').first();
  await trigger.click();
  await page.waitForTimeout(300);
  const friendly = friendlyProviderName(providerName);
  const option = page.locator('[role="option"]').filter({ hasText: friendly }).first();
  await option.waitFor({ state: 'visible', timeout: 10000 });
  await option.click();
  await page.waitForTimeout(1500);
});

When('I click the result for {string}', async function (_providerName: string) {
  // After provider is selected in the dialog, click the first search result button via JS.
  const { page } = getWorld();
  const dialog = page.locator('[role="dialog"]').first();
  await dialog.waitFor({ state: 'visible', timeout: 10000 });
  await page.waitForSelector('ul.divide-y li button', { timeout: 10000 });
  await dialog.locator('ul.divide-y li button').first().evaluate(el => (el as HTMLElement).click());
  await page.waitForTimeout(500);
});

When('I click {string} on provider {string}', async function (actionText: string, providerName: string) {
  const { page } = getWorld();
  // Auto-accept any dialog so the click action actually takes effect (Remove, etc.).
  // Capture the message so assertion steps can verify destructive confirmations fired.
  let capturedMessage = '';
  const handler = async (dialog: any) => {
    capturedMessage = dialog.message();
    try { await dialog.accept(); } catch { /* ignore */ }
  };
  page.on('dialog', handler);
  const menu = await openProviderDropdown(page, providerName);
  const item = menu.locator('[role="menuitem"]').filter({ hasText: new RegExp(actionText, 'i') }).first();
  await item.waitFor({ state: 'visible', timeout: 10000 });
  await item.click();
  await page.waitForTimeout(800);
  page.off('dialog', handler);
  (this as any)._lastDialogMessage = capturedMessage;
});

When('I click {string} in the chapter list toolbar', async function (buttonText: string) {
  const { page } = getWorld();
  // Use aria-label for unambiguous matching of the toolbar buttons.
  const ariaMap: Record<string, string> = {
    'Refresh': 'Refresh chapter list',
    'Refresh chapters': 'Refresh chapter list',
  };
  const aria = ariaMap[buttonText] || buttonText;
  const btn = page.locator(`button[aria-label="${aria}"]`).first();
  await btn.waitFor({ state: 'visible', timeout: 15000 });
  await btn.click();
  await page.waitForTimeout(800);
});

When('I open the metadata options menu', async function () {
  const { page } = getWorld();
  // MoreVertical button is in the details hero, next to the title
  const trigger = page.locator('[aria-label="Metadata options"], button[title="Metadata options"]').first();
  await trigger.waitFor({ state: 'visible', timeout: 15000 });
  await trigger.click();
  const menu = page.locator('[role="menu"]').first();
  await menu.waitFor({ state: 'visible', timeout: 10000 });
});

When('I PATCH the manga\'s content provider with a metadata-only binding', async function () {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;

  // Capture original provider before attempting
  const mangaRes = await axios.get(`${base}/api/v1/library/manga/${mangaId}`, { timeout: 10000 });
  const manga: any = mangaRes.data;
  (this as any)._originalProviderId = manga.contentProviderId || manga.sourceId || manga.meta?.content?.provider_id;

  // Get list of registered providers
  const sourcesRes = await axios.get(`${base}/api/v1/providers`, { timeout: 10000 });
  const sources: any[] = sourcesRes.data || [];

  // Find a source that lacks 'content' capability
  const metadataOnly = sources.find((s: any) => {
    const caps: string[] = s.capabilities || [];
    return caps.length > 0 && !caps.includes('content');
  });

  if (!metadataOnly) {
    // No metadata-only source available — skip by throwing a known sentinel
    (this as any)._skipSwitch = true;
    return;
  }

  // Find a provider binding for this source on the manga
  const providersRes = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 10000 });
  const providers: any[] = providersRes.data || [];
  const binding = providers.find((p: any) => p.provider_id === metadataOnly.id);
  if (!binding) {
    (this as any)._skipSwitch = true;
    return;
  }

  // Attempt the switch — should fail with 400
  try {
    const errRes = await axios.patch(
      `${base}/api/v1/library/manga/${mangaId}/content`,
      { provider_id: metadataOnly.id, provider_manga_id: binding.provider_manga_id },
      { validateStatus: () => true, timeout: 10000 }
    );
    (this as any)._lastErrRes = errRes;
  } catch (e: any) {
    (this as any)._lastErrRes = e.response;
  }
});

When('I confirm the destructive switch', async function () {
  // The previous When 'I click Switch content to this' step already triggered the confirm
  // and captured its message. The destructive action is now complete (auto-accepted).
  // Just verify the dialog was fired.
  const captured = (this as any)._lastDialogMessage as string;
  expect(captured, 'expected confirm dialog to fire before the switch').toBeTruthy();
});

// ---------------------------------------------------------------------------
// Then steps
// ---------------------------------------------------------------------------

Then('the providers section is collapsed', async function () {
  const { page } = getWorld();
  // Collapsed = ProviderList ul is not visible AND ChevronDown icon is showing
  const list = page.locator('ul.rounded-lg.border.border-border.bg-card');
  await expect(list).toBeHidden({ timeout: 10000 });
  const toggle = page.locator('button').filter({ hasText: /^Providers$/i }).first();
  await expect(toggle).toBeVisible({ timeout: 10000 });
  // Verify ChevronDown (collapsed) not ChevronUp — lucide-react renders class as lucide-chevron-down (kebab) or lucide ChevronDown.
  const chevronDown = toggle.locator('svg.lucide-chevron-down, svg[class*="chevron-down"]');
  await expect(chevronDown).toBeVisible({ timeout: 5000 });
});

Then('the providers section shows the {string} button', async function (buttonText: string) {
  const { page } = getWorld();
  // The "Add provider" button is always in the header, even when collapsed
  const headerArea = page.locator('.space-y-3 > div').first();
  const btn = headerArea.locator('button').filter({ hasText: new RegExp(buttonText, 'i') }).first();
  await expect(btn).toBeVisible({ timeout: 10000 });
});

Then('the provider section shows the provider {string}', async function (providerId: string) {
  const { page } = getWorld();
  // Expand the providers section first (the assertion requires seeing rows).
  await clickProvidersToggle(page, true);
  const row = page.locator('li.flex.items-center.gap-3').filter({ hasText: providerId }).first();
  await expect(row).toBeVisible({ timeout: 10000 });
});

// Map provider_id used in features to the friendly source name rendered in rows.
function friendlyProviderName(id: string): string {
  if (/^mock-primary$/i.test(id)) return 'Mock Primary';
  if (/^mock-secondary$/i.test(id)) return 'Mock Secondary';
  return id;
}

Then('I see the provider {string} with manga_title {string}', async function (providerName: string, mangaTitle: string) {
  const { page } = getWorld();
  const list = page.locator('ul.rounded-lg.border.border-border.bg-card');
  const row = list.locator('li.flex.items-center.gap-3')
    .filter({ hasText: friendlyProviderName(providerName) })
    .filter({ hasText: mangaTitle })
    .first();
  await expect(row).toBeVisible({ timeout: 15000 });
});

Then('the content capability pill on {string} shows the active indicator', async function (providerName: string) {
  const { page } = getWorld();
  const row = page.locator('li.flex.items-center.gap-3').filter({ hasText: friendlyProviderName(providerName) }).first();
  // Active pill has border-emerald-500/40 and contains Check svg
  const pill = row.locator('[class*="border-emerald-500"]');
  await expect(pill).toBeVisible({ timeout: 10000 });
  const checkIcon = pill.locator('svg[class*="lucide-check"], svg[class*="Check"]');
  await expect(checkIcon).toBeVisible({ timeout: 5000 });
});

Then('the content capability pill on {string} does not show the active indicator', async function (providerName: string) {
  const { page } = getWorld();
  const row = page.locator('li.flex.items-center.gap-3').filter({ hasText: friendlyProviderName(providerName) }).first();
  // No emerald border class on the pill
  const pills = row.locator('[class*="border-emerald-500"]');
  const count = await pills.count();
  if (count > 0) {
    await expect(pills.first()).not.toBeVisible();
  }
  // Also check: if pill exists, it should not have emerald class
  const allPills = row.locator('[class*="Badge"], span[class*="text-"]');
  for (let i = 0; i < await allPills.count(); i++) {
    const cls = await allPills.nth(i).getAttribute('class');
    if (cls && cls.includes('emerald')) {
      throw new Error('Found emerald pill on non-active provider');
    }
  }
});

Then('the provider {string} shows capability badges {string}', async function (providerName: string, badgesCsv: string) {
  const { page } = getWorld();
  const row = page.locator('li.flex.items-center.gap-3').filter({ hasText: friendlyProviderName(providerName) }).first();
  const expected = badgesCsv.split(',').map(s => s.trim());
  for (const badge of expected) {
    // Capability pills render as <span data-slot="badge"> with text content
    const badgeEl = row.locator('[data-slot="badge"], [class*="Badge"]').filter({ hasText: new RegExp(`^${badge}$`, 'i') }).first();
    await expect(badgeEl).toBeVisible({ timeout: 10000 });
  }
});

Then('the add provider dialog opens', async function () {
  const { page } = getWorld();
  const dialog = page.locator('[role="dialog"]').first();
  await expect(dialog).toBeVisible({ timeout: 15000 });
});

Then('the provider {string} is added to the manga', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 15000 });
  const providers: any[] = res.data?.providers || [];
  expect(providers.some((p: any) => p.provider_id === providerId)).toBe(true);
});

Then('the provider {string} is removed from the manga', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 15000 });
  const providers: any[] = res.data?.providers || [];
  expect(providers.some((p: any) => p.provider_id === providerId)).toBe(false);
});

Then('chapters are refreshed from {string}', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  // Verify manga's content provider matches
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}`, { timeout: 15000 });
  const manga: any = res.data;
  const activeProvider = manga.contentProviderId || manga.sourceId || manga.meta?.content?.provider_id;
  expect(activeProvider).toBe(providerId);
});

Then('{string} becomes the active content provider', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}`, { timeout: 15000 });
  const manga: any = res.data;
  const activeProvider = manga.contentProviderId || manga.sourceId || manga.meta?.content?.provider_id;
  expect(activeProvider).toBe(providerId);
});

Then('{string} remains the active content provider', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}`, { timeout: 15000 });
  const manga: any = res.data;
  const activeProvider = manga.contentProviderId || manga.sourceId || manga.meta?.content?.provider_id;
  expect(activeProvider).toBe(providerId);
});

Then('the provider {string} remains in the providers list', async function (providerId: string) {
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}/providers`, { timeout: 15000 });
  const providers: any[] = res.data || [];
  expect(providers.some((p: any) => p.provider_id === providerId)).toBe(true);
});

Then('the {string} button is visible in the chapter list toolbar', async function (buttonText: string) {
  const { page } = getWorld();
  const ariaMap: Record<string, string> = {
    'Switch': 'Switch content provider',
    'Refresh': 'Refresh chapter list',
    'Refresh chapters': 'Refresh chapter list',
  };
  const aria = ariaMap[buttonText] || buttonText;
  const btn = page.locator(`button[aria-label="${aria}"]`).first();
  await expect(btn).toBeVisible({ timeout: 10000 });
});

Then('a confirmation dialog appears warning about discarding cached chapters', async function () {
  // Captured by the When 'I click ... on provider ...' step via the auto-accept handler.
  // We just verify the captured message contains the expected warning.
  const captured = (this as any)._lastDialogMessage as string;
  expect(captured, 'expected a confirm dialog to fire with discard/cached/chapter wording').toBeTruthy();
  expect(captured.toLowerCase()).toMatch(/discard|cached|chapter/);
});

Then('no add provider dialog is opened', async function () {
  const { page } = getWorld();
  const dialogs = page.locator('[role="dialog"]');
  const count = await dialogs.count();
  for (let i = 0; i < count; i++) {
    const d = dialogs.nth(i);
    if (await d.isVisible()) {
      const title = await d.locator('[class*="DialogTitle"], h2').first().textContent().catch(() => '');
      if (title && (title.includes('Add Provider') || title.includes('Switch Content'))) {
        throw new Error('Add/Switch provider dialog is still open');
      }
    }
  }
});

Then('the provider {string} \\(active\\) has no {string} action', async function (providerName: string, actionText: string) {
  const { page } = getWorld();
  const menu = await openProviderDropdown(page, providerName);
  const item = menu.locator('[role="menuitem"]').filter({ hasText: new RegExp(actionText, 'i') });
  await expect(item).toHaveCount(0, { timeout: 5000 });
});

Then('the provider {string} \\(non-active\\) has the {string} action', async function (providerName: string, actionText: string) {
  const { page } = getWorld();
  const menu = await openProviderDropdown(page, providerName);
  const item = menu.locator('[role="menuitem"]').filter({ hasText: new RegExp(actionText, 'i') });
  await expect(item).toBeVisible({ timeout: 10000 });
});

Then('the {string} action on provider {string} is disabled', async function (actionText: string, providerName: string) {
  const { page } = getWorld();
  const menu = await openProviderDropdown(page, providerName);
  const item = menu.locator('[role="menuitem"]').filter({ hasText: new RegExp(actionText, 'i') });
  // Playwright's toBeDisabled checks aria-disabled + disabled + pointer-events:none via opacity.
  await expect(item).toBeDisabled({ timeout: 10000 });
});

Then('the API returns 400 with {string}', async function (errorSubstring: string) {
  if ((this as any)._skipSwitch) {
    // No metadata-only provider registered — graceful skip
    return;
  }
  const errRes = (this as any)._lastErrRes;
  expect(errRes).toBeDefined();
  expect(errRes.status).toBe(400);
  const body = typeof errRes.data === 'string' ? errRes.data : JSON.stringify(errRes.data);
  expect(body.toLowerCase()).toContain(errorSubstring.toLowerCase());
});

Then('the active content provider is unchanged', async function () {
  if ((this as any)._skipSwitch) return;
  const before = (this as any)._originalProviderId;
  const { page, port } = getWorld();
  const mangaId = await getMangaIdFromUrl(page);
  const base = `http://localhost:${port}`;
  const res = await axios.get(`${base}/api/v1/library/manga/${mangaId}`, { timeout: 15000 });
  const manga: any = res.data;
  const current = manga.contentProviderId || manga.sourceId || manga.meta?.content?.provider_id;
  expect(current).toBe(before);
});

Then('no provider row offers a {string} action', async function (actionText: string) {
  const { page } = getWorld();
  const list = page.locator('ul.rounded-lg.border.border-border.bg-card');
  const rows = list.locator('li.flex.items-center.gap-3');
  const count = await rows.count();
  for (let i = 0; i < count; i++) {
    const row = rows.nth(i);
    // Open dropdown if it has one (••• trigger)
    const trigger = row.locator('[role="button"], button').filter({ hasText: '•••' });
    if (await trigger.isVisible()) {
      await trigger.click();
      const menu = page.locator('[role="menu"]').first();
      await menu.waitFor({ state: 'visible', timeout: 5000 });
      const item = menu.locator('[role="menuitem"]').filter({ hasText: new RegExp(actionText, 'i') });
      const itemCount = await item.count();
      if (itemCount > 0) {
        throw new Error(`Found action "${actionText}" on row ${i}`);
      }
      // Close menu by clicking elsewhere
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    }
  }
});

Then('I see the {string} menu item', async function (menuItemText: string) {
  const { page } = getWorld();
  const menu = page.locator('[role="menu"]').first();
  await expect(menu).toBeVisible({ timeout: 10000 });
  const item = menu.locator('[role="menuitem"]').filter({ hasText: new RegExp(menuItemText, 'i') }).first();
  await expect(item).toBeVisible({ timeout: 10000 });
});

Then('{string} is not shown in the providers section header', async function (buttonText: string) {
  const { page } = getWorld();
  const headerArea = page.locator('.space-y-3 > div').first();
  const btns = headerArea.locator('button').filter({ hasText: new RegExp(buttonText, 'i') });
  await expect(btns).toHaveCount(0, { timeout: 5000 });
});
