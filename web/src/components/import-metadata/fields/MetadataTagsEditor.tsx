import React from 'react';
import { Compass, X, Plus } from 'lucide-react';
import { Button } from '../../ui/button';
import { Badge } from '../../ui/badge';
import { mergeStringArrays } from '../../../utils/metadataCompare';

interface MetadataTagsEditorProps {
  currentTags: string[];
  incomingTags: string[];
  selectedTags: string[];
  unselectedTags: string[];
  tagInput: string;
  onTagInputChange: (val: string) => void;
  onTagInputKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  onAddTag: (tag: string) => void;
  onRemoveTag: (tag: string) => void;
  onSetSelectedTags: (tags: string[]) => void;
}

export const MetadataTagsEditor: React.FC<MetadataTagsEditorProps> = ({
  currentTags,
  incomingTags,
  selectedTags,
  unselectedTags,
  tagInput,
  onTagInputChange,
  onTagInputKeyDown,
  onAddTag,
  onRemoveTag,
  onSetSelectedTags,
}) => {
  return (
    <div className="space-y-2.5">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold flex items-center gap-1.5">
          <Compass className="size-4 text-muted-foreground" />
          Tags & Genres ({selectedTags.length})
        </span>
        <div className="flex items-center gap-1.5">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onSetSelectedTags(mergeStringArrays(currentTags, incomingTags))}
            className="h-7 text-xs px-2.5 cursor-pointer"
          >
            Merge
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onSetSelectedTags([...currentTags])}
            className="h-7 text-xs px-2.5 cursor-pointer"
          >
            Current
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onSetSelectedTags([...incomingTags])}
            className="h-7 text-xs px-2.5 cursor-pointer"
          >
            Incoming
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onSetSelectedTags([])}
            className="h-7 text-xs px-2.5 text-muted-foreground hover:text-destructive cursor-pointer"
          >
            Clear
          </Button>
        </div>
      </div>

      {/* Active Tags Typeable Pill Box */}
      <div className="flex flex-wrap items-center gap-2 p-3 rounded-xl border border-border bg-card focus-within:ring-1 focus-within:ring-ring focus-within:border-ring min-h-11 cursor-text">
        {selectedTags.map((tag) => (
          <Badge
            key={tag}
            variant="secondary"
            className="text-xs gap-1.5 py-1 px-2.5 font-normal"
          >
            <span>{tag}</span>
            <button
              type="button"
              onClick={() => onRemoveTag(tag)}
              className="rounded-full hover:bg-muted-foreground/20 p-0.5 cursor-pointer ml-0.5"
              title={`Remove tag "${tag}"`}
            >
              <X className="size-3" />
            </button>
          </Badge>
        ))}
        <input
          type="text"
          value={tagInput}
          onChange={(e) => onTagInputChange(e.target.value)}
          onKeyDown={onTagInputKeyDown}
          placeholder={selectedTags.length === 0 ? 'Type a tag and press Enter...' : 'Add tag...'}
          aria-label="Add tag"
          className="flex-1 min-w-[120px] bg-transparent text-xs outline-none placeholder:text-muted-foreground"
        />
      </div>

      {/* Click-to-re-add unselected tags */}
      {unselectedTags.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <span className="text-xs text-muted-foreground">Unselected tags:</span>
          {unselectedTags.map((tag) => (
            <button
              key={tag}
              type="button"
              onClick={() => onAddTag(tag)}
              className="text-xs px-2.5 py-1 rounded-full border border-dashed border-muted-foreground/40 hover:border-foreground text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-1"
            >
              <Plus className="size-3" />
              <span>{tag}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};
