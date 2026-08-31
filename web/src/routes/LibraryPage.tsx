import React from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Library as LibraryIcon, Plus } from 'lucide-react';
import { useLibraryManga, useDeleteLibraryMangaMutation } from '../api/hooks';
import { Button } from '../components/ui/button';
import { Skeleton } from '../components/ui/skeleton';
import { LibraryMangaCard } from '../components/LibraryMangaCard';
import {
  useLibraryFilters,
  LibraryDesktopFilters,
  LibraryMobileFilterSheet,
} from '../components/library';

export const LibraryPage: React.FC = () => {
  const navigate = useNavigate();

  // 1. Query Library Manga
  const { data: libraryManga = [], isLoading: isLibraryLoading } = useLibraryManga();

  // 2. Delete Manga Mutation
  const deleteMangaMutation = useDeleteLibraryMangaMutation();

  // 3. Filter & Sort hook
  const {
    activeShelf,
    setActiveShelf,
    selectedTag,
    setSelectedTag,
    filterSearch,
    setFilterSearch,
    sortBy,
    setSortBy,
    allTags,
    allShelves,
    counts,
    filteredManga,
    clearFilters,
  } = useLibraryFilters({
    libraryManga,
  });

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
        {/* Desktop Filter Layout */}
        <LibraryDesktopFilters
          hasManga={libraryManga.length > 0}
          allShelves={allShelves}
          counts={counts}
          activeShelf={activeShelf}
          onSelectShelf={setActiveShelf}
          sortBy={sortBy}
          onSortByChange={setSortBy}
          allTags={allTags}
          selectedTag={selectedTag}
          onSelectTag={setSelectedTag}
          filterSearch={filterSearch}
          onSearchChange={setFilterSearch}
        />

        {/* Mobile Filter Layout */}
        <LibraryMobileFilterSheet
          hasManga={libraryManga.length > 0}
          allShelves={allShelves}
          counts={counts}
          activeShelf={activeShelf}
          onSelectShelf={setActiveShelf}
          sortBy={sortBy}
          onSortByChange={setSortBy}
          allTags={allTags}
          selectedTag={selectedTag}
          onSelectTag={setSelectedTag}
          filterSearch={filterSearch}
          onSearchChange={setFilterSearch}
        />
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
              onClick={clearFilters}
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
