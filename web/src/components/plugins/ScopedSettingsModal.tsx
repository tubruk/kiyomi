import React, { useState, useEffect } from 'react';
import { Sliders, Eye, EyeOff, Save } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Badge } from '../ui/badge';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../ui/tabs';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { PluginItem, PluginSettingSpec } from '../../types/api';
import { api } from '../../api/client';
import { useToast } from '../../context/ToastContext';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../lib/queryKeys';
import { ErrorDetailsModal } from '../ErrorDetailsModal';

interface ScopedSettingsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  plugin: PluginItem | null;
}

export const ScopedSettingsModal: React.FC<ScopedSettingsModalProps> = ({
  open,
  onOpenChange,
  plugin,
}) => {
  const { showToast } = useToast();
  const queryClient = useQueryClient();

  const [globalValues, setGlobalValues] = useState<Record<string, string>>({});
  const [providerValues, setProviderValues] = useState<Record<string, Record<string, string>>>({});
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});
  const [isSaving, setIsSaving] = useState(false);

  // Error modal state
  const [errorModalOpen, setErrorModalOpen] = useState(false);
  const [errorTitle, setErrorTitle] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [errorDetails, setErrorDetails] = useState('');

  // Sync state whenever modal opens or plugin changes
  useEffect(() => {
    if (!plugin) return;

    // Initialize global config
    const gVals: Record<string, string> = {};
    if (plugin.pluginSettingsSchema) {
      for (const s of plugin.pluginSettingsSchema) {
        gVals[s.key] = plugin.globalConfig?.[s.key] ?? s.defaultValue ?? '';
      }
    }
    setGlobalValues(gVals);

    // Initialize provider-specific configs
    const pVals: Record<string, Record<string, string>> = {};
    if (plugin.providers) {
      for (const prov of plugin.providers) {
        pVals[prov.id] = {};
        if (prov.settingsSchema) {
          for (const s of prov.settingsSchema) {
            pVals[prov.id][s.key] =
              plugin.providerConfigs?.[prov.id]?.[s.key] ?? s.defaultValue ?? '';
          }
        }
      }
    }
    setProviderValues(pVals);
    setShowSecrets({});
  }, [plugin, open]);

  if (!plugin) return null;

  const hasGlobalSchema = (plugin.pluginSettingsSchema?.length ?? 0) > 0;
  const providersWithSchema = (plugin.providers || []).filter(
    (p) => (p.settingsSchema?.length ?? 0) > 0
  );
  const hasAnySettings = hasGlobalSchema || providersWithSchema.length > 0;

  const toggleSecret = (key: string) => {
    setShowSecrets((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const handleGlobalChange = (key: string, value: string) => {
    setGlobalValues((prev) => ({ ...prev, [key]: value }));
  };

  const handleProviderChange = (providerId: string, key: string, value: string) => {
    setProviderValues((prev) => ({
      ...prev,
      [providerId]: {
        ...(prev[providerId] || {}),
        [key]: value,
      },
    }));
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.updatePluginConfig(plugin.pluginId, {
        globalConfig: globalValues,
        providerConfigs: providerValues,
      });

      showToast(`Saved settings for ${plugin.pluginName}`, 'success');
      await queryClient.invalidateQueries({ queryKey: queryKeys.plugins.all });
      await queryClient.invalidateQueries({ queryKey: queryKeys.sources.all });
      onOpenChange(false);
    } catch (err: any) {
      console.error('Failed to save plugin config:', err);
      const msg = err.message || 'Failed to update plugin configuration';
      const details = err.details
        ? typeof err.details === 'object'
          ? JSON.stringify(err.details, null, 2)
          : String(err.details)
        : err.stack || '';

      showToast(msg, 'error', details);
      setErrorTitle(`Failed to configure ${plugin.pluginName}`);
      setErrorMessage(msg);
      setErrorDetails(details);
    } finally {
      setIsSaving(false);
    }
  };

  const renderSettingField = (
    setting: PluginSettingSpec,
    value: string,
    onChange: (val: string) => void,
    fieldId: string
  ) => {
    const isSecret = setting.type === 'secret';
    const isBool = setting.type === 'boolean';
    const isSelect = setting.type === 'select';
    const isNumber = setting.type === 'number';

    return (
      <div key={setting.key} className="space-y-1.5 rounded-lg border border-border/70 bg-card/60 p-3.5 transition-colors hover:border-border">
        <div className="flex items-center justify-between gap-2">
          <label htmlFor={fieldId} className="text-xs font-semibold text-foreground">
            {setting.label || setting.key}
          </label>
          <Badge variant="outline" className="text-[10px] font-mono text-muted-foreground">
            {setting.key}
          </Badge>
        </div>

        {setting.description && (
          <p className="text-xs text-muted-foreground leading-relaxed">
            {setting.description}
          </p>
        )}

        <div className="pt-1">
          {isSecret ? (
            <div className="relative flex items-center">
              <Input
                id={fieldId}
                type={showSecrets[fieldId] ? 'text' : 'password'}
                value={value}
                onChange={(e) => onChange(e.target.value)}
                placeholder={setting.defaultValue || 'Enter secret...'}
                className="pr-10 text-xs font-mono"
              />
              <button
                type="button"
                onClick={() => toggleSecret(fieldId)}
                className="absolute right-2.5 text-muted-foreground hover:text-foreground transition-colors p-1"
                aria-label={showSecrets[fieldId] ? 'Hide secret' : 'Show secret'}
              >
                {showSecrets[fieldId] ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </button>
            </div>
          ) : isBool ? (
            <div className="flex items-center gap-3">
              <button
                id={fieldId}
                type="button"
                role="switch"
                aria-checked={value === 'true'}
                onClick={() => onChange(value === 'true' ? 'false' : 'true')}
                className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                  value === 'true' ? 'bg-primary' : 'bg-muted-foreground/30'
                }`}
              >
                <span
                  className={`pointer-events-none inline-block size-5 transform rounded-full bg-background shadow-lg ring-0 transition-transform ${
                    value === 'true' ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </button>
              <span className="text-xs font-medium text-muted-foreground">
                {value === 'true' ? 'Enabled' : 'Disabled'}
              </span>
            </div>
          ) : isSelect ? (
            <Select value={value || setting.defaultValue || ''} onValueChange={(val) => onChange(val ?? '')}>
              <SelectTrigger id={fieldId} className="w-full text-xs">
                <SelectValue placeholder="Select an option" />
              </SelectTrigger>
              <SelectContent>
                {(setting.options || []).map((opt) => (
                  <SelectItem key={opt} value={opt} className="text-xs">
                    {opt}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              id={fieldId}
              type={isNumber ? 'number' : 'text'}
              value={value}
              onChange={(e) => onChange(e.target.value)}
              placeholder={setting.defaultValue || `Enter ${setting.label || setting.key}...`}
              className="text-xs"
            />
          )}
        </div>
      </div>
    );
  };

  const defaultTab = hasGlobalSchema
    ? 'global'
    : providersWithSchema.length > 0
    ? providersWithSchema[0].id
    : 'global';

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl sm:max-w-2xl max-h-[85vh] flex flex-col p-0 overflow-hidden">
          <DialogHeader className="px-6 pt-6 pb-4 border-b border-border/80 bg-card/40">
            <div className="flex items-center gap-2">
              <div className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Sliders className="size-5" />
              </div>
              <div>
                <DialogTitle className="text-lg font-bold tracking-tight text-foreground">
                  Configure {plugin.pluginName}
                </DialogTitle>
                <DialogDescription className="text-xs text-muted-foreground">
                  Scoped runtime configuration parameters and credential keys.
                </DialogDescription>
              </div>
            </div>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto px-6 py-4">
            {!hasAnySettings ? (
              <div className="flex flex-col items-center justify-center py-10 text-center">
                <div className="size-12 rounded-full bg-muted/60 flex items-center justify-center text-muted-foreground mb-3">
                  <Sliders className="size-6 opacity-40" />
                </div>
                <h4 className="text-sm font-semibold text-foreground">No Configurable Settings</h4>
                <p className="text-xs text-muted-foreground max-w-sm mt-1">
                  This plugin does not declare any custom setting specifications or credentials in its schema.
                </p>
              </div>
            ) : (
              <Tabs defaultValue={defaultTab} className="w-full">
                <TabsList className="mb-4 w-full justify-start overflow-x-auto">
                  {hasGlobalSchema && (
                    <TabsTrigger value="global" className="text-xs">
                      Plugin Global
                    </TabsTrigger>
                  )}
                  {providersWithSchema.map((prov) => (
                    <TabsTrigger key={prov.id} value={prov.id} className="text-xs">
                      {prov.name || prov.id}
                    </TabsTrigger>
                  ))}
                </TabsList>

                {hasGlobalSchema && (
                  <TabsContent value="global" className="space-y-3 pt-1">
                    <p className="text-xs text-muted-foreground mb-2">
                      Plugin-wide parameters shared across all contained provider instances.
                    </p>
                    {plugin.pluginSettingsSchema?.map((s) =>
                      renderSettingField(
                        s,
                        globalValues[s.key] ?? '',
                        (val) => handleGlobalChange(s.key, val),
                        `global-${s.key}`
                      )
                    )}
                  </TabsContent>
                )}

                {providersWithSchema.map((prov) => (
                  <TabsContent key={prov.id} value={prov.id} className="space-y-3 pt-1">
                    <p className="text-xs text-muted-foreground mb-2">
                      Configuration scoped specifically to provider <span className="font-semibold text-foreground">{prov.name}</span> (<code className="text-primary">{prov.id}</code>).
                    </p>
                    {prov.settingsSchema?.map((s) =>
                      renderSettingField(
                        s,
                        providerValues[prov.id]?.[s.key] ?? '',
                        (val) => handleProviderChange(prov.id, s.key, val),
                        `prov-${prov.id}-${s.key}`
                      )
                    )}
                  </TabsContent>
                ))}
              </Tabs>
            )}
          </div>

          <DialogFooter className="px-6 py-4 border-t border-border/80 bg-muted/30 flex items-center justify-between gap-3">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
              className="text-xs cursor-pointer"
            >
              Cancel
            </Button>
            {hasAnySettings && (
              <Button
                type="button"
                size="sm"
                onClick={handleSave}
                disabled={isSaving}
                className="text-xs cursor-pointer gap-1.5"
              >
                {isSaving ? (
                  <>
                    <div className="size-3.5 border-2 border-current border-t-transparent animate-spin rounded-full" />
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="size-3.5" />
                    <span>Save Settings</span>
                  </>
                )}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ErrorDetailsModal
        open={errorModalOpen}
        onOpenChange={setErrorModalOpen}
        title={errorTitle}
        message={errorMessage}
        details={errorDetails}
      />
    </>
  );
};
