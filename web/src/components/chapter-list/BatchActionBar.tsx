import React from 'react';
import {
  BookOpen,
  CheckCheck,
  Download,
  FileX,
  Loader2,
  RefreshCw,
  Trash2,
  X,
} from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';

export interface BatchActionBarProps {
  selectedCount: number;
  totalCount: number;
  hasDownloadedInSelection: boolean;
  onSelectAll: () => void;
  onSelectNone: () => void;
  onInvertSelection: () => void;
  onExitSelectionMode: () => void;
  onBatchUpdateProgress?: (progress: { is_read?: boolean; last_read_page?: number }) => void;
  isBatchUpdatingProgress?: boolean;
  onBatchPull?: () => void;
  isBatchPulling?: boolean;
  onBatchRefresh?: () => void;
  isBatchRefreshing?: boolean;
  onBatchDeleteFiles?: () => void;
  isBatchDeletingFiles?: boolean;
  onOpenRemoveDialog: () => void;
  isBatchRemoving?: boolean;
  canPull?: boolean;
  canDeleteFiles?: boolean;
  canRemove?: boolean;
}

export const BatchActionBar: React.FC<BatchActionBarProps> = ({
  selectedCount,
  totalCount: _totalCount,
  hasDownloadedInSelection,
  onSelectAll,
  onSelectNone,
  onInvertSelection,
  onExitSelectionMode,
  onBatchUpdateProgress,
  isBatchUpdatingProgress = false,
  onBatchPull,
  isBatchPulling = false,
  onBatchRefresh,
  isBatchRefreshing = false,
  onBatchDeleteFiles,
  isBatchDeletingFiles = false,
  onOpenRemoveDialog,
  isBatchRemoving = false,
  canPull = true,
  canDeleteFiles = true,
  canRemove = true,
}) => {
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-primary/30 bg-primary/5 p-3.5 sm:p-4 transition-all">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={onExitSelectionMode}
            className="h-8 px-2.5 text-xs gap-1.5 cursor-pointer font-medium hover:bg-primary/10"
            aria-label="Exit selection mode"
          >
            <X className="size-4" />
            Done
          </Button>

          <Badge variant="secondary" className="font-semibold text-xs px-2.5 py-1">
            {selectedCount} selected
          </Badge>

          <div className="h-4 w-px bg-border/60 mx-1 hidden sm:block" />

          <Button
            variant="outline"
            size="sm"
            onClick={onSelectAll}
            className="h-7 text-xs px-2.5 bg-background border-border cursor-pointer"
          >
            Select All
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onSelectNone}
            disabled={selectedCount === 0}
            className="h-7 text-xs px-2.5 bg-background border-border cursor-pointer disabled:opacity-50"
          >
            Select None
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onInvertSelection}
            className="h-7 text-xs px-2.5 bg-background border-border cursor-pointer"
          >
            Invert
          </Button>
        </div>

        {/* Shared Actions */}
        <div className="flex flex-wrap items-center gap-2 pt-2 sm:pt-0 border-t sm:border-t-0 border-border/40">
          {onBatchUpdateProgress && (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onBatchUpdateProgress({ is_read: true, last_read_page: 0 })}
                disabled={selectedCount === 0 || isBatchUpdatingProgress}
                className="h-8 text-xs bg-background border-border gap-1.5 cursor-pointer"
                title="Mark selected chapters as read"
              >
                {isBatchUpdatingProgress ? (
                  <Loader2 className="size-3.5 animate-spin" />
                ) : (
                  <CheckCheck className="size-3.5 text-emerald-500" />
                )}
                Mark Read
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onBatchUpdateProgress({ is_read: false, last_read_page: 0 })}
                disabled={selectedCount === 0 || isBatchUpdatingProgress}
                className="h-8 text-xs bg-background border-border gap-1.5 cursor-pointer"
                title="Mark selected chapters as unread"
              >
                {isBatchUpdatingProgress ? (
                  <Loader2 className="size-3.5 animate-spin" />
                ) : (
                  <BookOpen className="size-3.5" />
                )}
                Mark Unread
              </Button>
            </>
          )}

          {onBatchRefresh && (
            <Button
              variant="outline"
              size="sm"
              onClick={onBatchRefresh}
              disabled={selectedCount === 0 || isBatchRefreshing}
              className="h-8 text-xs bg-background border-border gap-1.5 cursor-pointer"
              title="Refresh metadata of selected chapters from provider"
              aria-label="Refresh metadata of selected chapters from provider"
            >
              {isBatchRefreshing ? (
                <RefreshCw className="size-3.5 animate-spin" />
              ) : (
                <RefreshCw className="size-3.5" />
              )}
              Refresh
            </Button>
          )}

          {onBatchPull && canPull && (
            <Button
              variant="outline"
              size="sm"
              onClick={onBatchPull}
              disabled={selectedCount === 0 || isBatchPulling}
              className="h-8 text-xs bg-background border-border gap-1.5 cursor-pointer"
              title="Enqueue download jobs for selected chapters"
            >
              {isBatchPulling ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Download className="size-3.5" />
              )}
              Pull
            </Button>
          )}

          {onBatchDeleteFiles && canDeleteFiles && (
            <Button
              variant="outline"
              size="sm"
              onClick={onBatchDeleteFiles}
              disabled={
                selectedCount === 0 ||
                !hasDownloadedInSelection ||
                isBatchDeletingFiles
              }
              className="h-8 text-xs bg-background border-border gap-1.5 cursor-pointer disabled:opacity-50"
              title={
                !hasDownloadedInSelection
                  ? 'No downloaded files in selection'
                  : 'Delete downloaded files for selected chapters'
              }
            >
              {isBatchDeletingFiles ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <FileX className="size-3.5" />
              )}
              Delete Files
            </Button>
          )}

          {onOpenRemoveDialog && canRemove && (
            <Button
              variant="destructive"
              size="sm"
              onClick={onOpenRemoveDialog}
              disabled={selectedCount === 0 || isBatchRemoving}
              className="h-8 text-xs gap-1.5 cursor-pointer disabled:opacity-50"
              title="Remove selected chapters from library"
            >
              {isBatchRemoving ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Trash2 className="size-3.5" />
              )}
              Remove
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};
