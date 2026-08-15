import { Locator, Page } from '@playwright/test';

export class ExplorePage {
  constructor(private page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto('/#/discover');
    await this.page.waitForLoadState('networkidle');
  }

  async searchByTag(tag: string): Promise<void> {
    const tagInput = this.page.locator('input[placeholder*="tag" i], input[placeholder*="Tag" i], [data-testid="tag-search"]').first();
    await tagInput.fill(tag);
    const searchBtn = this.page.locator('button:has-text("Search"), button[aria-label*="Search" i]').first();
    await searchBtn.click();
    await this.page.waitForTimeout(500);
  }

  async getSearchResults(): Promise<Locator[]> {
    return await this.page.locator('[data-testid="search-result"], .search-result, [class*="result"]').all();
  }
}
