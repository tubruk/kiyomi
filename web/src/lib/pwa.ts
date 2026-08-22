import React, { useEffect } from 'react';
import { registerSW } from 'virtual:pwa-register';
import { useToast } from '../context/ToastContext';

export const isE2E = (): boolean => {
  if (typeof window === 'undefined') return false;
  return (
    import.meta.env.MODE === 'test' ||
    window.location.search.includes('e2e=true') ||
    Boolean((window as unknown as { __playwright?: boolean }).__playwright) ||
    Boolean(navigator.webdriver)
  );
};

export type RefreshHandler = (reload: () => void) => void;

let updateSWFn: ((reloadPage?: boolean) => Promise<void>) | null = null;
let onNeedRefreshHandler: RefreshHandler | null = null;
let pendingReload: (() => void) | null = null;

export const setOnNeedRefreshHandler = (handler: RefreshHandler) => {
  onNeedRefreshHandler = handler;
  if (pendingReload) {
    const reload = pendingReload;
    pendingReload = null;
    handler(reload);
  }
};

export const initServiceWorker = () => {
  if (isE2E() || typeof window === 'undefined' || !('serviceWorker' in navigator)) {
    return;
  }

  // Reload page when new service worker takes over control
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    window.location.reload();
  });

  updateSWFn = registerSW({
    immediate: true,
    onNeedRefresh() {
      const triggerReload = () => {
        if (updateSWFn) {
          updateSWFn(true);
        }
      };

      if (onNeedRefreshHandler) {
        onNeedRefreshHandler(triggerReload);
      } else {
        pendingReload = triggerReload;
      }
    },
    onOfflineReady() {
      // offline cache ready
    },
  });
};

export const PWARegistrationManager: React.FC = () => {
  const { showToast } = useToast();

  useEffect(() => {
    setOnNeedRefreshHandler((reload) => {
      showToast(
        'A new version of Kiyomi is available.',
        'info',
        undefined,
        'default',
        {
          label: 'Reload',
          onClick: reload,
        }
      );
    });
  }, [showToast]);

  return null;
};
