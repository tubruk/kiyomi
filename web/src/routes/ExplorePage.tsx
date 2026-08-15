import React from 'react';
import { useNavigate } from '@tanstack/react-router';
import { providerCatalogRoute } from '../router';
import { useSources, useExploreCatalog } from '../api/hooks';
import { Input } from '../components/ui/input';
import { Button } from '../components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Flame, Clock, Search, ChevronLeft, ChevronRight, Compass, X } from 'lucide-react';
import { MangaCard } from '../components/MangaCard';
import { SkeletonCard } from '../components/SkeletonCard';

export const ExplorePage: React.FC = () => {
  const navigate = useNavigate();
  const { providerId } = providerCatalogRoute.useParams();
  const searchParams = providerCatalogRoute.useSearch();

  const mode = searchParams.mode ?? 'popular';
  const query = searchParams.q ?? '';
  const page = searchParams.page ?? 1;

  const [searchTerm, setSearchTerm] = React.useState(query);

  // Sync searchTerm whenever query changes (e.g. from direct URL navigation)
  React.useEffect(() => {
    setSearchTerm(query);
  }, [query]);

  // Fetch Available Content Providers
  const { data: sources = [] } = useSources();

  const activeProviderId = providerId || sources[0]?.id || '';

  // Debouncing effect (500ms)
  React.useEffect(() => {
    if (searchTerm === query) {
      return;
    }

    const timer = setTimeout(() => {
      navigate({
        to: '/providers/$providerId',
        params: { providerId: activeProviderId || 'mangafox' },
        search: (prev) => ({ ...prev, q: searchTerm || undefined, page: 1 }),
      });
    }, 500);

    return () => clearTimeout(timer);
  }, [searchTerm, query, navigate, activeProviderId]);

  // Fetch Catalog / Explore Data
  const {
    data: catalogData,
    isLoading,
    isError,
  } = useExploreCatalog(activeProviderId, mode, query, page, {
    enabled: Boolean(activeProviderId),
  });

  const mangas = catalogData?.mangas || [];
  const hasNext = catalogData?.hasNext ?? false;

  const handleSourceChange = (newSource: string) => {
    navigate({
      to: '/providers/$providerId',
      params: { providerId: newSource },
      search: (prev) => ({ ...prev, page: 1 }),
    });
  };

  const handleModeChange = (newMode: 'popular' | 'latest') => {
    setSearchTerm('');
    navigate({
      to: '/providers/$providerId',
      params: { providerId: activeProviderId || 'mangafox' },
      search: (prev) => ({ ...prev, mode: newMode, q: undefined, page: 1 }),
    });
  };

  const handleClearSearch = () => {
    setSearchTerm('');
    navigate({
      to: '/providers/$providerId',
      params: { providerId: activeProviderId || 'mangafox' },
      search: (prev) => ({ ...prev, q: undefined, page: 1 }),
    });
  };

  return (
    <div className="space-y-6 pb-12">
      {/* Header section */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-border/50 pb-4">
        <div>
          <div className="flex items-center gap-2">
            <Compass className="size-7 text-primary" />
            <h1 className="text-3xl font-bold tracking-tight text-foreground">Explore Catalog</h1>
          </div>
          <p className="text-muted-foreground text-sm mt-1">
            Browse popular rankings and latest manga updates directly from remote providers.
          </p>
        </div>

        {/* Source Selector */}
        {sources.length > 0 && (
          <div className="flex items-center gap-2 bg-card border border-border rounded-lg p-1.5 shadow-xs">
            <span className="text-xs font-semibold px-2 text-muted-foreground uppercase tracking-wider">Source:</span>
            <Select value={activeProviderId} onValueChange={(val) => val && handleSourceChange(val)}>
              <SelectTrigger className="w-[180px] h-8 text-xs border-none bg-transparent shadow-none focus:ring-0">
                <SelectValue placeholder="Select Provider">
                  {sources.find((s) => s.id === activeProviderId)?.name || activeProviderId}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {sources.map((s) => (
                  <SelectItem key={s.id} value={s.id} className="text-xs">
                    {s.name} {s.language || s.lang ? `(${s.language || s.lang})` : ''}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

      {/* Control Bar: Mode Tabs & Search Input */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-muted/40 border border-border p-3 rounded-xl">
        <Tabs value={mode} onValueChange={(v) => handleModeChange(v as 'popular' | 'latest')}>
          <TabsList variant="default" className="bg-card">
            <TabsTrigger value="popular" className="gap-1.5 px-4 text-xs cursor-pointer">
              <Flame className="size-3.5 text-orange-500 fill-orange-500" />
              <span>Popular / Top</span>
            </TabsTrigger>
            <TabsTrigger value="latest" className="gap-1.5 px-4 text-xs cursor-pointer">
              <Clock className="size-3.5 text-blue-500" />
              <span>Latest Updates</span>
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="relative flex-1 sm:max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            placeholder="Search catalog titles..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                navigate({
                  to: '/providers/$providerId',
                  params: { providerId: activeProviderId || 'mangafox' },
                  search: (prev) => ({ ...prev, q: searchTerm || undefined, page: 1 }),
                });
              }
            }}
            className="pl-9 text-xs bg-card border-border pr-8"
          />
          {searchTerm && (
            <button
              type="button"
              onClick={handleClearSearch}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
            >
              <X className="size-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Main Content Area */}
      {isLoading ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
          {Array.from({ length: 12 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
      ) : isError ? (
        <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-8 text-center text-sm text-destructive">
          Failed to fetch catalog from source.
        </div>
      ) : mangas.length === 0 ? (
        <div className="flex flex-col items-center justify-center min-h-[35vh] text-center p-8 border border-dashed rounded-xl bg-card">
          <p className="text-lg font-medium text-foreground">No manga titles found</p>
          <p className="text-sm text-muted-foreground mt-1">Try adjusting your search terms or selecting another provider.</p>
        </div>
      ) : (
        <div className="space-y-8">
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
            {mangas.map((manga) => (
              <MangaCard key={manga.id || manga.url || manga.contentRemoteId} manga={{ ...manga, sourceId: activeProviderId }} isExplore />
            ))}
          </div>

          {/* Pagination Controls */}
          <div className="flex items-center justify-between border-t border-border pt-4">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                navigate({
                  to: '/providers/$providerId',
                  params: { providerId: activeProviderId || 'mangafox' },
                  search: (prev) => ({ ...prev, page: Math.max(1, page - 1) }),
                });
              }}
              disabled={page <= 1 || isLoading}
              className="gap-1 text-xs cursor-pointer"
            >
              <ChevronLeft className="size-4" /> Previous
            </Button>

            <span className="text-xs font-medium text-muted-foreground">
              Page {page}
            </span>

            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                navigate({
                  to: '/providers/$providerId',
                  params: { providerId: activeProviderId || 'mangafox' },
                  search: (prev) => ({ ...prev, page: page + 1 }),
                });
              }}
              disabled={!hasNext || isLoading}
              className="gap-1 text-xs cursor-pointer"
            >
              Next <ChevronRight className="size-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};

