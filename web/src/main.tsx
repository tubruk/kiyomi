import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider, useIsRestoring } from '@tanstack/react-query';
import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client';
import { RouterProvider } from '@tanstack/react-router';
import { router } from './router';
import { TooltipProvider } from './components/ui/tooltip';
import { initServiceWorker, isE2E } from './lib/pwa';
import { persister, shouldDehydrateQuery, CACHE_BUSTER } from './lib/persister';
import './index.css';

// Initialize Service Worker unless running under E2E testing
initServiceWorker();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      gcTime: 1000 * 60 * 60 * 24, // 24 hours persistence
      staleTime: 1000 * 60 * 5, // 5 minutes fresh window
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

const AppContent: React.FC = () => {
  const isRestoring = useIsRestoring();

  if (isRestoring) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  return (
    <TooltipProvider>
      <RouterProvider router={router} context={{ queryClient }} />
    </TooltipProvider>
  );
};

const App: React.FC = () => {
  const e2eMode = isE2E();

  if (e2eMode) {
    return (
      <QueryClientProvider client={queryClient}>
        <AppContent />
      </QueryClientProvider>
    );
  }

  return (
    <PersistQueryClientProvider
      client={queryClient}
      persistOptions={{
        persister,
        buster: CACHE_BUSTER,
        maxAge: 1000 * 60 * 60 * 24, // 24 hours
        dehydrateOptions: {
          shouldDehydrateQuery,
        },
      }}
    >
      <AppContent />
    </PersistQueryClientProvider>
  );
};

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

