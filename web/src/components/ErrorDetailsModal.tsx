import React, { useState } from 'react';
import { AlertCircle, Copy, Check } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from './ui/dialog';
import { Button } from './ui/button';

export interface ErrorDetailsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title?: string;
  message?: string;
  details?: string;
}

export const ErrorDetailsModal: React.FC<ErrorDetailsModalProps> = ({
  open,
  onOpenChange,
  title,
  message,
  details,
}) => {
  const [copiedMessage, setCopiedMessage] = useState(false);
  const [copiedDetails, setCopiedDetails] = useState(false);

  const handleCopyMessage = async () => {
    const textToCopy = message || title || '';
    if (textToCopy) {
      try {
        await navigator.clipboard.writeText(textToCopy);
        setCopiedMessage(true);
        setTimeout(() => setCopiedMessage(false), 2000);
      } catch (err) {
        console.error('Failed to copy error message:', err);
      }
    }
  };

  const handleCopyFull = async () => {
    const parts: string[] = [];
    if (title) parts.push(`Title: ${title}`);
    if (message) parts.push(`Message: ${message}`);
    if (details) parts.push(`Details:\n${details}`);
    const textToCopy = parts.length > 0 ? parts.join('\n\n') : (details || message || title || '');
    if (textToCopy) {
      try {
        await navigator.clipboard.writeText(textToCopy);
        setCopiedDetails(true);
        setTimeout(() => setCopiedDetails(false), 2000);
      } catch (err) {
        console.error('Failed to copy full error:', err);
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl sm:max-w-xl">
        <DialogHeader>
          <div className="flex items-center gap-2 text-destructive">
            <AlertCircle className="size-5 shrink-0" />
            <DialogTitle className="text-base font-semibold text-foreground">
              {title || 'Error Details'}
            </DialogTitle>
          </div>
          {message && (
            <DialogDescription className="text-xs text-muted-foreground pt-1">
              {message}
            </DialogDescription>
          )}
        </DialogHeader>

        <div className="py-1">
          <pre className="max-h-72 overflow-y-auto rounded-lg bg-muted/60 p-4 text-xs font-mono text-foreground whitespace-pre-wrap break-all border border-border">
            {details || message}
          </pre>
        </div>

        <DialogFooter className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopyMessage}
            className="gap-1.5 text-xs cursor-pointer"
          >
            {copiedMessage ? (
              <>
                <Check className="size-3.5 text-emerald-500" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="size-3.5" />
                Copy Message
              </>
            )}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopyFull}
            className="gap-1.5 text-xs cursor-pointer"
          >
            {copiedDetails ? (
              <>
                <Check className="size-3.5 text-emerald-500" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="size-3.5" />
                Copy Full Error
              </>
            )}
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={() => onOpenChange(false)}
            className="text-xs cursor-pointer"
          >
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
