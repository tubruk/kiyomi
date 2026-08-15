import { Locator, Page } from '@playwright/test';

export class DownloadsPage {
  constructor(private page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto('/#/downloads');
    await this.page.waitForLoadState('networkidle');
  }

  async getDownloadItems(): Promise<Locator[]> {
    return await this.page.locator('[data-testid="download-item"], .download-item, [class*="download"]').all();
  }
}
