import React from 'react';
import { Skeleton } from './ui/skeleton';

export const SkeletonCard: React.FC = () => {
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <Skeleton className="aspect-[2/3] w-full" />
      <div className="flex flex-col gap-2 p-3">
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-3 w-1/2" />
      </div>
    </div>
  );
};
