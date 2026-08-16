import { BeforeAll, Before, After, AfterAll, setDefaultTimeout } from '@cucumber/cucumber';

setDefaultTimeout(30000);
import { chromium, BrowserContext } from '@playwright/test';
import { E2EWorld } from './world';
import { mkdtemp, copyFixtures } from './support/tmp';
import { spawnServer, killServer } from './support/server';
import { allocatePort } from './support/ports';
import * as path from 'path';
import * as fs from 'fs';
import { fileURLToPath } from 'url';

// Module-level singleton — survives across Cucumber's World lifecycle in tsx
let _world: E2EWorld;

export function getWorld(): E2EWorld {
  if (!_world) throw new Error('world not initialized — check BeforeAll hook');
  return _world;
}

const __dirname = path.dirname(fileURLToPath(import.meta.url));

BeforeAll(async function () {
  _world = this as E2EWorld;

  const headless = process.env.E2E_HEADLESS !== '0' && !process.argv.includes('--headed');
  _world.browser = await chromium.launch({ headless });
  _world.context = await _world.browser.newContext({ hasTouch: true });

  // Initialize GOCOVERDIR for backend coverage
  const coverDir = path.resolve(__dirname, '../test-results/coverage-go');
  fs.mkdirSync(coverDir, { recursive: true });
  process.env.GOCOVERDIR = coverDir;
});

Before(async function () {
  _world.port = await allocatePort();
  _world.home = mkdtemp();

  const fixturesSrc = path.join(__dirname, '..', '..', 'docs', 'e2e', 'fixtures');
  copyFixtures(fixturesSrc, _world.home);
  _world.fixturesDir = path.join(_world.home, 'fixtures');

  _world.server = await spawnServer(_world.home, _world.port, _world.fixturesDir);

  if (_world.context) {
    _world.page = await _world.context.newPage();
    // Start frontend V8 JS coverage
    await _world.page.coverage.startJSCoverage();
  }
});

After(async function (scenario) {
  if (_world?.page) {
    try {
      const coverage = await _world.page.coverage.stopJSCoverage();
      // Filter for compiled web assets (c8 will map them back to /src/ via sourcemaps)
      const appCoverage = coverage.filter(entry => entry.url.includes('/assets/'));
      
      const distDir = path.resolve(__dirname, '../../web/dist');
      const mappedCoverage = appCoverage.map(entry => {
        // Map http://localhost:PORT/assets/... to file:///<absolute-path-to-web-dist>/assets/...
        const mappedUrl = entry.url.replace(/^https?:\/\/localhost:\d+/, 'file://' + distDir);
        return {
          ...entry,
          url: mappedUrl,
        };
      });
      
      const targetDir = path.resolve(__dirname, '../test-results/coverage-frontend-v8');
      fs.mkdirSync(targetDir, { recursive: true });
      fs.writeFileSync(
        path.join(targetDir, `${scenario?.pickle?.id || Date.now()}.json`),
        JSON.stringify({ result: mappedCoverage })
      );
    } catch (e) {
      // Ignore errors if browser is closed or coverage failed
    }
    await _world.page.close();
    _world.page = null;
  }
  if (_world?.server) {
    killServer(_world.server, _world.port);
    _world.server = null;
  }
  if (_world?.home) {
    try { fs.rmSync(_world.home, { recursive: true, force: true }); } catch { /* ignore */ }
    _world.home = null;
  }
});

AfterAll(async function () {
  if (_world?.browser) {
    await _world.browser.close();
  }
  _world = undefined as any;
});
