import * as net from 'net';

// Each parallel worker gets a unique base port via CUCUMBER_WORKER_ID
// (1-based). Workers MUST use disjoint ranges to avoid port collisions.
//
//   Worker 1 → 4111 .. 4119
//   Worker 2 → 4121 .. 4129
//   ...
//
// Per-scenario allocator walks upward from the worker base. SIGTERM
// in After releases the port; the new server re-binds (Go sets SO_REUSEADDR).
const WORKER_BASE_PORT = 4111;
const WORKER_STRIDE = 10;
const MAX_WORKER_INDEX = 8;

export function workerBasePort(): number {
  const workerIdRaw = process.env.CUCUMBER_WORKER_ID;
  const workerId = workerIdRaw ? parseInt(workerIdRaw, 10) : 0;
  if (workerId < 0 || workerId >= MAX_WORKER_INDEX) {
    throw new Error(`workerBasePort: workerId ${workerId} out of range`);
  }
  return WORKER_BASE_PORT + workerId * WORKER_STRIDE;
}

export function allocatePort(): Promise<number> {
  return pickFreePort(workerBasePort());
}

function pickFreePort(start: number): Promise<number> {
  return new Promise((resolve) => {
    const probe = net.createServer();
    probe.once('error', () => {
      probe.close();
      resolve(pickFreePort(start + 1));
    });
    probe.once('listening', () => {
      probe.close(() => resolve(start));
    });
    probe.listen(start);
  });
}
