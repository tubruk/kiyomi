import React from 'react';
import { ArrowUpDown, Filter, Search } from 'lucide-react';
import { Input } from '../ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Badge } from '../ui/badge';
import { LibraryShelfFilters } from '../LibraryShelfFilters';
import { ShelfItem, LIBRARY_SORT_OPTIONS } from './hooks/useLibraryFilters';

export interface LibraryDesktopFiltersProps {
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

export const LibraryDesktopFilters: React.FC<LibraryDesktopFiltersProps> = ({
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
    <div className="hidden md:flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        {hasManga && (
          <LibraryShelfFilters
            shelves={allShelves}
            counts={counts}
            activeShelf={activeShelf}
            onSelectShelf={onSelectShelf}
          />
        )}

        <div className="flex flex-wrap items-center justify-between gap-2 w-full pt-1">
          <div className="flex flex-wrap items-center gap-2">
            {/* Sort Dropdown */}
            <div className="flex items-center gap-1.5">
              <Select value={sortBy} onValueChange={(val) => val && onSortByChange(val)}>
                <SelectTrigger className="w-[170px] text-xs h-9">
                  <ArrowUpDown className="size-3.5 mr-1 text-muted-foreground" />
                  <SelectValue placeholder="Sort by">
                    {LIBRARY_SORT_OPTIONS[sortBy] || sortBy}
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
              <Select
                value={selectedTag}
                onValueChange={(val) => onSelectTag(!val || val === 'all_tags' ? '' : val)}
              >
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
              onChange={(e) => onSearchChange(e.target.value)}
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
              onClick={() => onSelectTag('')}
              className="ml-1 text-muted-foreground hover:text-foreground font-bold cursor-pointer"
            >
              ×
            </button>
          </Badge>
        </div>
      )}
    </div>
  );
};
