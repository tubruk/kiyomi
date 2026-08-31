import React from 'react';
import { cn } from '../../../lib/utils';
import { Choice } from '../types';

interface MetadataFieldRowProps {
  label: string;
  icon?: React.ReactNode;
  currentValue: string | number;
  incomingValue: string | number;
  selectedValue: Choice;
  onSelect: (choice: Choice) => void;
  textTransform?: 'uppercase' | 'capitalize' | 'none';
  formatDisplay?: (val: string | number) => string;
}

export const MetadataFieldRow: React.FC<MetadataFieldRowProps> = ({
  label,
  icon,
  currentValue,
  incomingValue,
  selectedValue,
  onSelect,
  textTransform = 'none',
  formatDisplay,
}) => {
  const format = (val: string | number): string => {
    if (formatDisplay) return formatDisplay(val);
    if (!val && val !== 0) return '—';
    if (typeof val === 'number') {
      return val > 0 ? String(val) : '—';
    }
    return val.trim() || '—';
  };

  const currentDisplay = format(currentValue);
  const incomingDisplay = format(incomingValue);

  return (
    <div className="grid grid-cols-3 py-3 px-4 items-center text-xs gap-3">
      <span className="font-medium text-muted-foreground flex items-center gap-2">
        {icon} {label}
      </span>
      <button
        type="button"
        onClick={() => onSelect('current')}
        className={cn(
          'rounded-md p-2 text-left truncate transition-colors cursor-pointer',
          textTransform === 'uppercase' && 'uppercase',
          textTransform === 'capitalize' && 'capitalize',
          selectedValue === 'current'
            ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
            : 'hover:bg-muted text-muted-foreground'
        )}
      >
        {currentDisplay}
      </button>
      <button
        type="button"
        onClick={() => onSelect('incoming')}
        className={cn(
          'rounded-md p-2 text-left truncate transition-colors cursor-pointer',
          textTransform === 'uppercase' && 'uppercase',
          textTransform === 'capitalize' && 'capitalize',
          selectedValue === 'incoming'
            ? 'bg-primary/10 text-primary font-medium ring-1 ring-primary/40'
            : 'hover:bg-muted text-muted-foreground'
        )}
      >
        {incomingDisplay}
      </button>
    </div>
  );
};
