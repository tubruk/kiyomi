import { describe, it, expect } from 'vitest';
import { queryKeys } from './queryKeys';

describe('queryKeys', () => {
  it('generates sources keys', () => {
    expect(queryKeys.sources.all).toEqual(['sources']);
  });

  it('generates library keys', () => {
    expect(queryKeys.library.all).toEqual(['library']);
    expect(queryKeys.library.mangas()).toEqual(['library', 'manga']);
    expect(queryKeys.library.pull('manga-1')).toEqual(['library', 'pull', 'manga-1']);
    expect(queryKeys.library.pull('manga-1', 'prov-1')).toEqual(['library', 'pull', 'manga-1', 'prov-1']);
    expect(queryKeys.library.refresh('manga-1')).toEqual(['library', 'refresh', 'manga-1']);
  });

  it('generates manga keys', () => {
    expect(queryKeys.manga.all).toEqual(['manga']);
    expect(queryKeys.manga.details('m-1')).toEqual(['manga', 'detail', 'm-1']);
    expect(queryKeys.manga.providerDetails('p-1', 'r-1')).toEqual(['manga', 'provider', 'p-1', 'r-1']);
  });

  it('generates explore keys', () => {
    expect(queryKeys.explore.all).toEqual(['explore']);
    expect(queryKeys.explore.catalog('p1', 'popular', 'naruto', 1)).toEqual([
      'explore',
      'p1',
      'popular',
      'naruto',
      1,
    ]);
  });

  it('generates chapters keys', () => {
    expect(queryKeys.chapters.all).toEqual(['chapters']);
    expect(queryKeys.chapters.list('m1')).toEqual(['chapters', 'm1']);
    expect(queryKeys.chapters.providerList('m1', 'p1')).toEqual(['chapters', 'm1', 'provider', 'p1']);
    expect(queryKeys.chapters.remoteList('p1', 'r1')).toEqual(['chapters', 'remote', 'p1', 'r1']);
    expect(queryKeys.chapters.pages('c1')).toEqual(['chapters', 'pages', 'c1']);
    expect(queryKeys.chapters.pages('c1', 'm1', 'p1')).toEqual(['chapters', 'pages', 'c1', 'm1', 'p1']);
  });

  it('generates plugins and collisions keys', () => {
    expect(queryKeys.plugins.all).toEqual(['plugins']);
    expect(queryKeys.plugins.logs('plug-1')).toEqual(['plugins', 'logs', 'plug-1']);
    expect(queryKeys.collisions.all).toEqual(['collisions']);
  });

  it('generates jobs keys', () => {
    expect(queryKeys.jobs.all).toEqual(['jobs']);
    expect(queryKeys.jobs.list()).toEqual(['jobs', 'list']);
    expect(queryKeys.jobs.list({ status: 'running' })).toEqual(['jobs', 'list', { status: 'running' }]);
    expect(queryKeys.jobs.children('parent-1')).toEqual(['jobs', 'children', 'parent-1']);
  });
});
