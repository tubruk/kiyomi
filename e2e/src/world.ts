import { Before, After, BeforeAll, AfterAll, setWorldConstructor, World } from '@cucumber/cucumber';
import { Browser, Page, chromium, BrowserContext } from '@playwright/test';
import { ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

export class E2EWorld extends World {
  server: ChildProcess | null = null;
  browser: Browser | null = null;
  context: BrowserContext | null = null;
  page: Page | null = null;
  home: string = '';
  port: number = 0;
  fixturesDir: string = '';

  constructor(options: any) {
    super({ timeout: 60000, ...options });
  }

  apiBase(): string {
    return `http://localhost:${this.port}`;
  }
}

setWorldConstructor(E2EWorld);
