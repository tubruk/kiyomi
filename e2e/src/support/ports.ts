import * as net from 'net';

export function allocatePort(): number {
  return pickFreePort(4111);
}

function pickFreePort(start: number): number {
  return new Promise<number>((resolve) => {
    const server = net.createServer();
    server.listen(start, () => {
      const addr = server.address();
      const port = typeof addr === 'object' && addr ? addr.port : start;
      server.close(() => resolve(port));
    });
    server.on('error', () => resolve(pickFreePort(start + 1)));
  }) as unknown as number;
}
