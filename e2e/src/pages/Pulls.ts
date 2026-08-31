import { Locator, Page } from '@playwright/test';

export class PullsPage {
  constructor(private page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto('/#/pulls');
    await this.page.waitForLoadState('networkidle');
  }

  async getPullItems(): Promise<Locator[]> {
    return await this.page.locator('[data-testid="pull-item"], .pull-item, [class*="pull"]').all();
  }
}