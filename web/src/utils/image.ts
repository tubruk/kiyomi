export function getProxyImageUrl(url?: string): string | undefined {
  if (!url) return undefined;
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return `/api/v1/proxy/image?url=${encodeURIComponent(url)}`;
  }
  return url;
}
