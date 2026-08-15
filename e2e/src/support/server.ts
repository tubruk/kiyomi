import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import axios from 'axios';

const HEALTH_CHECK_INTERVAL = 500;
const HEALTH_CHECK_TIMEOUT = 30000;

export async function spawnServer(
  home: string,
  port: number,
  fixturesDir: string
): Promise<ChildProcess> {
  const env = {
    ...process.env,
    KIYOMI_HOME: home,
    KIYOMI_PORT: String(port),
    KIYOMI_MOCK_FIXTURES: fixturesDir,
    // Use the built web UI (served from disk during dev/e2e)
    KIYOMI_WEB_DIR: path.resolve(__dirname, '../../../web/dist'),
  };

  const serverPath = process.env.KIYOMI_E2E_BINARY
    || path.join(process.cwd(), 'bin', 'kiyomi-e2e');
  const server = spawn(serverPath, [], {
    env,
    stdio: 'pipe',
  });

  await waitForServer(port);

  return server;
}

export function killServer(pid: number): void {
  try {
    process.kill(pid, 'SIGTERM');
  } catch {
    // ignore
  }
}

async function waitForServer(port: number): Promise<void> {
  const deadline = Date.now() + HEALTH_CHECK_TIMEOUT;
  const url = `http://localhost:${port}/api/v1/library/manga`;

  while (Date.now() < deadline) {
    try {
      const response = await axios.get(url, { timeout: 1000 });
      if (response.status === 200) {
        return;
      }
    } catch {
      // still starting
    }
    await sleep(HEALTH_CHECK_INTERVAL);
  }

  throw new Error(`Server did not start within ${HEALTH_CHECK_TIMEOUT}ms`);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
