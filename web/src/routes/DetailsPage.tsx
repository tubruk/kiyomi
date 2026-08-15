import React, { useState, useEffect, useMemo } from 'react';
import { useParams, Link, useLocation, useNavigate } from '@tanstack/react-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  ArrowLeft,
  BookOpen,
  Edit3,
  Plus,
  Trash2,
  ChevronDown,
  ChevronUp,
  Star,
  Heart,
  Play,
  MoreVertical,
} from 'lucide-react';
import { api } from '../api/client';
import {
  useLibraryManga,
  useMangaDetails,
  useProviderMangaDetails,
  useSources,
  useChapterList,
  useProviderChapterList,
  useUpdateLibraryMangaMutation,
  useDeleteLibraryMangaMutation,
} from '../api/hooks';
import { useToast } from '../context/ToastContext';
import { Manga, UserStatus, Chapter } from '../types/api';
import { getProxyImageUrl } from '../lib/utils';
import { queryKeys } from '../lib/queryKeys';
import { Button } from '../components/ui/button';
import { GenrePill } from '../components/GenrePill';
import { ChapterList } from '../components/ChapterList';
import { EditMetadataDialog } from '../components/EditMetadataDialog';
import { Card } from '../components/ui/card';
import { Skeleton } from '../components/ui/skeleton';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../components/ui/dropdown-menu';
import { cn } from '../lib/utils';

export const STATUS_OPTIONS: { value: UserStatus; label: string }[] = [
  { value: 'reading', label: 'Reading' },
  { value: 'plan_to_read', label: 'Plan to Read' },
  { value: 'completed', label: 'Completed' },
  { value: 'on_hold', label: 'On Hold' },
  { value: 'dropped', label: 'Dropped' },
  { value: 'unread', label: 'Unread' },
];

export const READING_MODE_LABELS: Record<string, string> = {
  rtl: 'Right to Left (Manga)',
  ltr: 'Left to Right (Comic)',
  vertical: 'Vertical (Gapped)',
  longstrip: 'Longstrip (Webtoon)',
};

const formatStatus = (status?: string) => {
  if (!status) return 'Unread';
  const found = STATUS_OPTIONS.find((s) => s.value === status);
  if (found) return found.label;
  return status
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
};

