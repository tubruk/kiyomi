import React from 'react';
import { Link, useLocation } from '@tanstack/react-router';
import { BookOpen, Sun, Moon, Monitor, Check, Settings } from 'lucide-react';

import { Badge } from './ui/badge';
import { useTheme } from '../context/ThemeContext';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';

export const Header: React.FC = () => {
  const location = useLocation();
  const { theme, setTheme, isDark } = useTheme();

  const isReaderPage = location.pathname.startsWith('/reader/') || location.pathname.includes('/chapter/');
  if (isReaderPage) {
    return null;
  }

  const isExploreActive = location.pathname.startsWith('/explore') || location.pathname.startsWith('/providers');
  const isSettingsActive = location.pathname.startsWith('/settings') || location.pathname.startsWith('/plugins');

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border bg-background/80 backdrop-blur-md">
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-3 sm:px-6">
        {/* Brand Logo & Desktop Nav */}
        <div className="flex items-center gap-6">
          <Link
            to="/"
            className="flex items-center gap-2 text-foreground transition-opacity hover:opacity-90"
          >
            <BookOpen className="size-6 text-primary" aria-hidden />
            <span className="text-xl font-bold tracking-tight bg-gradient-to-r from-foreground via-foreground to-primary bg-clip-text text-transparent">
              Kiyomi
            </span>
            <Badge variant="outline" className="hidden border-primary/30 text-primary text-[10px] sm:inline-flex font-semibold">
              Local
            </Badge>
          </Link>

          {/* Desktop Navigation Links */}
          <nav className="hidden items-center gap-1 md:flex">
            <Link
              to="/"
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                location.pathname === '/'
                  ? 'bg-secondary text-primary font-semibold'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              }`}
            >
              Library
            </Link>
            <Link
              to="/explore"
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                isExploreActive
                  ? 'bg-secondary text-primary font-semibold'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              }`}
            >
              Explore
            </Link>
          </nav>
        </div>

        {/* Actions: Settings & Theme Switcher */}
        <div className="flex items-center gap-3">
          <Link
            to="/settings/plugins"
            className={`inline-flex size-9 items-center justify-center rounded-full border border-border bg-card transition-colors cursor-pointer shrink-0 ${
              isSettingsActive
                ? 'bg-secondary text-primary'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            }`}
            aria-label="Settings"
            title="Settings"
          >
            <Settings className="size-4" />
          </Link>

          <DropdownMenu>
            <DropdownMenuTrigger
              className="inline-flex size-9 items-center justify-center rounded-full border border-border bg-card text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer shrink-0"
              aria-label="Toggle theme"
            >
              {isDark ? <Moon className="size-4 text-primary" /> : <Sun className="size-4 text-amber-500" />}
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-36">
              <DropdownMenuItem onClick={() => setTheme('light')} className="text-xs cursor-pointer gap-2">
                <Sun className="size-3.5 text-amber-500" />
                <span>Light</span>
                {theme === 'light' && <Check className="size-3.5 ml-auto text-primary" />}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setTheme('dark')} className="text-xs cursor-pointer gap-2">
                <Moon className="size-3.5 text-primary" />
                <span>Dark</span>
                {theme === 'dark' && <Check className="size-3.5 ml-auto text-primary" />}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setTheme('system')} className="text-xs cursor-pointer gap-2">
                <Monitor className="size-3.5 text-muted-foreground" />
                <span>System</span>
                {theme === 'system' && <Check className="size-3.5 ml-auto text-primary" />}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
};
