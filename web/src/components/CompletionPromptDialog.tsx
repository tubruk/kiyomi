import React from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { useUpdateLibraryMangaMutation } from '@/api/hooks';

interface CompletionPromptDialogProps {
  open: boolean;
  mangaId: string;
  mangaTitle?: string;
  onClose: () => void;
}

export const CompletionPromptDialog: React.FC<CompletionPromptDialogProps> = ({
  open,
  mangaId,
  mangaTitle,
  onClose,
}) => {
  const updateMangaMutation = useUpdateLibraryMangaMutation();

  const handleMarkCompleted = () => {
    updateMangaMutation.mutate({
      mangaId,
      fields: {
        user_status: 'completed',
        meta: { user_status: 'completed' },
      },
    });
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>You&apos;ve finished this manga!</DialogTitle>
          <DialogDescription>
            {mangaTitle
              ? `"${mangaTitle}" is now complete. Would you like to mark it as completed?`
              : 'Would you like to mark this manga as completed?'}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Not Now
          </Button>
          <Button onClick={handleMarkCompleted} disabled={updateMangaMutation.isPending}>
            {updateMangaMutation.isPending ? 'Saving...' : 'Mark Completed'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
