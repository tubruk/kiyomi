import { Manga, Source, ExternalLink } from '../../types/api';

export type DialogStep = 'search' | 'compare';
export type SearchMode = 'keyword' | 'direct';
export type Choice = 'current' | 'incoming';
export type MultiChoice = 'current' | 'incoming' | 'merged';

export interface MetadataValues {
  title: string;
  coverUrl: string;
  description: string;
  authors: string[];
  artists: string[];
  tags: string[];
  aliases: string[];
  publisher: string;
  publishers: string[];
  releaseYear: number;
  startDate: string;
  endDate: string;
  contentRating: string;
  country: string;
  readingMode: string;
  externalLinks: ExternalLink[];
}

export interface MetadataDiffs {
  cover: boolean;
  title: boolean;
  description: boolean;
  authors: boolean;
  artists: boolean;
  tags: boolean;
  aliases: boolean;
  publisher: boolean;
  publishers: boolean;
  releaseYear: boolean;
  startDate: boolean;
  endDate: boolean;
  contentRating: boolean;
  country: boolean;
  readingMode: boolean;
  externalLinks: boolean;
}

export interface ImportMetadataDialogProps {
  manga: Manga;
  sources: Source[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialProviderId?: string;
  initialRemoteId?: string;
  onSuccess?: (manga: Manga) => void;
}
