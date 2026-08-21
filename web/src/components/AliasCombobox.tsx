import React, { useState, useRef, useEffect } from 'react';
import { Search, ChevronDown } from 'lucide-react';
import { cn } from '../lib/utils';

interface AliasComboboxProps {
  /** Default selected option (main title). */
  defaultValue: string;
  /** Aliases shown as suggestions in the dropdown. */
  suggestions: string[];
  /** Controlled value (search query). */
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  autoFocus?: boolean;
}

/**
 * Combobox input: free text editing, with a dropdown of preset suggestions
 * (main title + aliases). User can always type freely; suggestions are
 * presets only.
 */
export const AliasCombobox: React.FC<AliasComboboxProps> = ({
  defaultValue,
  suggestions,
  value,
  onChange,
  placeholder = 'Search...',
  className,
  autoFocus = false,
}) => {
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Close on outside click.
  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (!wrapperRef.current) return;
      if (!wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, []);

  const preset = [defaultValue, ...suggestions.filter((a) => a !== defaultValue)];

  return (
    <div ref={wrapperRef} className={cn('relative', className)}>
      <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground pointer-events-none" />
      <input
        ref={inputRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={() => setOpen(true)}
        placeholder={placeholder}
        autoFocus={autoFocus}
        className="flex h-9 w-full rounded-md border border-input bg-transparent pl-8 pr-8 py-1 text-xs shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      />
      <button
        type="button"
        onClick={() => {
          inputRef.current?.focus();
          setOpen((v) => !v);
        }}
        className="absolute right-1 top-1/2 -translate-y-1/2 flex size-7 items-center justify-center rounded text-muted-foreground hover:text-foreground focus:outline-none cursor-pointer"
        tabIndex={-1}
      >
        <ChevronDown className={cn('size-3.5 transition-transform', open && 'rotate-180')} />
      </button>

      {open && preset.length > 0 && (
        <ul className="absolute z-50 mt-1 w-full max-h-60 overflow-y-auto rounded-md border border-border bg-popover text-popover-foreground shadow-md">
          {preset.map((label) => (
            <li key={label}>
              <button
                type="button"
                onMouseDown={(e) => {
                  e.preventDefault();
                  onChange(label);
                  setOpen(false);
                }}
                className="flex w-full items-center gap-2 px-2 py-1.5 text-xs text-left hover:bg-muted/50 cursor-pointer"
              >
                <span className="flex-1 truncate">{label}</span>
                {label === defaultValue && (
                  <span className="text-[10px] text-muted-foreground">main title</span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};
