import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { spawn } from 'child_process';

export function mkdtemp(): string {
  const tmpDir = path.join(os.tmpdir(), `kiyomi-e2e-${uuidv4()}`);
  fs.mkdirSync(tmpDir, { recursive: true });
  return tmpDir;
}

export function teardownTmp(home: string): void {
  if (server) {
    killServer(server.pid!);
    server = null;
  }
  fs.rmSync(home, { recursive: true, force: true });
}

let server: ReturnType<typeof spawn> | null = null;

function killServer(pid: number): void {
  try {
    process.kill(pid, 'SIGTERM');
  } catch {
    // ignore
  }
}

export function copyFixtures(src: string, dest: string): void {
  // Provider reads from {fixturesDir}/providers/, so mirror the src structure
  // inside dest so {dest}/fixtures/providers/ matches the expected layout.
  const fixturesDest = path.join(dest, 'fixtures');
  const providersSrc = path.join(src, 'providers');
  const providersDest = path.join(fixturesDest, 'providers');
  const librarySeedSrc = path.join(src, 'library-seed');
  const librarySeedDest = path.join(fixturesDest, 'library-seed');

  if (fs.existsSync(providersSrc)) {
    copyDirRecursive(providersSrc, providersDest);
  }
  if (fs.existsSync(librarySeedSrc)) {
    copyDirRecursive(librarySeedSrc, librarySeedDest);
  }

  // Per-provider mock fixture directories (mock-primary/, mock-secondary/).
  // The e2e binary registers NewWithID(...) against these subdirs at startup.
  for (const name of ['mock-primary', 'mock-secondary']) {
    const subSrc = path.join(src, name);
    const subDest = path.join(fixturesDest, name);
    if (fs.existsSync(subSrc)) {
      copyDirRecursive(subSrc, subDest);
    }
  }
}

function copyDirRecursive(src: string, dest: string): void {
  fs.mkdirSync(dest, { recursive: true });
  const entries = fs.readdirSync(src, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isDirectory()) {
      copyDirRecursive(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
    }
  }
}

function uuidv4(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
