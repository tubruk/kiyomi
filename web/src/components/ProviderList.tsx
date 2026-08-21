import React from 'react';
import { Trash2, Info, SwitchCamera } from 'lucide-react';
import { ProviderRef, Source } from '../types/api';
import { CapabilityBadge } from './CapabilityBadge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';
import { cn } from '../lib/utils';

interface ProviderListProps {
  providers: ProviderRef[];
  contentProviderId?: string;
  contentProviderMangaId?: string;
  sources: Source[];
  onRemove?: (provider: ProviderRef) => void;
  onSwitchTo?: (provider: ProviderRef) => void;
  isRemoving?: boolean;
  canRemoveProvider?: (provider: ProviderRef) => boolean;
  isContentUnavailable?: boolean;
}

export const ProviderList: React.FC<ProviderListProps> = ({
  providers,
  contentProviderId,
  contentProviderMangaId,
  sources,
  onRemove,
  onSwitchTo,
  isRemoving = false,
  canRemoveProvider = () => true,
  isContentUnavailable = false,
}) => {
  if (providers.length === 0) {
    return (
      <div className="flex flex-col gap-2 rounded-lg border border-dashed border-border bg-muted/30 p-4 text-center">
        <Info className="mx-auto size-5 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">
          No providers linked. Add from provider to enable sync + downloads.
        </p>
      </div>
    );
  }

  const getSourceName = (providerId: string) => {
    const source = sources.find((s) => s.id === providerId);
    return source?.name || providerId;
  };

  const getSourceIcon = (providerId: string) => {
    const source = sources.find((s) => s.id === providerId);
    return source?.icon || null;
  };

  const isActiveContentProvider = (provider: ProviderRef) =>
    provider.provider_id === contentProviderId &&
    provider.provider_manga_id === contentProviderMangaId;

  const hasContentCapability = (providerId: string) => {
    const source = sources.find((s) => s.id === providerId);
    return source?.capabilities?.includes('content') ?? false;
  };

  const hasMetadataCapability = (providerId: string) => {
    const source = sources.find((s) => s.id === providerId);
    return source?.capabilities?.includes('metadata') ?? false;
  };

  return (
    <ul className="rounded-lg border border-border bg-card divide-y divide-border">
      {providers.map((provider) => {
        const active = isActiveContentProvider(provider);
        const canRemove = canRemoveProvider(provider);
        const providerName = getSourceName(provider.provider_id);
        const icon = getSourceIcon(provider.provider_id);

        return (
          <li
            key={`${provider.provider_id}-${provider.provider_manga_id}`}
            className="flex items-center gap-3 px-3 py-2 text-xs"
          >
            {icon ? (
              <img src={icon} alt="" className="size-4 shrink-0 rounded-sm object-contain" />
            ) : (
              <span className="size-4 shrink-0" />
            )}

            <span className="font-semibold shrink-0">{providerName}</span>

            <div className="flex-1 min-w-0 flex items-center gap-2">
              {provider.manga_title && (
                <span className="text-xs text-muted-foreground truncate">
                  {provider.manga_title}
                </span>
              )}
              {hasContentCapability(provider.provider_id) && (
                <CapabilityBadge
                  capability="content"
                  active={active}
                  unavailable={active && isContentUnavailable}
                />
              )}
              {hasMetadataCapability(provider.provider_id) && (
                <CapabilityBadge capability="metadata" />
              )}
            </div>

            <DropdownMenu>
              <DropdownMenuTrigger className="flex size-6 items-center justify-center rounded text-muted-foreground hover:text-foreground focus:outline-none cursor-pointer">
                <span className="text-xs">•••</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-44">
                {!active && hasContentCapability(provider.provider_id) && (
                  <DropdownMenuItem
                    onClick={() => onSwitchTo?.(provider)}
                    className="text-xs cursor-pointer gap-2"
                  >
                    <SwitchCamera className="size-3" />
                    Switch content to this
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  onClick={() => onRemove?.(provider)}
                  disabled={!canRemove || isRemoving}
                  className={cn(
                    'text-xs cursor-pointer gap-2',
                    canRemove ? 'text-destructive focus:text-destructive' : 'text-muted-foreground'
                  )}
                >
                  <Trash2 className="size-3" />
                  Remove
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </li>
        );
      })}
    </ul>
  );
};
