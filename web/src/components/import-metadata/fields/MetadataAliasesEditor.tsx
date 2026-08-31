import React from 'react';
import { Layers, X, Plus } from 'lucide-react';
import { Badge } from '../../ui/badge';

interface MetadataAliasesEditorProps {
  selectedAliases: string[];
  unselectedAliases: string[];
  aliasInput: string;
  onAliasInputChange: (val: string) => void;
  onAliasInputKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  onAddAlias: (alias: string) => void;
  onRemoveAlias: (alias: string) => void;
}

export const MetadataAliasesEditor: React.FC<MetadataAliasesEditorProps> = ({
  selectedAliases,
  unselectedAliases,
  aliasInput,
  onAliasInputChange,
  onAliasInputKeyDown,
  onAddAlias,
  onRemoveAlias,
}) => {
  return (
    <div className="space-y-2.5">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold flex items-center gap-1.5">
          <Layers className="size-4 text-muted-foreground" />
          Aliases ({selectedAliases.length})
        </span>
      </div>

      {/* Active Aliases Typeable Pill Box */}
      <div className="flex flex-wrap items-center gap-2 p-3 rounded-xl border border-border bg-card focus-within:ring-1 focus-within:ring-ring focus-within:border-ring min-h-11 cursor-text">
        {selectedAliases.map((alias) => (
          <Badge
            key={alias}
            variant="secondary"
            className="text-xs gap-1.5 py-1 px-2.5 font-normal"
          >
            <span className="max-w-xs truncate">{alias}</span>
            <button
              type="button"
              onClick={() => onRemoveAlias(alias)}
              className="rounded-full hover:bg-muted-foreground/20 p-0.5 cursor-pointer ml-0.5"
              title={`Remove "${alias}"`}
            >
              <X className="size-3" />
            </button>
          </Badge>
        ))}
        <input
          type="text"
          value={aliasInput}
          onChange={(e) => onAliasInputChange(e.target.value)}
          onKeyDown={onAliasInputKeyDown}
          placeholder={selectedAliases.length === 0 ? 'Type an alias and press Enter...' : 'Add alias...'}
          aria-label="Add alias"
          className="flex-1 min-w-[140px] bg-transparent text-xs outline-none placeholder:text-muted-foreground"
        />
      </div>

      {/* Click-to-re-add unselected aliases */}
      {unselectedAliases.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <span className="text-xs text-muted-foreground">Available to add:</span>
          {unselectedAliases.map((alias) => (
            <button
              key={alias}
              type="button"
              onClick={() => onAddAlias(alias)}
              className="text-xs px-2.5 py-1 rounded-full border border-dashed border-muted-foreground/40 hover:border-foreground text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-1"
            >
              <Plus className="size-3" />
              <span className="max-w-xs truncate">{alias}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};
