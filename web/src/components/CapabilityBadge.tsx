import React from 'react';
import { Check, AlertTriangle } from 'lucide-react';
import { Badge } from './ui/badge';

type Capability = 'content' | 'metadata' | 'tracking';

interface CapabilityBadgeProps {
  capability: Capability;
  className?: string;
  active?: boolean;
  unavailable?: boolean;
}

const CAPABILITY_LABEL: Record<Capability, string> = {
  content: 'Content',
  metadata: 'Metadata',
  tracking: 'Tracking',
};

export const CapabilityBadge: React.FC<CapabilityBadgeProps> = ({ capability, className, active, unavailable }) => {
  const label = CAPABILITY_LABEL[capability];
  if (!label) return null;

  const isActive = active && !unavailable;
  const isUnavailable = unavailable && !active;

  const stateClass = isActive
    ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
    : isUnavailable
    ? 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-400'
    : '';

  return (
    <Badge
      variant="outline"
      className={`h-4 gap-1 px-1.5 text-[10px] ${stateClass} ${className || ''}`.trim()}
      title={isUnavailable ? 'Content not available' : undefined}
    >
      {isActive && <Check className="size-2.5 text-emerald-500" />}
      {isUnavailable && <AlertTriangle className="size-2.5" />}
      {label}
    </Badge>
  );
};
