import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useReadingHint } from './useReadingHint';

describe('useReadingHint', () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows hint on first visit for a given manga and mode', () => {
    const { result } = renderHook(() => useReadingHint('manga-1', 'rtl'));

    expect(result.current.hint).toEqual({
      readingMode: 'rtl',
      message: 'Tap left/right to navigate',
      visible: true,
    });
    expect(sessionStorage.getItem('kiyomi_reading_hint_manga-1_rtl')).toBe('true');
  });

  it('provides mode-specific default messages', () => {
    const { result: vertResult } = renderHook(() => useReadingHint('manga-1', 'vertical'));
    expect(vertResult.current.hint?.message).toBe('Scroll to navigate');

    const { result: lsResult } = renderHook(() => useReadingHint('manga-1', 'longstrip'));
    expect(lsResult.current.hint?.message).toBe('Scroll to navigate');

    const { result: ltrResult } = renderHook(() => useReadingHint('manga-1', 'ltr'));
    expect(ltrResult.current.hint?.message).toBe('Tap left/right to navigate');
  });

  it('auto-dismisses after 2 seconds', () => {
    const { result } = renderHook(() => useReadingHint('manga-1', 'rtl'));
    expect(result.current.hint).not.toBeNull();

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(result.current.hint).toBeNull();
  });

  it('allows manual dismissHint to close immediately', () => {
    const { result } = renderHook(() => useReadingHint('manga-1', 'rtl'));
    expect(result.current.hint).not.toBeNull();

    act(() => {
      result.current.dismissHint();
    });

    expect(result.current.hint).toBeNull();
  });

  it('allows showHint with custom message', () => {
    const { result } = renderHook(() => useReadingHint('manga-1', 'rtl'));

    act(() => {
      result.current.showHint('Custom navigation tip');
    });

    expect(result.current.hint).toEqual({
      readingMode: 'rtl',
      message: 'Custom navigation tip',
      visible: true,
    });
  });
});
