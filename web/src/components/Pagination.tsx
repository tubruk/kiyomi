import React from 'react';
import { Button } from './ui/button';
import { ChevronLeft, ChevronRight } from 'lucide-react';

export interface PaginationProps {
  currentPage: number;
  hasNextPage?: boolean;
  totalPages?: number;
  onPageChange: (newPage: number) => void;
}

export const Pagination: React.FC<PaginationProps> = ({
  currentPage,
  hasNextPage,
  totalPages,
  onPageChange,
}) => {
  const canGoNext = totalPages !== undefined ? currentPage < totalPages : Boolean(hasNextPage);

  return (
    <div className="flex items-center justify-center gap-4 mt-8">
      <Button
        variant="outline"
        size="sm"
        disabled={currentPage <= 1}
        onClick={() => onPageChange(currentPage - 1)}
      >
        <ChevronLeft className="size-4" aria-hidden />
        Previous
      </Button>
      <span className="text-xs font-medium text-muted-foreground">
        {totalPages !== undefined ? `Page ${currentPage} of ${totalPages}` : `Page ${currentPage}`}
      </span>
      <Button
        variant="outline"
        size="sm"
        disabled={!canGoNext}
        onClick={() => onPageChange(currentPage + 1)}
      >
        Next
        <ChevronRight className="size-4" aria-hidden />
      </Button>
    </div>
  );
};
