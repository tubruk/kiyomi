import React from 'react';
import { useNavigate } from '@tanstack/react-router';
import { settingsTabRoute } from '../router';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs';
import {
  SettingsPluginsTab,
  SettingsCacheTab,
  SettingsJobsTab,
  SettingsAboutTab,
} from '../components/settings';

export const SettingsPage: React.FC = () => {
  const navigate = useNavigate();
  const { tab } = settingsTabRoute.useParams();
  const currentTab = tab && ['plugins', 'cache', 'jobs', 'about'].includes(tab) ? tab : 'plugins';
  const handleTabChange = (newTab: string) =>
    navigate({ to: '/settings/$tab', params: { tab: newTab } });

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 space-y-6">
      {/* Page Header */}
      <div className="space-y-1 border-b border-border pb-5">
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Settings</h1>
        <p className="text-xs sm:text-sm text-muted-foreground">
          Manage your content providers, system storage, and app information.
        </p>
      </div>

      {/* Tabs */}
      <Tabs value={currentTab} onValueChange={handleTabChange}>
        <TabsList className="mb-2">
          <TabsTrigger value="plugins">Plugins</TabsTrigger>
          <TabsTrigger value="cache">Cache</TabsTrigger>
          <TabsTrigger value="jobs">Jobs</TabsTrigger>
          <TabsTrigger value="about">About</TabsTrigger>
        </TabsList>

        {/* ── Plugins Tab ── */}
        <TabsContent value="plugins" className="space-y-6">
          <SettingsPluginsTab />
        </TabsContent>

        {/* ── Cache Tab ── */}
        <TabsContent value="cache" className="space-y-6">
          <SettingsCacheTab />
        </TabsContent>

        {/* ── Jobs Tab ── */}
        <TabsContent value="jobs" className="space-y-6">
          <SettingsJobsTab />
        </TabsContent>

        {/* ── About Tab ── */}
        <TabsContent value="about" className="space-y-6">
          <SettingsAboutTab />
        </TabsContent>
      </Tabs>
    </div>
  );
};
