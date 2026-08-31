import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useReaderScrollTracker } from './useReaderScrollTracker';

describe('useReaderScrollTracker', () => {
  let originalRAF: typeof window.requestAnimationFrame;

  beforeEach(() => {
    originalRAF = window.requestAnimationFrame;
    window.requestAnimationFrame = vi.fn((cb) => {
      cb(performance.now());
      return 1;
    });
  });

  afterEach(() => {
    window.requestAnimationFrame = originalRAF;
    vi.restoreAllMocks();
  });

  it('does not attach scroll event listener when isPaged is true', () => {
    const addEventSpy = vi.spyOn(window, 'addEventListener');
    const pageRefs = { current: [] };

    renderHook(() =>
      useReaderScrollTracker({
        isPaged: true,
        pagesCount: 5,
        pageRefs,
        onPageVisible: vi.fn(),
      })
    );

    expect(addEventSpy).not.toHaveBeenCalledWith('scroll', expect.any(Function), expect.any(Object));
  });

  it('attaches scroll event listener when isPaged is false', () => {
    const addEventSpy = vi.spyOn(window, 'addEventListener');
    const removeEventSpy = vi.spyOn(window, 'removeEventListener');
    const pageRefs = { current: [] };

    const { unmount } = renderHook(() =>
      useReaderScrollTracker({
        isPaged: false,
        pagesCount: 5,
        pageRefs,
        onPageVisible: vi.fn(),
      })
    );

    expect(addEventSpy).toHaveBeenCalledWith('scroll', expect.any(Function), { passive: true });

    unmount();
    expect(removeEventSpy).toHaveBeenCalledWith('scroll', expect.any(Function));
  });

  it('tracks visible page on scroll event using getBoundingClientRect', () => {
    const onPageVisible = vi.fn();

    const el1 = document.createElement('div');
    const el2 = document.createElement('div');
    const el3 = document.createElement('div');

    // Mock getBoundingClientRect
    // viewport mid = 384px (assuming window.innerHeight = 768)
    vi.spyOn(el1, 'getBoundingClientRect').mockReturnValue({ top: -400 } as any);
    vi.spyOn(el2, 'getBoundingClientRect').mockReturnValue({ top: 100 } as any); // visible (top <= 384)
    vi.spyOn(el3, 'getBoundingClientRect').mockReturnValue({ top: 900 } as any);

    const pageRefs = { current: [el1, el2, el3] };

    renderHook(() =>
      useReaderScrollTracker({
        isPaged: false,
        pagesCount: 3,
        pageRefs,
        onPageVisible,
        initialPage: 1,
      })
    );

    act(() => {
      window.dispatchEvent(new Event('scroll'));
    });

    expect(onPageVisible).toHaveBeenCalledWith(2);
  });

  it('scrollToPage calls scrollIntoView on target page element', () => {
    const el1 = document.createElement('div');
    const el2 = document.createElement('div');
    el2.scrollIntoView = vi.fn();

    const pageRefs = { current: [el1, el2] };

    const { result } = renderHook(() =>
      useReaderScrollTracker({
        isPaged: false,
        pagesCount: 2,
        pageRefs,
        onPageVisible: vi.fn(),
      })
    );

    act(() => {
      result.current.scrollToPage(2, true);
    });

    expect(el2.scrollIntoView).toHaveBeenCalledWith({
      behavior: 'smooth',
      block: 'start',
    });
  });
});
