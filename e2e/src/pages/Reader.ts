import { Page } from '@playwright/test';

export class ReaderPage {
  constructor(private page: Page) {}

  async goto(mangaId: string, chapterNumber: number): Promise<void> {
    await this.page.goto(`/reader/${mangaId}/${chapterNumber}`);
    await this.page.waitForLoadState('networkidle');
  }

  async getCurrentPage(): Promise<number> {
    const pageIndicator = this.page.locator('[data-testid="page-indicator"], .page-indicator, [class*="pageIndicator"]').first();
    const text = await pageIndicator.textContent() ?? '';
    const match = text.match(/(\d+)/);
    return match ? parseInt(match[1], 10) : 1;
  }

  async navigateNext(): Promise<void> {
    const nextBtn = this.page.locator('button[aria-label*="next" i], button:has-text("Next"), [class*="next"]').first();
    await nextBtn.click();
    await this.page.waitForTimeout(300);
  }

  async navigatePrev(): Promise<void> {
    const prevBtn = this.page.locator('button[aria-label*="prev" i], button:has-text("Prev"), [class*="prev"]').first();
    await prevBtn.click();
    await this.page.waitForTimeout(300);
  }
}
