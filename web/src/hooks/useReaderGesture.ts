import { useState, useRef, useCallback, useEffect } from 'react';

export interface UseReaderGestureOptions {
  readingMode: string;
  isPaged: boolean;
  currentPage: number;
  totalPages: number;
  hasNextChapter?: boolean;
  hasPrevChapter?: boolean;
  containerWidth?: number | (() => number);
  onNextPage: () => void;
  onPrevPage: () => void;
  disabled?: boolean;
}

export interface UseReaderGestureResult {
  dragOffset: number;
  isDragging: boolean;
  isAnimating: boolean;
  didDragRef: React.MutableRefObject<boolean>;
  transitionStyle: string;
  setDragOffset: React.Dispatch<React.SetStateAction<number>>;
  setIsAnimating: React.Dispatch<React.SetStateAction<boolean>>;
  setIsDragging: React.Dispatch<React.SetStateAction<boolean>>;
  resetGesture: () => void;
  handlers: {
    onTouchStart: (e: React.TouchEvent) => void;
    onTouchMove: (e: React.TouchEvent) => void;
    onTouchEnd: (e?: React.TouchEvent) => void;
    onTouchCancel: (e?: React.TouchEvent) => void;
    onMouseDown: (e: React.MouseEvent) => void;
    onMouseMove: (e: React.MouseEvent) => void;
    onMouseUp: (e?: React.MouseEvent) => void;
    onMouseLeave: (e?: React.MouseEvent) => void;
  };
}

export function useReaderGesture({
  readingMode,
  isPaged,
  currentPage,
  totalPages,
  hasNextChapter = false,
  hasPrevChapter = false,
  containerWidth,
  onNextPage,
  onPrevPage,
  disabled = false,
}: UseReaderGestureOptions): UseReaderGestureResult {
  const [dragOffset, setDragOffset] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isAnimating, setIsAnimating] = useState(false);

  const didDragRef = useRef(false);
  const startPosRef = useRef<{ x: number; y: number; time: number } | null>(null);
  const animTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const getWidth = useCallback((): number => {
    if (typeof containerWidth === 'function') {
      return containerWidth();
    }
    if (typeof containerWidth === 'number' && containerWidth > 0) {
      return containerWidth;
    }
    if (typeof window !== 'undefined') {
      return window.innerWidth;
    }
    return 800;
  }, [containerWidth]);

  const resetGesture = useCallback(() => {
    startPosRef.current = null;
    didDragRef.current = false;
    setDragOffset(0);
    setIsDragging(false);
    setIsAnimating(false);
    if (animTimeoutRef.current) {
      clearTimeout(animTimeoutRef.current);
      animTimeoutRef.current = null;
    }
  }, []);

  // Clean up timers on unmount
  useEffect(() => {
    return () => {
      if (animTimeoutRef.current) {
        clearTimeout(animTimeoutRef.current);
      }
    };
  }, []);

  const handleStart = (clientX: number, clientY: number) => {
    if (!isPaged || disabled || isAnimating) return;
    startPosRef.current = {
      x: clientX,
      y: clientY,
      time: Date.now(),
    };
    didDragRef.current = false;
    setIsDragging(false);
    setDragOffset(0);
  };

  const handleMove = (clientX: number, clientY: number) => {
    if (!startPosRef.current || !isPaged || disabled || isAnimating) return;

    const deltaX = clientX - startPosRef.current.x;
    const deltaY = clientY - startPosRef.current.y;

    // Check if horizontal movement dominates vertical scroll
    if (Math.abs(deltaX) > 10 && Math.abs(deltaX) > Math.abs(deltaY)) {
      didDragRef.current = true;
      setIsDragging(true);
      setDragOffset(deltaX);
    }
  };

  const handleEnd = () => {
    if (!startPosRef.current || !isPaged || disabled) return;

    const startInfo = startPosRef.current;
    const currentDrag = dragOffset;
    const wasDragging = didDragRef.current;
    const elapsed = Math.max(1, Date.now() - startInfo.time);
    const velocity = currentDrag / elapsed; // px/ms

    startPosRef.current = null;
    setIsDragging(false);

    if (!wasDragging || isAnimating) {
      setDragOffset(0);
      return;
    }

    const width = getWidth();
    const distanceThreshold = Math.max(50, width * 0.15);
    const isFlick = Math.abs(velocity) > 0.4 && Math.abs(currentDrag) > 20;
    const meetsThreshold = Math.abs(currentDrag) >= distanceThreshold || isFlick;

    if (meetsThreshold) {
      // Determine intended direction
      // RTL: Drag right (currentDrag > 0) -> Next Page, Drag left (currentDrag < 0) -> Prev Page
      // LTR: Drag left (currentDrag < 0) -> Next Page, Drag right (currentDrag > 0) -> Prev Page
      const isNext = readingMode === 'rtl' ? currentDrag > 0 : currentDrag < 0;

      if (isNext) {
        if (currentPage < totalPages || hasNextChapter) {
          onNextPage();
        } else {
          // At the end, spring back
          setIsAnimating(true);
          setDragOffset(0);
          animTimeoutRef.current = setTimeout(() => {
            setIsAnimating(false);
            didDragRef.current = false;
          }, 220);
        }
      } else {
        if (currentPage > 1 || hasPrevChapter) {
          onPrevPage();
        } else {
          // At the beginning, spring back
          setIsAnimating(true);
          setDragOffset(0);
          animTimeoutRef.current = setTimeout(() => {
            setIsAnimating(false);
            didDragRef.current = false;
          }, 220);
        }
      }
    } else {
      // Below threshold: spring back smoothly to 0
      setIsAnimating(true);
      setDragOffset(0);
      animTimeoutRef.current = setTimeout(() => {
        setIsAnimating(false);
        didDragRef.current = false;
      }, 220);
    }
  };

  const handleCancel = () => {
    startPosRef.current = null;
    setIsDragging(false);
    setDragOffset(0);
    animTimeoutRef.current = setTimeout(() => {
      didDragRef.current = false;
    }, 100);
  };

  // Touch Handlers
  const onTouchStart = (e: React.TouchEvent) => {
    if (e.touches.length === 1) {
      handleStart(e.touches[0].clientX, e.touches[0].clientY);
    }
  };

  const onTouchMove = (e: React.TouchEvent) => {
    if (e.touches.length === 1) {
      handleMove(e.touches[0].clientX, e.touches[0].clientY);
    }
  };

  const onTouchEnd = () => handleEnd();
  const onTouchCancel = () => handleCancel();

  // Mouse Drag Handlers
  const onMouseDown = (e: React.MouseEvent) => {
    if (e.button === 0) {
      handleStart(e.clientX, e.clientY);
    }
  };

  const onMouseMove = (e: React.MouseEvent) => {
    if (startPosRef.current) {
      handleMove(e.clientX, e.clientY);
    }
  };

  const onMouseUp = () => handleEnd();
  const onMouseLeave = () => {
    if (startPosRef.current) {
      handleEnd();
    }
  };

  const transitionStyle = isDragging
    ? 'none'
    : isAnimating
    ? 'transform 220ms cubic-bezier(0.16, 1, 0.3, 1)'
    : 'none';

  return {
    dragOffset,
    isDragging,
    isAnimating,
    didDragRef,
    transitionStyle,
    setDragOffset,
    setIsAnimating,
    setIsDragging,
    resetGesture,
    handlers: {
      onTouchStart,
      onTouchMove,
      onTouchEnd,
      onTouchCancel,
      onMouseDown,
      onMouseMove,
      onMouseUp,
      onMouseLeave,
    },
  };
}
