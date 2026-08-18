import { useState, useCallback } from 'react';

export type FitMode = 'fit-width' | 'fit-height' | 'fit-original';

const STORAGE_KEY = 'kiyomi_reader_fit_mode';
const DEFAULT_FIT_MODE: FitMode = 'fit-height';

function getStoredFitMode(): FitMode {
  if (typeof window === 'undefined') return DEFAULT_FIT_MODE;
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'fit-width' || stored === 'fit-height' || stored === 'fit-original') {
    return stored;
  }
  return DEFAULT_FIT_MODE;
}

function storeFitMode(mode: FitMode): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(STORAGE_KEY, mode);
}

export function useReaderFitMode(): {
  fitMode: FitMode;
  setFitMode: (mode: FitMode) => void;
} {
  const [fitMode, setFitModeState] = useState<FitMode>(getStoredFitMode);

  const setFitMode = useCallback((mode: FitMode) => {
    storeFitMode(mode);
    setFitModeState(mode);
  }, []);

  return { fitMode, setFitMode };
}
