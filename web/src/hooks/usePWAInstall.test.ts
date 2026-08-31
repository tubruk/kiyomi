import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePWAInstall, BeforeInstallPromptEvent } from './usePWAInstall';

describe('usePWAInstall', () => {
  let matchMediaMock: ReturnType<typeof vi.fn>;
  let originalMatchMedia: typeof window.matchMedia;

  beforeEach(() => {
    vi.clearAllMocks();
    originalMatchMedia = window.matchMedia;
    matchMediaMock = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
    window.matchMedia = matchMediaMock as any;

    delete (window.navigator as any).standalone;
  });

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
  });

  describe('Initial standalone detection', () => {
    it('initializes as not installed and not installable by default in standard browser', () => {
      const { result } = renderHook(() => usePWAInstall());

      expect(result.current.isInstalled).toBe(false);
      expect(result.current.isInstallable).toBe(false);
    });

    it('detects standalone mode from matchMedia display-mode: standalone', () => {
      matchMediaMock.mockImplementation((query: string) => ({
        matches: query === '(display-mode: standalone)',
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }));

      const { result } = renderHook(() => usePWAInstall());

      expect(result.current.isInstalled).toBe(true);
      expect(result.current.isInstallable).toBe(false);
    });

    it('detects standalone mode from iOS Safari navigator.standalone', () => {
      Object.defineProperty(window.navigator, 'standalone', {
        value: true,
        configurable: true,
      });

      const { result } = renderHook(() => usePWAInstall());

      expect(result.current.isInstalled).toBe(true);
      expect(result.current.isInstallable).toBe(false);
    });
  });

  describe('Event listeners and install prompt lifecycle', () => {
    it('captures beforeinstallprompt event, prevents default, and marks as installable', () => {
      const { result } = renderHook(() => usePWAInstall());
      expect(result.current.isInstallable).toBe(false);

      const preventDefault = vi.fn();
      const promptEvent = new Event('beforeinstallprompt') as BeforeInstallPromptEvent;
      promptEvent.preventDefault = preventDefault;

      act(() => {
        window.dispatchEvent(promptEvent);
      });

      expect(preventDefault).toHaveBeenCalledTimes(1);
      expect(result.current.isInstallable).toBe(true);
    });

    it('handles appinstalled event by resetting prompt and setting isInstalled to true', () => {
      const { result } = renderHook(() => usePWAInstall());

      const promptEvent = new Event('beforeinstallprompt') as BeforeInstallPromptEvent;
      promptEvent.preventDefault = vi.fn();

      act(() => {
        window.dispatchEvent(promptEvent);
      });
      expect(result.current.isInstallable).toBe(true);
      expect(result.current.isInstalled).toBe(false);

      act(() => {
        window.dispatchEvent(new Event('appinstalled'));
      });

      expect(result.current.isInstalled).toBe(true);
      expect(result.current.isInstallable).toBe(false);
    });

    it('cleans up event listeners on unmount', () => {
      const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener');
      const { unmount } = renderHook(() => usePWAInstall());

      unmount();

      expect(removeEventListenerSpy).toHaveBeenCalledWith(
        'beforeinstallprompt',
        expect.any(Function)
      );
      expect(removeEventListenerSpy).toHaveBeenCalledWith(
        'appinstalled',
        expect.any(Function)
      );
    });
  });

  describe('install() function', () => {
    it('returns false immediately if no deferredPrompt is available', async () => {
      const { result } = renderHook(() => usePWAInstall());

      let installed: boolean | undefined;
      await act(async () => {
        installed = await result.current.install();
      });

      expect(installed).toBe(false);
    });

    it('handles accepted prompt outcome: triggers prompt(), sets isInstalled: true, returns true', async () => {
      const { result } = renderHook(() => usePWAInstall());

      const promptMock = vi.fn().mockResolvedValue(undefined);
      const promptEvent = new Event('beforeinstallprompt') as BeforeInstallPromptEvent;
      promptEvent.preventDefault = vi.fn();
      promptEvent.prompt = promptMock;
      Object.defineProperty(promptEvent, 'userChoice', {
        value: Promise.resolve({ outcome: 'accepted', platform: 'web' }),
      });

      act(() => {
        window.dispatchEvent(promptEvent);
      });
      expect(result.current.isInstallable).toBe(true);

      let installResult: boolean | undefined;
      await act(async () => {
        installResult = await result.current.install();
      });

      expect(promptMock).toHaveBeenCalledTimes(1);
      expect(installResult).toBe(true);
      expect(result.current.isInstalled).toBe(true);
      expect(result.current.isInstallable).toBe(false);
    });

    it('handles dismissed prompt outcome: clears deferredPrompt, remains not installed, returns false', async () => {
      const { result } = renderHook(() => usePWAInstall());

      const promptMock = vi.fn().mockResolvedValue(undefined);
      const promptEvent = new Event('beforeinstallprompt') as BeforeInstallPromptEvent;
      promptEvent.preventDefault = vi.fn();
      promptEvent.prompt = promptMock;
      Object.defineProperty(promptEvent, 'userChoice', {
        value: Promise.resolve({ outcome: 'dismissed', platform: 'web' }),
      });

      act(() => {
        window.dispatchEvent(promptEvent);
      });
      expect(result.current.isInstallable).toBe(true);

      let installResult: boolean | undefined;
      await act(async () => {
        installResult = await result.current.install();
      });

      expect(promptMock).toHaveBeenCalledTimes(1);
      expect(installResult).toBe(false);
      expect(result.current.isInstalled).toBe(false);
      expect(result.current.isInstallable).toBe(false);
    });

    it('handles prompt error gracefully and returns false', async () => {
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const { result } = renderHook(() => usePWAInstall());

      const promptEvent = new Event('beforeinstallprompt') as BeforeInstallPromptEvent;
      promptEvent.preventDefault = vi.fn();
      promptEvent.prompt = vi.fn().mockRejectedValue(new Error('Prompt failed'));

      act(() => {
        window.dispatchEvent(promptEvent);
      });

      let installResult: boolean | undefined;
      await act(async () => {
        installResult = await result.current.install();
      });

      expect(installResult).toBe(false);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'PWA install prompt error:',
        expect.any(Error)
      );

      consoleErrorSpy.mockRestore();
    });
  });
});
