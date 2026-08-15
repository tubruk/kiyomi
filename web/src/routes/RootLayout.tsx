import React from 'react';
import { Outlet } from '@tanstack/react-router';
import { Header } from '../components/Header';
import { MobileBottomNav } from '../components/MobileBottomNav';
import { ToastContainer } from '../components/ToastContainer';
import { ToastProvider } from '../context/ToastContext';
import { ThemeProvider } from '../context/ThemeContext';

export const RootLayout: React.FC = () => {
  return (
    <ThemeProvider>
      <ToastProvider>
        <div className="min-h-screen flex flex-col bg-background text-foreground transition-colors duration-200">
          <Header />
          <main className="flex-1 w-full max-w-7xl mx-auto px-4 py-6 sm:px-6 pb-20 md:pb-8">
            <Outlet />
          </main>
          <MobileBottomNav />
          <ToastContainer />
        </div>
      </ToastProvider>
    </ThemeProvider>
  );
};
