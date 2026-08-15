import React, { useState } from 'react';
import { ChevronLeft, ChevronRight, Heart } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger } from './ui/tabs';
import { Button } from './ui/button';

export interface ShelfOption {
  id: string;
  label: string;
  isFavorite?: boolean;
}

export interface LibraryShelfFiltersProps {
  shelves: ShelfOption[];
  counts: Record<string, number>;
  activeShelf: string;
  onSelectShelf: (shelfId: string) => void;
}

export const LibraryShelfFilters: React.FC<LibraryShelfFiltersProps> = ({
  shelves,
  counts,
  activeShelf,
  onSelectShelf,
}) => {
  const [showEmptyFilters, setShowEmptyFilters] = useState<boolean>(false);

  const hasEmptyFilters = shelves.some((s) => s.id !== 'all' && (counts[s.id] || 0) === 0);

  const visibleShelves = shelves.filter(
    (s) =>
      s.id === 'all' ||
      showEmptyFilters ||
      (counts[s.id] || 0) > 0 ||
      s.id === activeShelf
  );

  return (
    <Tabs value={activeShelf} onValueChange={onSelectShelf} className="w-full">
      <TabsList className="flex flex-wrap items-center h-auto p-1 bg-muted/60 gap-1">
        {visibleShelves.map((s) => {
          const count = counts[s.id];
          return (
            <TabsTrigger
              key={s.id}
              value={s.id}
              className="text-xs px-3 py-1.5 gap-1.5 cursor-pointer"
            >
              {s.id === 'favorites' && (
                <Heart className="size-3 text-rose-500 fill-rose-500" />
              )}
              <span>{s.label}</span>
              {count !== undefined && count > 0 && (
                <span className="ml-0.5 rounded-full bg-muted-foreground/15 px-1.5 py-0.2 text-[10px] font-medium text-muted-foreground">
                  {count}
                </span>
              )}
            </TabsTrigger>
          );
        })}
        {hasEmptyFilters && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={() => setShowEmptyFilters((prev) => !prev)}
            className="h-7 w-7 text-muted-foreground hover:text-foreground cursor-pointer"
            title={showEmptyFilters ? 'Hide empty filters' : 'Show empty filters'}
            aria-label={showEmptyFilters ? 'Hide empty filters' : 'Show empty filters'}
          >
            {showEmptyFilters ? (
              <ChevronLeft className="size-4" />
            ) : (
              <ChevronRight className="size-4" />
            )}
          </Button>
        )}
      </TabsList>
    </Tabs>
  );
};
