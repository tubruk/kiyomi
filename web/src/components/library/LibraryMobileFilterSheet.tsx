import React from 'react';
import { ArrowUpDown, Filter, Search } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '../ui/sheet';
import { ShelfItem } from './hooks/useLibraryFilters';

export interface LibraryMobileFilterSheetProps {
  hasManga: boolean;
  allShelves: ShelfItem[];
  counts: Record<string, number>;
  activeShelf: string;
  onSelectShelf: (shelf: string) => void;
  sortBy: string;
  onSortByChange: (sortBy: string) => void;
  allTags: string[];
  selectedTag: string;
  onSelectTag: (tag: string) => void;
  filterSearch: string;
  onSearchChange: (search: string) => void;
}

export const LibraryMobileFilterSheet: React.FC<LibraryMobileFilterSheetProps> = ({
  hasManga,
  allShelves,
  counts,
  activeShelf,
  onSelectShelf,
  sortBy,
  onSortByChange,
  allTags,
  selectedTag,
  onSelectTag,
  filterSearch,
  onSearchChange,
}) => {
  return (
    <div className="flex md:hidden items-center gap-2 w-full">
      <div className="relative flex-1">
        <Search className="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" />
        <Input
          type="text"
          placeholder="Search..."
          value={filterSearch}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-8 text-xs h-9"
        />
      </div>

      <Sheet>
        <SheetTrigger
          render={
            <Button
              variant="outline"
              size="sm"
              className="h-9 gap-1.5 text-xs cursor-pointer shrink-0"
            >
              <Filter className="size-3.5" />
              <span>Filter & Sort</span>
            </Button>
          }
        />
        <SheetContent side="bottom" className="h-[80vh] rounded-t-xl sm:max-w-full">
          <SheetHeader>
            <SheetTitle>Filter & Sort</SheetTitle>
          </SheetHeader>
          <div className="mt-4 flex flex-col gap-6 overflow-y-auto max-h-[calc(80vh-100px)] pb-10">
            {/* Sorting */}
            <div className="flex flex-col gap-2">
              <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Sort By
              </span>
              <Select value={sortBy} onValueChange={(val) => val && onSortByChange(val)}>
                <SelectTrigger className="w-full text-xs h-10">
                  <ArrowUpDown className="size-3.5 mr-1 text-muted-foreground" />
                  <SelectValue placeholder="Sort by" />
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

            {/* Shelves Selection */}
            {hasManga && (
              <div className="flex flex-col gap-2">
                <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Shelf / Status
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {allShelves.map((shelf) => {
                    const isActive = activeShelf === shelf.id;
                    const count = counts[shelf.id] ?? 0;
                    return (
                      <button
                        key={shelf.id}
                        type="button"
                        onClick={() => onSelectShelf(shelf.id)}
                        className={`rounded-full px-3 py-1 text-xs font-medium border cursor-pointer transition-colors ${
                          isActive
                            ? 'bg-primary text-primary-foreground border-primary'
                            : 'bg-card text-foreground border-border hover:bg-accent'
                        }`}
                      >
                        {shelf.label} ({count})
                      </button>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Tag Filter */}
            {allTags.length > 0 && (
              <div className="flex flex-col gap-2">
                <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Filter by Tag
                </span>
                <div className="flex flex-wrap gap-1.5">
                  <button
                    type="button"
                    onClick={() => onSelectTag('')}
                    className={`rounded-full px-3 py-1 text-xs font-medium border cursor-pointer transition-colors ${
                      !selectedTag
                        ? 'bg-primary text-primary-foreground border-primary'
                        : 'bg-card text-foreground border-border hover:bg-accent'
                    }`}
                  >
                    All Tags
                  </button>
                  {allTags.map((tag) => {
                    const isActive = selectedTag === tag;
                    return (
                      <button
                        key={tag}
                        type="button"
                        onClick={() => onSelectTag(tag)}
                        className={`rounded-full px-3 py-1 text-xs font-medium border cursor-pointer transition-colors ${
                          isActive
                            ? 'bg-primary text-primary-foreground border-primary'
                            : 'bg-card text-foreground border-border hover:bg-accent'
                        }`}
                      >
                        {tag}
                      </button>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
};
