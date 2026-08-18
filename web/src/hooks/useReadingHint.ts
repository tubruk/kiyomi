import { useState, useEffect, useCallback, useRef } from 'react';

export interface ReadingHint {
  readingMode: string;
  message: string;
  visible: boolean;
}

export function useReadingHint(
  mangaId: string | undefined,
  readingMode: string
): {
  hint: ReadingHint | null;
  dismissHint: () => void;
  showHint: (customMessage?: string) => void;
} {
  const [hint, setHint] = useState<ReadingHint | null>(null);
  const dismissTimerRef = useRef<NodeJS.Timeout | null>(null);
  const prevModeRef = useRef<string | null>(null);

  const getStorageKey = useCallback(
    (mode: string) => `kiyomi_reading_hint_${mangaId}_${mode}`,
    [mangaId]
  );

  const dismissHint = useCallback(() => {
    if (dismissTimerRef.current) {
      clearTimeout(dismissTimerRef.current);
      dismissTimerRef.current = null;
    }
    setHint(null);
  }, []);

  const showHint = useCallback(
    (customMessage?: string) => {
      if (dismissTimerRef.current) {
        clearTimeout(dismissTimerRef.current);
      }

      let message = customMessage;
      if (!message) {
        switch (readingMode) {
          case 'rtl':
            message = 'Tap left/right to navigate';
            break;
          case 'ltr':
            message = 'Tap left/right to navigate';
            break;
          case 'vertical':
            message = 'Scroll to navigate';
            break;
          case 'longstrip':
            message = 'Scroll to navigate';
            break;
          default:
            message = 'Tap left/right to navigate';
        }
      }

      const newHint: ReadingHint = {
        readingMode,
        message,
        visible: true,
      };

      setHint(newHint);

      dismissTimerRef.current = setTimeout(() => {
        setHint(null);
        dismissTimerRef.current = null;
      }, 2000);
    },
    [readingMode]
  );

  useEffect(() => {
    if (!mangaId || !readingMode) return;

    const storageKey = getStorageKey(readingMode);
    const hasSeenHint = sessionStorage.getItem(storageKey);

    // First visit or mode changed mid-session
    if (!hasSeenHint || prevModeRef.current !== readingMode) {
      sessionStorage.setItem(storageKey, 'true');
      prevModeRef.current = readingMode;
      showHint();
    }

    return () => {
      if (dismissTimerRef.current) {
        clearTimeout(dismissTimerRef.current);
      }
    };
  }, [mangaId, readingMode, getStorageKey, showHint]);

  return { hint, dismissHint, showHint };
}
