import { spawn, execSync, ChildProcess } from 'child_process';
import * as path from 'path';
import axios from 'axios';

const HEALTH_CHECK_INTERVAL = 500;
const HEALTH_CHECK_TIMEOUT = 30000;

export async function spawnServer(
  home: string,
  port: number,
  fixturesDir: string
): Promise<ChildProcess> {
  const useDocker = process.env.KIYOMI_E2E_USE_DOCKER === 'true';
  let server: ChildProcess;

  if (useDocker) {
    const imageName = process.env.KIYOMI_DOCKER_IMAGE || 'kiyomi:latest';
    const containerName = `kiyomi-e2e-${port}`;
    
    server = spawn('docker', [
      'run', '--rm',
      '--name', containerName,
      '-p', `${port}:${port}`,
      '-v', `${fixturesDir}:/fixtures`,
      '-v', `${home}:/data`,
      '-e', `KIYOMI_PORT=${port}`,
      '-e', `KIYOMI_MOCK_FIXTURES=/fixtures`,
      '-e', `KIYOMI_HOME=/data`,
      imageName
    ]);
  } else {
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
    server = spawn(serverPath, [], {
      env,
      stdio: 'pipe',
    });
  }

  await waitForServer(port);

  return server;
}

export function killServer(server: ChildProcess, port: number): void {
  const useDocker = process.env.KIYOMI_E2E_USE_DOCKER === 'true';
  if (useDocker) {
    try {
      execSync(`docker kill kiyomi-e2e-${port} 2>/dev/null || true`);
    } catch {
      // ignore
    }
  } else if (server && typeof server.pid === 'number') {
    try {
      process.kill(server.pid, 'SIGTERM');
    } catch {
      // ignore
    }
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
