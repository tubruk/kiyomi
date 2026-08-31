import { Locator, Page, expect } from '@playwright/test';

export class MangaDetailPage {
  constructor(private page: Page) {}

  async goto(mangaId: string): Promise<void> {
    await this.page.goto(`/manga/${mangaId}`);
    await this.page.waitForLoadState('networkidle');
  }

  async getChapterList(): Promise<Locator[]> {
    return await this.page.locator('[data-testid="chapter-item"], .chapter-item, [class*="chapter"]').all();
  }

  async pullChapter(chapterNumber: number): Promise<void> {
    const chapterItem = this.page.locator(`[data-testid="chapter-item"], .chapter-item, [class*="chapter"]`).filter({
      hasText: new RegExp(`^(ch\\.?|chapter\\s*)?${chapterNumber}`, 'i'),
    }).first();

    const pullBtn = chapterItem.locator('button:has-text("Pull"), [aria-label*="pull" i]').first();
    await pullBtn.click();
    await this.page.waitForTimeout(500);
  }

  async getStatus(): Promise<string> {
    const statusEl = this.page.locator('[data-testid="manga-status"], .status, [class*="status"]').first();
    return await statusEl.textContent() ?? '';
  }
}
