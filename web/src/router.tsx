import React, { Suspense } from 'react';
import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  redirect,
} from '@tanstack/react-router';
import { QueryClient } from '@tanstack/react-query';
import { RootLayout } from './routes/RootLayout';
import {
  sourcesQueryOptions,
  libraryMangasQueryOptions,
  mangaDetailsQueryOptions,
  providerMangaDetailsQueryOptions,
  exploreCatalogQueryOptions,
  pluginsQueryOptions,
} from './lib/queryOptions';

export interface RouterContext {
  queryClient: QueryClient;
}

export const rootRoute = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
});

const Loading = () => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
  </div>
);

const DefaultNotFoundComponent = () => (
  <div className="flex flex-col items-center justify-center min-h-[50vh] text-center p-8">
    <h2 className="text-2xl font-bold text-foreground">404 - Page Not Found</h2>
    <p className="text-muted-foreground mt-2">The page you are looking for does not exist.</p>
  </div>
);

const DefaultErrorComponent = ({ error }: { error: Error }) => (
  <div className="flex flex-col items-center justify-center min-h-[50vh] text-center p-8">
    <h2 className="text-2xl font-bold text-destructive">An Error Occurred</h2>
    <p className="text-muted-foreground mt-2">{error.message || 'Something went wrong.'}</p>
  </div>
);

const LibraryPage = React.lazy(() => import('./routes/LibraryPage').then(m => ({ default: m.LibraryPage })));
const ExplorePage = React.lazy(() => import('./routes/ExplorePage').then(m => ({ default: m.ExplorePage })));
const DetailsPage = React.lazy(() => import('./routes/DetailsPage').then(m => ({ default: m.DetailsPage })));
const ReaderPage = React.lazy(() => import('./routes/ReaderPage').then(m => ({ default: m.ReaderPage })));
const SettingsPage = React.lazy(() => import('./routes/SettingsPage').then(m => ({ default: m.SettingsPage })));

export interface ExploreSearch {
  mode?: 'popular' | 'latest';
  q?: string;
  page?: number;
}

export const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(libraryMangasQueryOptions()).catch(() => undefined),
  component: () => <Suspense fallback={<Loading />}><LibraryPage /></Suspense>,
});

export const libraryRoute = indexRoute;

export const exploreLandingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/explore',
  loader: async ({ context }) => {
    let defaultProviderId = 'mangafox';
    try {
      const sources = await context.queryClient.fetchQuery(sourcesQueryOptions());
      if (sources && sources.length > 0 && sources[0].id) {
        defaultProviderId = sources[0].id;
      }
    } catch {
      // fallback to default provider
    }
    throw redirect({
      to: '/providers/$providerId',
      params: { providerId: defaultProviderId },
      replace: true,
    });
  },
});

export const providerCatalogRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/providers/$providerId',
  validateSearch: (search: Record<string, unknown>): ExploreSearch => {
    return {
      mode: search.mode === 'latest' ? 'latest' : 'popular',
      q: typeof search.q === 'string' && search.q.trim() ? search.q : undefined,
      page: typeof search.page === 'number' && search.page > 0 ? search.page : Number(search.page) || 1,
    };
  },
  loaderDeps: ({ search: { mode, q, page } }) => ({ mode, q, page }),
  loader: ({ context, params, deps }) => {
    context.queryClient.ensureQueryData(
      exploreCatalogQueryOptions(params.providerId, deps.mode || 'popular', deps.q, deps.page)
    ).catch(() => undefined);
  },
  component: () => <Suspense fallback={<Loading />}><ExplorePage /></Suspense>,
});

export const providerRemoteDetailsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/providers/$providerId/manga/$remoteId',
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(
      providerMangaDetailsQueryOptions(params.providerId, params.remoteId)
    ).catch(() => undefined),
  component: () => <Suspense fallback={<Loading />}><DetailsPage /></Suspense>,
});

export const providerDetailsRoute = providerRemoteDetailsRoute;

export const providerRemoteReaderRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/providers/$providerId/manga/$remoteId/chapter/$chapterId',
  component: () => <Suspense fallback={<Loading />}><ReaderPage /></Suspense>,
});

export const localDetailsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/manga/$mangaId',
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(mangaDetailsQueryOptions(params.mangaId)).catch(() => undefined),
  component: () => <Suspense fallback={<Loading />}><DetailsPage /></Suspense>,
});

export const localReaderRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/manga/$mangaId/chapter/$chapterId',
  component: () => <Suspense fallback={<Loading />}><ReaderPage /></Suspense>,
});

export const exploreProviderRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/explore/$providerId',
  component: () => <Suspense fallback={<Loading />}><ExplorePage /></Suspense>,
});

export const exploreRemoteDetailsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/explore/$providerId/$remoteId',
  component: () => <Suspense fallback={<Loading />}><DetailsPage /></Suspense>,
});

export const readerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reader/$chapterId',
  component: () => <Suspense fallback={<Loading />}><ReaderPage /></Suspense>,
});

export const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  loader: () => {
    throw redirect({
      to: '/settings/plugins',
      replace: true,
    });
  },
});

export const settingsPluginsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/plugins',
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(pluginsQueryOptions()).catch(() => undefined),
  component: () => <Suspense fallback={<Loading />}><SettingsPage /></Suspense>,
});

export const pluginsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/plugins',
  loader: () => {
    throw redirect({
      to: '/settings/plugins',
      replace: true,
    });
  },
});


const routeTree = rootRoute.addChildren([
  indexRoute,
  exploreLandingRoute,
  providerCatalogRoute,
  providerRemoteDetailsRoute,
  providerRemoteReaderRoute,
  localDetailsRoute,
  localReaderRoute,
  exploreProviderRoute,
  exploreRemoteDetailsRoute,
  readerRoute,
  settingsRoute,
  settingsPluginsRoute,
  pluginsRoute,
]);

export const router = createRouter({
  routeTree,
  context: { queryClient: undefined! },
  defaultPreload: 'intent',
  defaultNotFoundComponent: DefaultNotFoundComponent,
  defaultErrorComponent: DefaultErrorComponent,
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

