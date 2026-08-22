import React, { createContext, useContext, useState, useCallback } from 'react';

export type ToastType = 'info' | 'error' | 'success';

export interface ToastAction {
  label: string;
  onClick: () => void;
}

export interface ToastMessage {
  id: string;
  message: string;
  type: ToastType;
  details?: string;
  /** 'default' shows full toast with dismiss button. 'subtle' shows minimal inline toast above bottom bar. */
  mode?: 'default' | 'subtle';
  action?: ToastAction;
}

interface ToastContextValue {
  toasts: ToastMessage[];
  showToast: (
    message: string,
    type?: ToastType,
    details?: string,
    mode?: 'default' | 'subtle',
    action?: ToastAction,
    durationMs?: number
  ) => void;
  removeToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined);

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const showToast = useCallback(
    (
      message: string,
      type: ToastType = 'info',
      details?: string,
      mode: 'default' | 'subtle' = 'default',
      action?: ToastAction,
      durationMs?: number
    ) => {
      const id = Math.random().toString(36).substring(2, 9);
      setToasts((prev) => [...prev, { id, message, type, details, mode, action }]);

      const timeout = durationMs ?? (action ? 10000 : 3200);
      setTimeout(() => {
        removeToast(id);
      }, timeout);
    },
    [removeToast]
  );

  return (
    <ToastContext.Provider value={{ toasts, showToast, removeToast }}>
      {children}
    </ToastContext.Provider>
  );
};

export const useToast = () => {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return ctx;
};
