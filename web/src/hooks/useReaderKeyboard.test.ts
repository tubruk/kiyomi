import { describe, it, expect, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useReaderKeyboard } from './useReaderKeyboard';

describe('useReaderKeyboard', () => {
  it('handles RTL arrow key navigation correctly', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'rtl',
        onNextPage,
        onPrevPage,
      })
    );

    // RTL: Left Arrow -> Next Page
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
    expect(onNextPage).toHaveBeenCalledTimes(1);
    expect(onPrevPage).not.toHaveBeenCalled();

    // RTL: Right Arrow -> Prev Page
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }));
    expect(onPrevPage).toHaveBeenCalledTimes(1);
  });

  it('handles LTR arrow key navigation correctly', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'ltr',
        onNextPage,
        onPrevPage,
      })
    );

    // LTR: Right Arrow -> Next Page
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }));
    expect(onNextPage).toHaveBeenCalledTimes(1);
    expect(onPrevPage).not.toHaveBeenCalled();

    // LTR: Left Arrow -> Prev Page
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
    expect(onPrevPage).toHaveBeenCalledTimes(1);
  });

  it('handles Spacebar navigation (next without Shift, prev with Shift)', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'rtl',
        onNextPage,
        onPrevPage,
      })
    );

    window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ' }));
    expect(onNextPage).toHaveBeenCalledTimes(1);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', shiftKey: true }));
    expect(onPrevPage).toHaveBeenCalledTimes(1);
  });

  it('handles PageUp and PageDown keys', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'ltr',
        onNextPage,
        onPrevPage,
      })
    );

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'PageDown' }));
    expect(onNextPage).toHaveBeenCalledTimes(1);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'PageUp' }));
    expect(onPrevPage).toHaveBeenCalledTimes(1);
  });

  it('handles Home and End keys', () => {
    const onFirstPage = vi.fn();
    const onLastPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'rtl',
        onFirstPage,
        onLastPage,
      })
    );

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home' }));
    expect(onFirstPage).toHaveBeenCalledTimes(1);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'End' }));
    expect(onLastPage).toHaveBeenCalledTimes(1);
  });

  it('handles overlay and fullscreen toggle shortcuts', () => {
    const onToggleOverlays = vi.fn();
    const onToggleFullscreen = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'rtl',
        onToggleOverlays,
        onToggleFullscreen,
      })
    );

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'm' }));
    expect(onToggleOverlays).toHaveBeenCalledTimes(1);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'f' }));
    expect(onToggleFullscreen).toHaveBeenCalledTimes(1);
  });

  it('ignores keys when typing in input, textarea or dialog', () => {
    const onNextPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'rtl',
        onNextPage,
      })
    );

    const input = document.createElement('input');
    document.body.appendChild(input);

    const event = new KeyboardEvent('keydown', { key: 'ArrowLeft' });
    Object.defineProperty(event, 'target', { value: input, enumerable: true });
    window.dispatchEvent(event);

    expect(onNextPage).not.toHaveBeenCalled();

    document.body.removeChild(input);
  });

  it('ignores key events when enabled is false', () => {
    const onNextPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'rtl',
        enabled: false,
        onNextPage,
      })
    );

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
    expect(onNextPage).not.toHaveBeenCalled();
  });

  it('ignores events with Ctrl/Meta/Alt modifiers', () => {
    const onNextPage = vi.fn();

    renderHook(() =>
      useReaderKeyboard({
        readingMode: 'rtl',
        onNextPage,
      })
    );

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', ctrlKey: true }));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', metaKey: true }));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', altKey: true }));

    expect(onNextPage).not.toHaveBeenCalled();
  });
});
