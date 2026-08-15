export interface ExternalLink {
  provider: string;
  label: string;
  url: string;
}

export interface ContentSource {
  provider_id: string;
  manga_id?: string;
  chapter_ref?: string;
  last_synced_at?: string;
  reading_mode?: string;
}

export interface Source {
  id: string;
  name: string;
  baseUrl?: string;
  lang?: string;
  language?: string;
  icon?: string;
  supportsLatest?: boolean;
  capabilities?: string[];
}

export type ContentAvailability = 'available' | 'unavailable' | 'unknown';

export type UserStatus = 'unread' | 'reading' | 'completed' | 'on_hold' | 'dropped' | 'plan_to_read';

export interface MangaMeta {
  title?: string;
  aliases?: string[];
  description?: string;
  authors?: string[];
  artists?: string[];
  tags?: string[];
  collections?: string[];
  reading_direction?: string;
  reading_mode?: string;
  content_rating?: string;
  publisher?: string;
  release_year?: number;
  content?: ContentSource;
  user_status?: UserStatus | string;
  user_rating?: number;
  user_favorite?: boolean;
  user_notes?: string;
  availability?: ContentAvailability;
  last_read_chapter_id?: string;
  last_read_at?: string;
  added_at?: string;
  updated_at?: string;
}

export interface Manga {
  id: string;
  title: string;
  cover?: string;
  coverUrl?: string;
  coverAssetUrl?: string;
  banner?: string;
  url?: string;
  description?: string;
  author?: string;
  authors?: string[];
  artist?: string;
  artists?: string[];
  tags?: string[];
  genres?: string[];
  aliases?: string[];
  shelves?: string[];
  status?: string;
  userStatus?: UserStatus | string;
  user_status?: UserStatus | string;
  userFavorite?: boolean;
  user_favorite?: boolean;
  userRating?: number;
  user_rating?: number;
  userNotes?: string;
  user_notes?: string;
  lastReadChapterId?: string;
  last_read_chapter_id?: string;
  lastReadAt?: string;
  last_read_at?: string;
  readingDirection?: string;
  reading_direction?: string;
  readingMode?: string;
  reading_mode?: string;
  contentRating?: string;
  publisher?: string;
  releaseYear?: number;
  country?: string;
  externalLinks?: ExternalLink[];
  sourceId?: string;
  contentProviderId?: string;
  contentRemoteId?: string;
  content?: ContentSource;
  availability?: ContentAvailability;
  meta?: MangaMeta;
}

export interface ChapterMeta {
  title?: string;
  number?: number;
  volume?: number;
  language?: string;
  upload_date?: string;
  source_order?: number;
  content?: ContentSource;
  page_count?: number;
  page_format?: string;
  downloaded_at?: string;
  is_read?: boolean;
  last_read_page?: number;
  last_read_at?: string;
}

export interface Chapter {
  id: string;
  manga_id?: string;
  mangaId?: string;
  name: string;
  title?: string;
  number: number;
  volume?: number;
  uploadDate?: string;
  uploadedAt?: string;
  meta?: ChapterMeta;
  sourceOrder?: number;
  is_read?: boolean;
  isRead?: boolean;
  last_read_page?: number;
  lastReadPage?: number;
  last_read_at?: string;
  lastReadAt?: string;
}

export interface Page {
  index: number;
  url: string;
  assetUrl?: string;
}

export interface ExploreResponse {
  mangas: Manga[];
  hasNext?: boolean;
  page?: number;
}

export interface ChapterListResponse {
  chapters: Chapter[];
}

export interface PageListResponse {
  pages: Page[];
}
