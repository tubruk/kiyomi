import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useReaderFitMode } from './useReaderFitMode';

describe('useReaderFitMode', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('defaults to fit-height when localStorage is empty', () => {
    const { result } = renderHook(() => useReaderFitMode());
    expect(result.current.fitMode).toBe('fit-height');
  });

  it('initializes with stored value from localStorage', () => {
    localStorage.setItem('kiyomi_reader_fit_mode', 'fit-width');
    const { result } = renderHook(() => useReaderFitMode());
    expect(result.current.fitMode).toBe('fit-width');
  });

  it('falls back to default if stored value is invalid', () => {
    localStorage.setItem('kiyomi_reader_fit_mode', 'invalid-mode');
    const { result } = renderHook(() => useReaderFitMode());
    expect(result.current.fitMode).toBe('fit-height');
  });

  it('updates state and persists to localStorage when setFitMode is called', () => {
    const { result } = renderHook(() => useReaderFitMode());

    act(() => {
      result.current.setFitMode('fit-original');
    });

    expect(result.current.fitMode).toBe('fit-original');
    expect(localStorage.getItem('kiyomi_reader_fit_mode')).toBe('fit-original');
  });
});
