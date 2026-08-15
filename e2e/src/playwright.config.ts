import { defineConfig } from '@playwright/test';

export default defineConfig({
  timeout: 30000,
  retries: 0,
  use: {
    trace: 'off',
    screenshot: 'only-on-failure',
  },
  outputDir: 'test-results/',
});
