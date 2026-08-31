import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useReaderGesture } from './useReaderGesture';

describe('useReaderGesture', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('updates dragOffset and isDragging on touch move', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'rtl',
        isPaged: true,
        currentPage: 1,
        totalPages: 10,
        containerWidth: 800,
        onNextPage,
        onPrevPage,
      })
    );

    // Touch start
    act(() => {
      result.current.handlers.onTouchStart({
        touches: [{ clientX: 200, clientY: 100 }],
      } as any);
    });

    expect(result.current.isDragging).toBe(false);
    expect(result.current.dragOffset).toBe(0);

    // Touch move horizontally (deltaX = 50)
    act(() => {
      result.current.handlers.onTouchMove({
        touches: [{ clientX: 250, clientY: 102 }],
      } as any);
    });

    expect(result.current.isDragging).toBe(true);
    expect(result.current.dragOffset).toBe(50);
  });

  it('triggers onNextPage in RTL when swiping right past threshold', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'rtl',
        isPaged: true,
        currentPage: 2,
        totalPages: 10,
        containerWidth: 800, // threshold = 120px
        onNextPage,
        onPrevPage,
      })
    );

    act(() => {
      result.current.handlers.onTouchStart({
        touches: [{ clientX: 200, clientY: 100 }],
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchMove({
        touches: [{ clientX: 350, clientY: 100 }], // +150px
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchEnd();
    });

    expect(onNextPage).toHaveBeenCalledTimes(1);
    expect(onPrevPage).not.toHaveBeenCalled();
  });

  it('triggers onPrevPage in RTL when swiping left past threshold', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'rtl',
        isPaged: true,
        currentPage: 2,
        totalPages: 10,
        containerWidth: 800,
        onNextPage,
        onPrevPage,
      })
    );

    act(() => {
      result.current.handlers.onTouchStart({
        touches: [{ clientX: 300, clientY: 100 }],
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchMove({
        touches: [{ clientX: 150, clientY: 100 }], // -150px
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchEnd();
    });

    expect(onPrevPage).toHaveBeenCalledTimes(1);
    expect(onNextPage).not.toHaveBeenCalled();
  });

  it('triggers onNextPage in LTR when swiping left past threshold', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'ltr',
        isPaged: true,
        currentPage: 2,
        totalPages: 10,
        containerWidth: 800,
        onNextPage,
        onPrevPage,
      })
    );

    act(() => {
      result.current.handlers.onTouchStart({
        touches: [{ clientX: 300, clientY: 100 }],
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchMove({
        touches: [{ clientX: 150, clientY: 100 }], // -150px
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchEnd();
    });

    expect(onNextPage).toHaveBeenCalledTimes(1);
    expect(onPrevPage).not.toHaveBeenCalled();
  });

  it('springs back to 0 when swipe is below threshold', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'rtl',
        isPaged: true,
        currentPage: 2,
        totalPages: 10,
        containerWidth: 800,
        onNextPage,
        onPrevPage,
      })
    );

    act(() => {
      result.current.handlers.onTouchStart({
        touches: [{ clientX: 200, clientY: 100 }],
      } as any);
    });

    // Advance time so velocity is low
    act(() => {
      vi.advanceTimersByTime(500);
    });

    act(() => {
      result.current.handlers.onTouchMove({
        touches: [{ clientX: 220, clientY: 100 }], // +20px (below threshold)
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchEnd();
    });

    expect(onNextPage).not.toHaveBeenCalled();
    expect(onPrevPage).not.toHaveBeenCalled();
    expect(result.current.dragOffset).toBe(0);
    expect(result.current.isAnimating).toBe(true);

    act(() => {
      vi.advanceTimersByTime(250);
    });

    expect(result.current.isAnimating).toBe(false);
  });

  it('supports mouse drag events', () => {
    const onNextPage = vi.fn();
    const onPrevPage = vi.fn();

    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'rtl',
        isPaged: true,
        currentPage: 2,
        totalPages: 10,
        containerWidth: 800,
        onNextPage,
        onPrevPage,
      })
    );

    act(() => {
      result.current.handlers.onMouseDown({
        button: 0,
        clientX: 200,
        clientY: 100,
      } as any);
    });

    act(() => {
      result.current.handlers.onMouseMove({
        clientX: 350,
        clientY: 100,
      } as any);
    });

    expect(result.current.isDragging).toBe(true);
    expect(result.current.dragOffset).toBe(150);

    act(() => {
      result.current.handlers.onMouseUp();
    });

    expect(onNextPage).toHaveBeenCalledTimes(1);
  });

  it('handles onTouchCancel properly', () => {
    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'rtl',
        isPaged: true,
        currentPage: 1,
        totalPages: 5,
        onNextPage: vi.fn(),
        onPrevPage: vi.fn(),
      })
    );

    act(() => {
      result.current.handlers.onTouchStart({
        touches: [{ clientX: 200, clientY: 100 }],
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchMove({
        touches: [{ clientX: 250, clientY: 100 }],
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchCancel();
    });

    expect(result.current.isDragging).toBe(false);
    expect(result.current.dragOffset).toBe(0);
  });

  it('does nothing when isPaged is false or disabled', () => {
    const onNextPage = vi.fn();

    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'vertical',
        isPaged: false,
        currentPage: 1,
        totalPages: 5,
        onNextPage,
        onPrevPage: vi.fn(),
      })
    );

    act(() => {
      result.current.handlers.onTouchStart({
        touches: [{ clientX: 200, clientY: 100 }],
      } as any);
    });

    act(() => {
      result.current.handlers.onTouchMove({
        touches: [{ clientX: 400, clientY: 100 }],
      } as any);
    });

    expect(result.current.isDragging).toBe(false);
    expect(result.current.dragOffset).toBe(0);
  });

  it('resetGesture cleans up state', () => {
    const { result } = renderHook(() =>
      useReaderGesture({
        readingMode: 'rtl',
        isPaged: true,
        currentPage: 1,
        totalPages: 5,
        onNextPage: vi.fn(),
        onPrevPage: vi.fn(),
      })
    );

    act(() => {
      result.current.setDragOffset(100);
      result.current.setIsDragging(true);
      result.current.setIsAnimating(true);
    });

    act(() => {
      result.current.resetGesture();
    });

    expect(result.current.dragOffset).toBe(0);
    expect(result.current.isDragging).toBe(false);
    expect(result.current.isAnimating).toBe(false);
  });
});
