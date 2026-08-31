import { describe, it, expect } from 'vitest';
import { getProxyImageUrl } from './image';

describe('utils/image getProxyImageUrl', () => {
  it('returns undefined if url is not provided or empty', () => {
    expect(getProxyImageUrl(undefined)).toBeUndefined();
    expect(getProxyImageUrl('')).toBeUndefined();
  });

  it('proxies http and https URLs', () => {
    expect(getProxyImageUrl('http://example.com/cover.jpg')).toBe(
      '/api/v1/proxy/image?url=http%3A%2F%2Fexample.com%2Fcover.jpg'
    );
    expect(getProxyImageUrl('https://example.com/cover.png')).toBe(
      '/api/v1/proxy/image?url=https%3A%2F%2Fexample.com%2Fcover.png'
    );
  });

  it('returns non-http(s) urls as-is', () => {
    expect(getProxyImageUrl('/local/path.jpg')).toBe('/local/path.jpg');
    expect(getProxyImageUrl('data:image/png;base64,...')).toBe('data:image/png;base64,...');
  });
});
