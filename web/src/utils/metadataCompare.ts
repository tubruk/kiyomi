import { ExternalLink } from '../types/api';

const normalizeSet = (items?: string[]): Set<string> => {
  return new Set((items || []).map((s) => s.trim().toLowerCase()).filter(Boolean));
};

export const areArraySetsEqual = (a?: string[], b?: string[]): boolean => {
  const setA = normalizeSet(a);
  const setB = normalizeSet(b);
  if (setA.size !== setB.size) return false;
  for (const item of setA) {
    if (!setB.has(item)) return false;
  }
  return true;
};

export const mergeStringArrays = (...arrays: (string[] | undefined)[]): string[] => {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const arr of arrays) {
    if (!arr) continue;
    for (const item of arr) {
      const trimmed = (item ?? '').trim();
      if (!trimmed) continue;
      const lower = trimmed.toLowerCase();
      if (!seen.has(lower)) {
        seen.add(lower);
        result.push(trimmed);
      }
    }
  }
  return result;
};

export const areExternalLinkSetsEqual = (a?: ExternalLink[], b?: ExternalLink[]): boolean => {
  const linksA = (a || []).filter((l) => Boolean(l && l.url?.trim()));
  const linksB = (b || []).filter((l) => Boolean(l && l.url?.trim()));
  if (linksA.length !== linksB.length) return false;
  const setA = new Set(linksA.map((l) => l.url.trim().toLowerCase()));
  const setB = new Set(linksB.map((l) => l.url.trim().toLowerCase()));
  if (setA.size !== setB.size) return false;
  for (const url of setA) {
    if (!setB.has(url)) return false;
  }
  return true;
};

export const mergeExternalLinkArrays = (...arrays: (ExternalLink[] | undefined)[]): ExternalLink[] => {
  const seen = new Set<string>();
  const result: ExternalLink[] = [];
  for (const arr of arrays) {
    if (!arr) continue;
    for (const item of arr) {
      if (!item || !item.url?.trim()) continue;
      const key = item.url.trim().toLowerCase();
      if (!seen.has(key)) {
        seen.add(key);
        result.push({
          provider: item.provider || 'custom',
          label: item.label || item.provider || 'Link',
          url: item.url.trim(),
        });
      }
    }
  }
  return result;
};
