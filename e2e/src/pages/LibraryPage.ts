import { Locator, Page, expect } from '@playwright/test';

export class LibraryPage {
  constructor(private page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto('/');
  }

  getMangaCards(): Promise<Locator[]> {
    return this.page.locator('[data-testid="manga-card"], [class*="mangaCard"]').all();
  }

  getMangaCard(title: string): Locator {
    return this.page.locator(`[data-testid="manga-card"], .manga-card, [class*="mangaCard"]`).filter({ hasText: title }).first();
  }

  async openAddModal(): Promise<void> {
    const addButton = this.page.locator('button:has-text("Add"), button:has-text("Add manga"), [aria-label="Add manga"]').first();
    await addButton.click();
    await expect(this.page.locator('[role="dialog"], .modal, [aria-label="Add manga"]')).toBeVisible();
  }

  async searchFor(query: string): Promise<void> {
    const searchInput = this.page.locator('input[type="search"], input[placeholder*="search" i], input[placeholder*="Search" i]').first();
    await searchInput.fill(query);
    await this.page.waitForTimeout(300); // debounce
  }

  async addMangaFromResults(title: string): Promise<void> {
    const resultItem = this.page.locator(`[data-testid="search-result"], .search-result, [class*="result"]`).filter({ hasText: title }).first();
    const addBtn = resultItem.locator('button:has-text("Add"), [aria-label="Add"]').first();
    await addBtn.click();
    await this.page.waitForTimeout(500); // wait for state update
  }
}
