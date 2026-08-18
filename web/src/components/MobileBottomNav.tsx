import React from 'react';
import { Link, useLocation } from '@tanstack/react-router';
import { BookOpen, Compass, Settings } from 'lucide-react';

export const MobileBottomNav: React.FC = () => {
  const location = useLocation();

  const isReaderPage = location.pathname.startsWith('/reader/') || location.pathname.includes('/chapter/');
  if (isReaderPage) return null;

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 flex h-16 items-center justify-around border-t border-border bg-background/95 backdrop-blur-md md:hidden">
      <Link
        to="/"
        activeOptions={{ exact: true }}
        activeProps={{
          className: 'flex flex-col items-center justify-center gap-1 text-xs font-medium transition-colors text-primary',
        }}
        inactiveProps={{
          className: 'flex flex-col items-center justify-center gap-1 text-xs font-medium transition-colors text-muted-foreground hover:text-foreground',
        }}
      >
        <BookOpen className="size-5" aria-hidden />
        <span>Library</span>
      </Link>

      <Link
        to="/explore"
        activeOptions={{ exact: false }}
        activeProps={{
          className: 'flex flex-col items-center justify-center gap-1 text-xs font-medium transition-colors text-primary',
        }}
        inactiveProps={{
          className: 'flex flex-col items-center justify-center gap-1 text-xs font-medium transition-colors text-muted-foreground hover:text-foreground',
        }}
      >
        <Compass className="size-5" aria-hidden />
        <span>Explore</span>
      </Link>

      <Link
        to="/settings/plugins"
        activeOptions={{ exact: false }}
        activeProps={{
          className: 'flex flex-col items-center justify-center gap-1 text-xs font-medium transition-colors text-primary',
        }}
        inactiveProps={{
          className: 'flex flex-col items-center justify-center gap-1 text-xs font-medium transition-colors text-muted-foreground hover:text-foreground',
        }}
      >
        <Settings className="size-5" aria-hidden />
        <span>Settings</span>
      </Link>
    </nav>
  );
};
