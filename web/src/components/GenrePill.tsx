import React from 'react';
import { Badge } from './ui/badge';

interface GenrePillProps {
  genre: string;
}

export const GenrePill: React.FC<GenrePillProps> = ({ genre }) => {
  return (
    <Badge variant="secondary" className="text-xs font-medium">
      {genre}
    </Badge>
  );
};