export const DetailsPage: React.FC = () => {
  const queryClient = useQueryClient();
  const location = useLocation();
  const navigate = useNavigate();
  const { showToast } = useToast();
  const params = useParams({ strict: false }) as { mangaId?: string; providerId?: string; remoteId?: string };


  const isRemoteRoute = location.pathname.startsWith('/explore/') || location.pathname.startsWith('/providers/') || Boolean(params.providerId && params.remoteId);
  const providerIdParam = params.providerId || '';
  const remoteIdParam = params.remoteId || '';

  const [sortBy, setSortBy] = useState('source');
  const [order, setOrder] = useState<'asc' | 'desc'>('desc');

  const [isEditMetadataOpen, setIsEditMetadataOpen] = useState(false);
  const [showDetailedMetadata, setShowDetailedMetadata] = useState(false);
  const [localUserNotes, setLocalUserNotes] = useState('');
  const [isEditingNotes, setIsEditingNotes] = useState(false);
  const [hoverRating, setHoverRating] = useState<number | null>(null);

  // 1. Fetch Library List to check if in Library
  const { data: libraryManga = [] } = useLibraryManga();

  const libraryEntry = isRemoteRoute
    ? libraryManga.find(
        (m) =>
          (m.contentProviderId === providerIdParam || m.sourceId === providerIdParam || m.meta?.content?.provider_id === providerIdParam) &&
          (m.contentRemoteId === remoteIdParam || m.url === remoteIdParam || m.id === remoteIdParam || m.meta?.content?.manga_id === remoteIdParam)
      )
    : libraryManga.find((m) => m.id === params.mangaId);

  const isInLibrary = Boolean(libraryEntry);
  const targetMangaId = libraryEntry?.id || params.mangaId || '';

  // 2. Fetch Manga Details (Remote vs Local)
  const { data: localDetailsManga, isLoading: isLocalMangaLoading } = useMangaDetails(targetMangaId, {
    enabled: !isRemoteRoute && Boolean(targetMangaId),
  });

  const { data: remoteDetailsManga, isLoading: isRemoteMangaLoading } = useProviderMangaDetails(
    providerIdParam,
    remoteIdParam,
    { enabled: isRemoteRoute && Boolean(providerIdParam && remoteIdParam) }
  );

  const manga = isRemoteRoute ? remoteDetailsManga : localDetailsManga;
  const isMangaLoading = isRemoteRoute ? isRemoteMangaLoading : isLocalMangaLoading;

  useEffect(() => {
    if (manga && !isEditingNotes) {
      setLocalUserNotes(manga.userNotes || manga.meta?.user_notes || '');
    }
  }, [manga?.userNotes, manga?.meta?.user_notes, isEditingNotes]);

  // Fetch remote details of library manga from its content provider to get original provider title
  const providerIdForRemote = (manga?.contentProviderId || manga?.sourceId) || '';
  const remoteIdForRemote = manga?.contentRemoteId || '';
  const { data: remoteManga } = useProviderMangaDetails(providerIdForRemote, remoteIdForRemote, {
    enabled: !isRemoteRoute && Boolean(providerIdForRemote && remoteIdForRemote),
  });

  // Fetch Sources for provider info
  const { data: sources = [] } = useSources();

  const activeContentProviderId = isRemoteRoute ? providerIdParam : manga?.contentProviderId || manga?.sourceId || manga?.meta?.content?.provider_id;

  // 4. Fetch Manga Chapters
  const {
    data: localChaptersData,
    isLoading: isLocalChaptersLoading,
    isError: isLocalChaptersError,
  } = useChapterList(targetMangaId, {
    enabled: (!isRemoteRoute || isInLibrary) && Boolean(targetMangaId),
  });

  const {
    data: remoteChaptersData,
    isLoading: isRemoteChaptersLoading,
    isError: isRemoteChaptersError,
  } = useProviderChapterList(providerIdParam, remoteIdParam, {
    enabled: isRemoteRoute && !isInLibrary && Boolean(providerIdParam && remoteIdParam),
  });

  const chaptersData = isRemoteRoute && !isInLibrary ? remoteChaptersData : localChaptersData;
  const isChaptersLoading = isRemoteRoute && !isInLibrary ? isRemoteChaptersLoading : isLocalChaptersLoading;
  const isChaptersError = isRemoteRoute && !isInLibrary ? isRemoteChaptersError : isLocalChaptersError;

  // Mutations
  const addToLibraryMutation = useMutation({
    mutationFn: async () => {
      if (!manga) return null;
      const created = await api.importProviderManga(
        providerIdParam || manga.sourceId || 'mangafox',
        remoteIdParam,
        'plan_to_read'
      );
      return created;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.all });
      showToast('Series added to library', 'success');
    },
    onError: (err: any) => {
      const detail = err.details ? (typeof err.details === 'string' ? err.details : JSON.stringify(err.details, null, 2)) : (err.stack || String(err));
      showToast(`Failed to add to library: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const deleteLibraryMutation = useDeleteLibraryMangaMutation();

  const updateLibraryMangaMutation = useUpdateLibraryMangaMutation();

  const refreshChaptersMutation = useMutation({
    mutationFn: () => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.refreshMangaChapters(targetMangaId);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(targetMangaId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.manga.details(targetMangaId) });
      if (data.added === 0) {
        showToast('Up to date', 'success');
      } else {
        showToast(`Refresh complete: ${data.added} chapter(s)`, 'success');
      }
    },
    onError: (err: any) => {
      const detail = err.details ? (typeof err.details === 'string' ? err.details : JSON.stringify(err.details, null, 2)) : (err.stack || String(err));
      showToast(`Refresh failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const removeChapterMutation = useMutation({
    mutationFn: (chapterId: string) => {
      if (!targetMangaId) throw new Error('No manga ID');
      return api.deleteChapter(targetMangaId, chapterId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.chapters.list(targetMangaId) });
      showToast('Chapter removed from library', 'success');
    },
    onError: (err: any) => {
      const detail = err.details ? (typeof err.details === 'string' ? err.details : JSON.stringify(err.details, null, 2)) : (err.stack || String(err));
      showToast(`Remove failed: ${err.message || 'An error occurred'}`, 'error', detail);
    },
  });

  const handleUserMetadataChange = (updatedFields: Partial<Manga>) => {
    if (!targetMangaId) return;
    updateLibraryMangaMutation.mutate(
      { mangaId: targetMangaId, fields: updatedFields },
      {
        onError: (err: any) => {
          const detail = err.details ? (typeof err.details === 'string' ? err.details : JSON.stringify(err.details, null, 2)) : (err.stack || String(err));
          showToast(`Failed to update metadata: ${err.message || 'An error occurred'}`, 'error', detail);
        },
      }
    );
  };

  const readingCta = useMemo(() => {
    const chapters = chaptersData?.chapters || [];
    if (chapters.length === 0) return null;

    const sorted = [...chapters].sort((a, b) => (a.number ?? 0) - (b.number ?? 0));
    const firstChapter = sorted[0];

    const lastReadChapterId = manga?.lastReadChapterId || manga?.last_read_chapter_id || manga?.meta?.last_read_chapter_id;
    const lastReadChapter = lastReadChapterId ? chapters.find((c) => c.id === lastReadChapterId) : undefined;

    // 1. Check in-progress chapter (!is_read && last_read_page > 1)
    let inProgressChapter: Chapter | undefined;
    if (lastReadChapter) {
      const isRead = Boolean(lastReadChapter.meta?.is_read ?? (lastReadChapter as any).is_read);
      const lastPage = lastReadChapter.meta?.last_read_page ?? (lastReadChapter as any).last_read_page ?? 0;
      if (!isRead && lastPage > 1) {
        inProgressChapter = lastReadChapter;
      }
    }

    if (!inProgressChapter) {
      const inProgressList = sorted.filter((c) => {
        const isRead = Boolean(c.meta?.is_read ?? (c as any).is_read);
        const lastPage = c.meta?.last_read_page ?? (c as any).last_read_page ?? 0;
        return !isRead && lastPage > 1;
      });
      if (inProgressList.length > 0) {
        inProgressChapter = inProgressList[inProgressList.length - 1];
      }
    }

    if (inProgressChapter) {
      const lastPage = inProgressChapter.meta?.last_read_page ?? (inProgressChapter as any).last_read_page ?? 1;
      const chNum = inProgressChapter.number ?? 1;
      return {
        type: 'resume',
        label: `Resume Ch. ${chNum} (p. ${lastPage})`,
        chapterId: inProgressChapter.id,
        page: lastPage,
      };
    }

    // 2. Check read chapters
    const readChapters = sorted.filter((c) => Boolean(c.meta?.is_read ?? (c as any).is_read));

    if (readChapters.length === 0) {
      return {
        type: 'start',
        label: 'Start Reading',
        chapterId: firstChapter.id,
        page: 1,
      };
    }

    if (readChapters.length === sorted.length) {
      const chNum = firstChapter.number ?? 1;
      return {
        type: 'reread',
        label: `Re-read Ch. ${chNum}`,
        chapterId: firstChapter.id,
        page: 1,
      };
    }

    // 3. If Ch X completed: Read Ch. X+1
    let nextChapter: Chapter | undefined;
    if (lastReadChapter && Boolean(lastReadChapter.meta?.is_read ?? (lastReadChapter as any).is_read)) {
      const lastIdx = sorted.findIndex((c) => c.id === lastReadChapter.id);
      if (lastIdx >= 0) {
        nextChapter = sorted.slice(lastIdx + 1).find((c) => !Boolean(c.meta?.is_read ?? (c as any).is_read));
      }
    }

    if (!nextChapter) {
      const lastReadCh = readChapters[readChapters.length - 1];
      const lastReadIdx = sorted.findIndex((c) => c.id === lastReadCh.id);
      if (lastReadIdx >= 0) {
        nextChapter = sorted.slice(lastReadIdx + 1).find((c) => !Boolean(c.meta?.is_read ?? (c as any).is_read));
      }
    }

    if (!nextChapter) {
      nextChapter = sorted.find((c) => !Boolean(c.meta?.is_read ?? (c as any).is_read));
    }

    if (nextChapter) {
      const chNum = nextChapter.number ?? 1;
      return {
        type: 'next',
        label: `Read Ch. ${chNum}`,
        chapterId: nextChapter.id,
        page: 1,
      };
    }

    return {
      type: 'start',
      label: 'Start Reading',
      chapterId: firstChapter.id,
      page: 1,
    };
  }, [chaptersData?.chapters, manga]);

  const handlePrimaryCta = () => {
    if (!readingCta) return;
    if (isRemoteRoute && !isInLibrary) {
      navigate({
        to: '/providers/$providerId/manga/$remoteId/chapter/$chapterId',
        params: {
          providerId: providerIdParam || manga?.sourceId || 'mangafox',
          remoteId: remoteIdParam || manga?.contentRemoteId || manga?.id || '',
          chapterId: readingCta.chapterId,
        },
        search: readingCta.page > 1 ? { page: readingCta.page } : {},
      });
    } else {
      navigate({
        to: '/manga/$mangaId/chapter/$chapterId',
        params: {
          mangaId: targetMangaId || manga?.id || '',
          chapterId: readingCta.chapterId,
        },
        search: readingCta.page > 1 ? { page: readingCta.page } : {},
      });
    }
  };

  const handleConfirmRemoveFromLibrary = () => {
    if (confirm(`Are you sure you want to remove "${manga?.title || 'this series'}" from your library?`)) {
      if (isInLibrary && targetMangaId) {
        deleteLibraryMutation.mutate(targetMangaId, {
          onSuccess: () => {
            showToast('Removed from library', 'info');
            navigate({ to: '/' });
          },
          onError: (err: any) => {
            const detail = err.details ? (typeof err.details === 'string' ? err.details : JSON.stringify(err.details, null, 2)) : (err.stack || String(err));
            showToast(`Failed to remove: ${err.message || 'An error occurred'}`, 'error', detail);
          },
        });
      }
    }
  };

  const handleOrderToggle = () => {
    setOrder((prev) => (prev === 'desc' ? 'asc' : 'desc'));
  };

  const filteredAliases = (manga?.aliases || manga?.meta?.aliases || []).filter(
    (alias) => alias.toLowerCase().trim() !== (manga?.title || '').toLowerCase().trim()
  );

  const authorsList = manga?.authors && manga.authors.length > 0 ? manga.authors : manga?.author ? [manga.author] : manga?.meta?.authors || [];
  const artistsList = manga?.artists && manga.artists.length > 0 ? manga.artists : manga?.artist ? [manga.artist] : manga?.meta?.artists || [];
  const authorsJoined = authorsList.join(', ');
  const artistsJoined = artistsList.join(', ');
  const areAuthorsAndArtistsSame =
    authorsJoined && artistsJoined && authorsJoined.toLowerCase().trim() === artistsJoined.toLowerCase().trim();

  const activeProvider = sources.find((s) => s.id === activeContentProviderId);
  const contentProviderName = activeProvider
    ? `${activeProvider.name}${activeProvider.language || activeProvider.lang ? ` (${(activeProvider.language || activeProvider.lang)!.toUpperCase()})` : ''}`
    : activeContentProviderId || undefined;

  const contentProviderTitle = isRemoteRoute
    ? manga?.title
    : remoteManga?.title;

  const coverSrc = manga?.coverAssetUrl || getProxyImageUrl(manga?.coverUrl || manga?.cover, manga?.url);

  const isFavorite = manga?.userFavorite || manga?.user_favorite || manga?.meta?.user_favorite;
  const userStatus = manga?.userStatus || manga?.meta?.user_status || 'reading';
  const userRating = manga?.userRating || manga?.meta?.user_rating || 0;
  const userNotes = manga?.userNotes || manga?.meta?.user_notes || '';
  const hasZeroChapters = isRemoteRoute && !isInLibrary
    ? !isRemoteChaptersLoading && Boolean(remoteChaptersData) && remoteChaptersData?.chapters?.length === 0
    : !isLocalChaptersLoading && Boolean(localChaptersData) && localChaptersData?.chapters?.length === 0;
  const isUnavailable = (manga?.availability === 'unavailable') || (remoteManga?.availability === 'unavailable') || (manga?.meta?.availability === 'unavailable') || (isRemoteRoute && hasZeroChapters);

  return (
    <div className="flex flex-col gap-6">
      {/* Top Action Bar */}
      <div className="flex items-center justify-between gap-4">
        {isRemoteRoute ? (
          <Link
            to="/providers/$providerId"
            params={{ providerId: providerIdParam || 'mangafox' }}
            className="inline-flex items-center gap-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="size-4" aria-hidden />
            Back to Explore
          </Link>
        ) : (
          <Link
            to="/"
            className="inline-flex items-center gap-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="size-4" aria-hidden />
            Back to Library
          </Link>
        )}

        <div className="flex flex-wrap items-center gap-2">
          {isRemoteRoute ? (
            isInLibrary ? (
              <Link
                to="/manga/$mangaId"
                params={{ mangaId: targetMangaId }}
                className="inline-flex h-9 items-center gap-2 rounded-md border border-emerald-500/40 bg-emerald-500/10 px-3.5 text-xs font-semibold text-emerald-500 hover:bg-emerald-500/20 transition-colors"
                title="Series is saved in your local library. Click to view library entry."
              >
                <BookOpen className="size-4" aria-hidden />
                View in Library
              </Link>
            ) : (
              <Button
                variant="default"
                onClick={() => addToLibraryMutation.mutate()}
                disabled={addToLibraryMutation.isPending || !manga || hasZeroChapters || isUnavailable}
                className="gap-2 cursor-pointer"
              >
                <Plus className="size-4" aria-hidden />
                {addToLibraryMutation.isPending ? 'Adding...' : 'Add to Library'}
              </Button>
            )
          ) : (
            manga && !isUnavailable && chaptersData?.chapters && chaptersData.chapters.length > 0 && readingCta && (
              <Button
                variant="default"
                onClick={handlePrimaryCta}
                className="font-semibold cursor-pointer gap-2"
              >
                <Play className="size-4 fill-current" aria-hidden />
                {readingCta.label}
              </Button>
            )
          )}
        </div>
      </div>

      {/* Hero Card */}
      <Card className="grid grid-cols-1 gap-6 p-6 md:grid-cols-[240px_1fr]">
        <div className="flex flex-col gap-4">
          <div className="overflow-hidden rounded-lg bg-muted shadow-md">
            {isMangaLoading ? (
              <Skeleton className="aspect-[2/3] w-full" />
            ) : (
              <img
                src={coverSrc}
                alt={manga?.title || 'Manga Cover'}
                className="aspect-[2/3] w-full object-cover"
                onError={(e) => {
                  const fallback = manga?.coverUrl || manga?.cover;
                  if (fallback && e.currentTarget.src !== fallback) {
                    e.currentTarget.src = fallback;
                  }
                }}
              />
            )}
          </div>

          {/* User Metadata / Library Actions (only if local library entry) */}
          {!isRemoteRoute && isInLibrary && manga && (
            <div className={cn(
              "flex flex-col gap-3 p-3.5 rounded-lg border border-border bg-card shadow-xs transition-opacity duration-200",
              updateLibraryMangaMutation.isPending && "opacity-60 pointer-events-none"
            )}>
              {/* Reading Status Dropdown */}
              <div className="flex flex-col gap-1">
                <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Reading Status</span>
                <Select
                  value={userStatus}
                  onValueChange={(val) => handleUserMetadataChange({ user_status: val || undefined })}
                  disabled={updateLibraryMangaMutation.isPending}
                >
                  <SelectTrigger className="h-9 w-full text-xs bg-background border-border">
                    <SelectValue placeholder="Select Status">
                      {formatStatus(userStatus)}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {STATUS_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value} className="text-xs">
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Rating and Favorite row */}
              <div className="flex items-center justify-between border-t border-border/40 pt-3 mt-0.5">
                {/* 5-star rating with score indicator */}
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-1.5">
                    <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Rating</span>
                    <span className="text-[10px] font-semibold text-foreground/80">
                      {(hoverRating !== null ? hoverRating : userRating) > 0
                        ? `${hoverRating !== null ? hoverRating : userRating}/10`
                        : 'Unrated'}
                    </span>
                    {userRating > 0 && (
                      <button
                        type="button"
                        onClick={() => handleUserMetadataChange({ user_rating: 0 })}
                        disabled={updateLibraryMangaMutation.isPending}
                        className="text-[10px] text-muted-foreground hover:text-destructive cursor-pointer"
                        title="Clear Rating"
                      >
                        ×
                      </button>
                    )}
                  </div>
                  <div
                    className="flex items-center gap-0.5"
                    onMouseLeave={() => setHoverRating(null)}
                  >
                    {Array.from({ length: 5 }).map((_, idx) => {
                      const starValue = (idx + 1) * 2;
                      const activeValue = hoverRating !== null ? hoverRating : userRating;
                      const isFilled = activeValue >= starValue;
                      return (
                        <button
                          key={idx}
                          type="button"
                          disabled={updateLibraryMangaMutation.isPending}
                          onMouseEnter={() => setHoverRating(starValue)}
                          onClick={() => {
                            const newRating = userRating === starValue ? 0 : starValue;
                            handleUserMetadataChange({ user_rating: newRating });
                          }}
                          className="text-amber-400 hover:scale-110 active:scale-95 transition-transform focus:outline-hidden disabled:opacity-50 cursor-pointer"
                          title={`Rate ${starValue}/10 (${idx + 1} Stars)`}
                        >
                          <Star
                            className={cn(
                              "size-4",
                              isFilled ? "fill-amber-400 text-amber-400" : "text-muted-foreground/30"
                            )}
                          />
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Love / Favorite button */}
                <div className="flex flex-col gap-1 items-end">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Favorite</span>
                  <button
                    type="button"
                    disabled={updateLibraryMangaMutation.isPending}
                    onClick={() => handleUserMetadataChange({ user_favorite: !isFavorite })}
                    className={cn(
                      "flex items-center justify-center size-8 rounded-full border transition-all active:scale-95 hover:bg-muted/50 cursor-pointer disabled:opacity-50",
                      isFavorite
                        ? "bg-rose-500/10 border-rose-500/30 text-rose-500 hover:bg-rose-500/20"
                        : "bg-background border-border text-muted-foreground hover:text-foreground"
                    )}
                    title={isFavorite ? "Remove from Favorites" : "Add to Favorites"}
                    aria-label={isFavorite ? "Remove from Favorites" : "Add to Favorites"}
                  >
                    <Heart className="size-4" fill={isFavorite ? "currentColor" : "none"} />
                  </button>
                </div>
              </div>

              {/* Personal Notes */}
              <div className="flex flex-col gap-1.5 border-t border-border/40 pt-3 mt-0.5">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Personal Notes</span>
                  {!isEditingNotes && (
                    <button
                      type="button"
                      onClick={() => setIsEditingNotes(true)}
                      className="text-[10px] font-semibold text-primary hover:underline cursor-pointer"
                    >
                      {userNotes ? 'Edit' : '+ Add Note'}
                    </button>
                  )}
                </div>

                {isEditingNotes ? (
                  <div className="flex flex-col gap-2">
                    <textarea
                      value={localUserNotes}
                      onChange={(e) => setLocalUserNotes(e.target.value)}
                      disabled={updateLibraryMangaMutation.isPending}
                      placeholder="Add private personal notes..."
                      autoFocus
                      className="w-full min-h-16 h-20 max-h-32 rounded-md border border-border bg-background px-3 py-2 text-xs placeholder:text-muted-foreground focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-ring disabled:opacity-50"
                    />
                    <div className="flex items-center justify-end gap-1.5">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setLocalUserNotes(userNotes);
                          setIsEditingNotes(false);
                        }}
                        disabled={updateLibraryMangaMutation.isPending}
                        className="h-7 text-[11px] px-2.5 cursor-pointer"
                      >
                        Cancel
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        onClick={() => {
                          handleUserMetadataChange({ user_notes: localUserNotes || undefined });
                          setIsEditingNotes(false);
                        }}
                        disabled={updateLibraryMangaMutation.isPending}
                        className="h-7 text-[11px] px-2.5 bg-primary hover:bg-primary/90 text-primary-foreground cursor-pointer"
                      >
                        {updateLibraryMangaMutation.isPending ? 'Saving...' : 'Save'}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="text-xs text-foreground bg-muted/40 border border-border/20 rounded-md p-2.5 min-h-12 break-words whitespace-pre-wrap">
                    {userNotes ? userNotes : (
                      <span className="text-muted-foreground italic text-[11px]">No notes added yet.</span>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        <div className="flex flex-col gap-4">
          <div className="flex items-start justify-between gap-4">
            <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl">
              {isMangaLoading ? 'Loading Manga...' : manga?.title || 'Untitled Manga'}
            </h1>

            {isInLibrary && manga && (
              <DropdownMenu>
                <DropdownMenuTrigger
                  className="inline-flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-card text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer"
                  aria-label="Metadata options"
                  title="Metadata options"
                >
                  <MoreVertical className="size-4" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                  <DropdownMenuItem
                    onClick={() => setIsEditMetadataOpen(true)}
                    className="text-xs cursor-pointer gap-2"
                  >
                    <Edit3 className="size-4" />
                    Edit Metadata
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={handleConfirmRemoveFromLibrary}
                    className="text-destructive focus:text-destructive focus:bg-destructive/10 text-xs cursor-pointer gap-2"
                  >
                    <Trash2 className="size-4" />
                    Remove from Library
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>

          {filteredAliases.length > 0 && (
            <p className="text-xs text-muted-foreground">
              <span className="font-semibold text-foreground">Aliases:</span> {filteredAliases.join(', ')}
            </p>
          )}

          {/* Essential Info: merged Author/Artist if same */}
          <div className="flex flex-col gap-1 text-sm border-t border-b border-border/40 py-3 mt-1">
            {areAuthorsAndArtistsSame ? (
              <div>
                <span className="text-muted-foreground font-medium">Author / Artist: </span>
                <span className="text-foreground font-semibold">{authorsJoined}</span>
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-1">
                {authorsJoined && (
                  <div>
                    <span className="text-muted-foreground font-medium">Author: </span>
                    <span className="text-foreground font-semibold">{authorsJoined}</span>
                  </div>
                )}
                {artistsJoined && (
                  <div>
                    <span className="text-muted-foreground font-medium">Artist: </span>
                    <span className="text-foreground font-semibold">{artistsJoined}</span>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Collapsible Detailed Metadata */}
          <div className="border-b border-border/40 pb-3 mt-1">
            <button
              type="button"
              onClick={() => setShowDetailedMetadata(!showDetailedMetadata)}
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground font-semibold transition-colors focus:outline-none cursor-pointer"
            >
              {showDetailedMetadata ? (
                <>
                  <ChevronUp className="size-3.5" />
                  Hide Details
                </>
              ) : (
                <>
                  <ChevronDown className="size-3.5" />
                  Show Detailed Metadata
                </>
              )}
            </button>

            {showDetailedMetadata && (
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mt-3 text-xs bg-muted/40 p-3 rounded-lg border border-border/30">
                <div>
                  <span className="text-muted-foreground block text-[10px] uppercase font-bold">Reading Mode</span>
                  <span className="font-medium text-foreground uppercase">
                    {READING_MODE_LABELS[
                      (
                        manga?.content?.reading_mode ||
                        manga?.meta?.content?.reading_mode ||
                        manga?.readingMode ||
                        manga?.reading_mode ||
                        manga?.readingDirection ||
                        manga?.meta?.reading_direction ||
                        'rtl'
                      ).toLowerCase()
                    ] ||
                      manga?.content?.reading_mode ||
                      manga?.meta?.content?.reading_mode ||
                      manga?.readingMode ||
                      manga?.reading_mode ||
                      manga?.readingDirection ||
                      manga?.meta?.reading_direction ||
                      'rtl'}
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground block text-[10px] uppercase font-bold">Content Rating</span>
                  <span className="font-medium text-foreground capitalize">{manga?.contentRating || manga?.meta?.content_rating || 'safe'}</span>
                </div>
                {manga?.publisher && (
                  <div>
                    <span className="text-muted-foreground block text-[10px] uppercase font-bold">Publisher</span>
                    <span className="font-medium text-foreground">{manga.publisher}</span>
                  </div>
                )}
                {manga?.releaseYear && (
                  <div>
                    <span className="text-muted-foreground block text-[10px] uppercase font-bold">Release Year</span>
                    <span className="font-medium text-foreground">{manga.releaseYear}</span>
                  </div>
                )}
                {manga?.country && (
                  <div>
                    <span className="text-muted-foreground block text-[10px] uppercase font-bold">Country</span>
                    <span className="font-medium text-foreground uppercase">{manga.country}</span>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Genre / Tag Pills */}
          {((manga?.tags && manga.tags.length > 0) || (manga?.genres && manga.genres.length > 0)) && (
            <div className="flex flex-wrap gap-1.5">
              {(manga?.tags || manga?.genres || manga?.meta?.tags || []).map((genre) => (
                <GenrePill key={genre} genre={genre} />
              ))}
            </div>
          )}

          {/* Description */}
          <div className="text-sm leading-relaxed text-muted-foreground whitespace-pre-wrap">
            {manga?.description || manga?.meta?.description || 'No description available for this series.'}
          </div>
        </div>
      </Card>

      {/* Chapters Section using ChapterList */}
      {manga && (
        <ChapterList
          chapters={chaptersData?.chapters || []}
          mangaId={manga.id}
          providerId={isRemoteRoute ? providerIdParam : undefined}
          remoteId={isRemoteRoute ? remoteIdParam : undefined}
          sortBy={sortBy}
          order={order}
          onSortByChange={setSortBy}
          onOrderToggle={handleOrderToggle}
          isLoading={isChaptersLoading}
          isError={isChaptersError}
          contentProviderName={contentProviderName}
          contentProviderTitle={contentProviderTitle}
          providerName={activeProvider?.name || activeContentProviderId}
          isUnavailable={isUnavailable}
          isInLibrary={isInLibrary}
          onRefreshChapters={isInLibrary ? () => refreshChaptersMutation.mutate() : undefined}
          isRefreshing={refreshChaptersMutation.isPending}
          onRemoveChapter={isInLibrary ? (chId) => removeChapterMutation.mutate(chId) : undefined}
        />
      )}

      {/* Edit Metadata Dialog */}
      {manga && (
        <EditMetadataDialog
          manga={manga}
          open={isEditMetadataOpen}
          onOpenChange={setIsEditMetadataOpen}
        />
      )}
    </div>
  );
};
