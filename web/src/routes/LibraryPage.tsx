import React, { useState, useDeferredValue, useMemo } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Library as LibraryIcon, Plus, Search, Filter, ArrowUpDown } from 'lucide-react';
import { useLibraryManga, useDeleteLibraryMangaMutation } from '../api/hooks';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { Input } from '../components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Skeleton } from '../components/ui/skeleton';
import { LibraryMangaCard } from '../components/LibraryMangaCard';
import { LibraryShelfFilters } from '../components/LibraryShelfFilters';

const sortOptions: Record<string, string> = {
  title_asc: 'Title (A to Z)',
  title_desc: 'Title (Z to A)',
  rating_desc: 'Rating (Highest)',
  rating_asc: 'Rating (Lowest)',
  added_desc: 'Recently Added',
  updated_desc: 'Recently Updated',
};

export const LibraryPage: React.FC = () => {
  const navigate = useNavigate();

  const [activeShelf, setActiveShelf] = useState<string>('all');
  const [selectedTag, setSelectedTag] = useState<string>('');
  const [filterSearch, setFilterSearch] = useState<string>('');
  const [sortBy, setSortBy] = useState<string>('title_asc');

  // 1. Query Library Manga
  const { data: libraryManga = [], isLoading: isLibraryLoading } = useLibraryManga();

  // 2. Delete Manga Mutation
  const deleteMangaMutation = useDeleteLibraryMangaMutation();

  const deferredFilterSearch = useDeferredValue(filterSearch);

  // Shelves & Tags Extraction — memoized to avoid recompute on every filter change
  const { customShelves, allTags } = useMemo(() => {
    const shelves = Array.from(
      new Set(libraryManga.flatMap((m) => m.shelves || []))
    ).filter(Boolean);
    const tags = Array.from(
      new Set(libraryManga.flatMap((m) => m.tags || m.genres || m.meta?.tags || []))
    ).sort();
    return { customShelves: shelves, allTags: tags };
  }, [libraryManga]);

  const defaultShelves = [
    { id: 'all', label: 'All' },
    { id: 'reading', label: 'Reading' },
    { id: 'favorites', label: 'Favorites', isFavorite: true },
    { id: 'plan_to_read', label: 'Plan to Read' },
    { id: 'completed', label: 'Completed' },
    { id: 'on_hold', label: 'On Hold' },
    { id: 'dropped', label: 'Dropped' },
    { id: 'unread', label: 'Unread' },
  ];

  const allShelves = useMemo(() => [
    ...defaultShelves,
    ...customShelves.map((s) => ({ id: `custom:${s}`, label: s })),
  ], [customShelves]);

  // Counts per status and favorites
  const counts = useMemo(() => {
    const res: Record<string, number> = {
      all: libraryManga.length,
      reading: 0,
      favorites: 0,
      plan_to_read: 0,
      completed: 0,
      on_hold: 0,
      dropped: 0,
      unread: 0,
    };
    for (const m of libraryManga) {
      const st = m.userStatus || m.user_status || m.meta?.user_status || 'unread';
      if (res[st] !== undefined) {
        res[st]++;
      }
      if (m.userFavorite || m.user_favorite || m.meta?.user_favorite) {
        res.favorites++;
      }
      if (m.shelves) {
        for (const s of m.shelves) {
          const key = 'custom:' + s;
          res[key] = (res[key] || 0) + 1;
        }
      }
    }
    return res;
  }, [libraryManga]);

  // Filtered and Sorted Manga List
  const filteredManga = useMemo(() => {
    const filtered = libraryManga.filter((manga) => {
      const uStatus = manga.userStatus || manga.user_status || manga.meta?.user_status || 'unread';
      const isFav = manga.userFavorite || manga.user_favorite || manga.meta?.user_favorite;

      if (activeShelf === 'reading' && uStatus !== 'reading') return false;
      if (activeShelf === 'favorites' && !isFav) return false;
      if (activeShelf === 'plan_to_read' && uStatus !== 'plan_to_read') return false;
      if (activeShelf === 'completed' && uStatus !== 'completed') return false;
      if (activeShelf === 'on_hold' && uStatus !== 'on_hold') return false;
      if (activeShelf === 'dropped' && uStatus !== 'dropped') return false;
      if (activeShelf === 'unread' && uStatus !== 'unread') return false;
      if (activeShelf.startsWith('custom:')) {
        const shelfName = activeShelf.replace('custom:', '');
        if (!manga.shelves?.includes(shelfName)) return false;
      }

      if (selectedTag) {
        const mangaTags = manga.tags || manga.genres || manga.meta?.tags || [];
        if (!mangaTags.includes(selectedTag)) return false;
      }

      if (deferredFilterSearch.trim()) {
        const q = deferredFilterSearch.toLowerCase();
        const titleMatch = manga.title.toLowerCase().includes(q);
        const authorList = manga.authors || (manga.author ? [manga.author] : []) || manga.meta?.authors || [];
        const authorMatch = authorList.some((a) => a.toLowerCase().includes(q));
        if (!titleMatch && !authorMatch) return false;
      }

      return true;
    });

    return [...filtered].sort((a, b) => {
      if (sortBy === 'title_asc') {
        return a.title.localeCompare(b.title);
      }
      if (sortBy === 'title_desc') {
        return b.title.localeCompare(a.title);
      }
      if (sortBy === 'rating_desc') {
        const rA = a.userRating || a.user_rating || a.meta?.user_rating || 0;
        const rB = b.userRating || b.user_rating || b.meta?.user_rating || 0;
        if (rB !== rA) return rB - rA;
        return a.title.localeCompare(b.title);
      }
      if (sortBy === 'rating_asc') {
        const rA = a.userRating || a.user_rating || a.meta?.user_rating || 0;
        const rB = b.userRating || b.user_rating || b.meta?.user_rating || 0;
        if (rA !== rB) return rA - rB;
        return a.title.localeCompare(b.title);
      }
      if (sortBy === 'added_desc') {
        const tA = a.meta?.added_at ? new Date(a.meta.added_at).getTime() : 0;
        const tB = b.meta?.added_at ? new Date(b.meta.added_at).getTime() : 0;
        return tB - tA;
      }
      if (sortBy === 'updated_desc') {
        const tA = a.meta?.updated_at ? new Date(a.meta.updated_at).getTime() : 0;
        const tB = b.meta?.updated_at ? new Date(b.meta.updated_at).getTime() : 0;
        return tB - tA;
      }
      return 0;
    });
  }, [libraryManga, activeShelf, selectedTag, deferredFilterSearch, sortBy]);

  return (
    <div className="flex flex-col gap-6">
      {/* Header Bar */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between border-b border-border/50 pb-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">My Library</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {libraryManga.length} {libraryManga.length === 1 ? 'series' : 'series'} saved in library
          </p>
        </div>

        <Button onClick={() => navigate({ to: '/explore' })} className="gap-2 cursor-pointer">
          <Plus className="size-4" aria-hidden />
          Explore Catalog
        </Button>
      </div>

      {/* Shelves Tabs & Filter Bar */}
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          {libraryManga.length > 0 && (
            <LibraryShelfFilters
              shelves={allShelves}
              counts={counts}
              activeShelf={activeShelf}
              onSelectShelf={setActiveShelf}
            />
          )}

          <div className="flex flex-wrap items-center justify-between gap-2 w-full pt-1">
            <div className="flex flex-wrap items-center gap-2">
              {/* Sort Dropdown */}
              <div className="flex items-center gap-1.5">
                <Select value={sortBy} onValueChange={(val) => val && setSortBy(val)}>
                  <SelectTrigger className="w-[170px] text-xs h-9">
                    <ArrowUpDown className="size-3.5 mr-1 text-muted-foreground" />
                    <SelectValue placeholder="Sort by">
                      {sortOptions[sortBy] || sortBy}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="title_asc" className="text-xs">Title (A to Z)</SelectItem>
                    <SelectItem value="title_desc" className="text-xs">Title (Z to A)</SelectItem>
                    <SelectItem value="rating_desc" className="text-xs">Rating (Highest)</SelectItem>
                    <SelectItem value="rating_asc" className="text-xs">Rating (Lowest)</SelectItem>
                    <SelectItem value="added_desc" className="text-xs">Recently Added</SelectItem>
                    <SelectItem value="updated_desc" className="text-xs">Recently Updated</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Tag Filter Dropdown */}
              {allTags.length > 0 && (
                <Select value={selectedTag} onValueChange={(val) => setSelectedTag(!val || val === 'all_tags' ? '' : val)}>
                  <SelectTrigger className="w-[140px] text-xs h-9">
                    <Filter className="size-3.5 mr-1 text-muted-foreground" />
                    <SelectValue placeholder="All Tags">
                      {selectedTag || 'All Tags'}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all_tags" className="text-xs">All Tags</SelectItem>
                    {allTags.map((tag) => (
                      <SelectItem key={tag} value={tag} className="text-xs">
                        {tag}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>

            {/* Quick Filter Input */}
            <div className="relative flex-1 min-w-[200px] sm:max-w-xs">
              <Search className="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" />
              <Input
                type="text"
                placeholder="Search by title or author..."
                value={filterSearch}
                onChange={(e) => setFilterSearch(e.target.value)}
                className="pl-8 text-xs h-9"
              />
            </div>
          </div>
        </div>

        {/* Selected Tag Badge indicator */}
        {selectedTag && (
          <div className="flex items-center gap-2 text-xs">
            <span className="text-muted-foreground">Filtered by tag:</span>
            <Badge variant="secondary" className="gap-1">
              {selectedTag}
              <button
                type="button"
                onClick={() => setSelectedTag('')}
                className="ml-1 text-muted-foreground hover:text-foreground font-bold cursor-pointer"
              >
                ×
              </button>
            </Badge>
          </div>
        )}
      </div>

      {/* Library Grid / Loading / Empty State */}
      {isLibraryLoading ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="flex flex-col gap-2 rounded-lg border border-border bg-card overflow-hidden">
              <Skeleton className="aspect-[2/3] w-full" />
              <div className="p-3">
                <Skeleton className="h-4 w-3/4" />
              </div>
            </div>
          ))}
        </div>
      ) : filteredManga.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-card p-12 text-center">
          <LibraryIcon className="size-12 text-muted-foreground/60 mb-4" aria-hidden />
          <h2 className="text-lg font-semibold text-foreground">
            {libraryManga.length === 0 ? 'Your library is empty' : 'No matching manga found'}
          </h2>
          <p className="mt-1 max-w-sm text-sm text-muted-foreground mb-6">
            {libraryManga.length === 0
              ? 'Add manga series from content sources or search metadata trackers to start building your collection.'
              : 'Try clearing your shelf or tag filters.'}
          </p>
          {libraryManga.length === 0 ? (
            <Button onClick={() => navigate({ to: '/explore' })} className="gap-2 cursor-pointer">
              <Plus className="size-4" aria-hidden />
              Start Exploring
            </Button>
          ) : (
            <Button
              variant="outline"
              onClick={() => {
                setActiveShelf('all');
                setSelectedTag('');
                setFilterSearch('');
              }}
              className="cursor-pointer"
            >
              Clear Filters
            </Button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
          {filteredManga.map((manga) => (
            <LibraryMangaCard
              key={manga.id}
              manga={manga}
              onDelete={(id) => deleteMangaMutation.mutate(id)}
              isDeleting={deleteMangaMutation.isPending}
            />
          ))}
        </div>
      )}
    </div>
  );
};
