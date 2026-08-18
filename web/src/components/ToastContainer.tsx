import React, { useState } from 'react';
import { useToast, ToastMessage } from '../context/ToastContext';
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { ErrorDetailsModal } from './ErrorDetailsModal';

export const ToastContainer: React.FC = () => {
  const { toasts, removeToast } = useToast();
  const [selectedToast, setSelectedToast] = useState<ToastMessage | null>(null);

  if (toasts.length === 0 && !selectedToast) return null;

  return (
    <>
      <div className="fixed bottom-20 sm:bottom-4 left-4 sm:left-auto sm:right-4 z-50 flex flex-col gap-2 sm:max-w-sm w-full sm:w-auto pointer-events-none items-center sm:items-end">
        {toasts.map((t) => {
          if (t.mode === 'subtle') {
            return (
              <div
                key={t.id}
                className="pointer-events-auto mx-auto rounded-md border border-border bg-card/90 backdrop-blur-sm px-3 py-1.5 text-[11px] font-medium text-foreground shadow-lg transition-all duration-200 animate-in slide-in-from-bottom-2 max-w-[200px] text-center"
              >
                <span>{t.message}</span>
              </div>
            );
          }

          const isError = t.type === 'error' || Boolean(t.details);
          return (
            <div
              key={t.id}
              onClick={() => {
                if (isError) {
                  setSelectedToast(t);
                }
              }}
              className={cn(
                'pointer-events-auto flex items-center gap-3 rounded-lg border px-4 py-3 text-xs font-medium shadow-lg transition-all duration-200 animate-in slide-in-from-bottom-2',
                isError && 'cursor-pointer hover:border-destructive/50',
                t.type === 'success' && 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
                t.type === 'error' && 'border-destructive/30 bg-destructive/10 text-destructive',
                t.type === 'info' && 'border-border bg-card text-foreground'
              )}
            >
              {t.type === 'success' && <CheckCircle2 className="size-4 shrink-0 text-emerald-500" />}
              {t.type === 'error' && <AlertCircle className="size-4 shrink-0 text-destructive" />}
              {t.type === 'info' && <Info className="size-4 shrink-0 text-primary" />}

              <span className="flex-1 line-clamp-2">{t.message}</span>

              {isError && (
                <Button
                  type="button"
                  variant="ghost"
                  size="xs"
                  onClick={(e) => {
                    e.stopPropagation();
                    setSelectedToast(t);
                  }}
                  className="h-6 px-1.5 text-[10px] font-semibold text-destructive hover:bg-destructive/10 cursor-pointer"
                >
                  Details
                </Button>
              )}

              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  removeToast(t.id);
                }}
                className="shrink-0 opacity-60 hover:opacity-100 transition-opacity cursor-pointer"
                aria-label="Dismiss toast"
              >
                <X className="size-3.5" />
              </button>
            </div>
          );
        })}
      </div>

      <ErrorDetailsModal
        open={Boolean(selectedToast)}
        onOpenChange={(open) => {
          if (!open) setSelectedToast(null);
        }}
        title="Error Details"
        message={selectedToast?.message}
        details={selectedToast?.details}
      />
    </>
  );
};
