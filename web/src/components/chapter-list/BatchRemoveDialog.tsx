import React from 'react';
import { Loader2 } from 'lucide-react';
import { Button } from '../ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';

export interface BatchRemoveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedCount: number;
  onConfirm: () => void;
  isRemoving: boolean;
}

export const BatchRemoveDialog: React.FC<BatchRemoveDialogProps> = ({
  open,
  onOpenChange,
  selectedCount,
  onConfirm,
  isRemoving,
}) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Remove Chapters</DialogTitle>
          <DialogDescription>
            Are you sure you want to remove {selectedCount} selected chapter{selectedCount === 1 ? '' : 's'} from your library?
            This will delete their library entries and all associated downloaded files.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2 sm:gap-0">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isRemoving}
            className="cursor-pointer"
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={isRemoving}
            className="gap-1.5 cursor-pointer"
          >
            {isRemoving && <Loader2 className="size-3.5 animate-spin" />}
            Remove {selectedCount} Chapter{selectedCount === 1 ? '' : 's'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
