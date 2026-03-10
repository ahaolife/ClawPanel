import { useState, useRef, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { Info } from 'lucide-react';

interface InfoTooltipProps {
  content: React.ReactNode;
  /** Icon size in px, default 13 */
  size?: number;
  /** Additional class on the wrapper */
  className?: string;
}

/**
 * A small ⓘ icon that shows a floating tooltip on hover / tap.
 * Renders via Portal to avoid ancestor overflow:hidden clipping.
 */
export default function InfoTooltip({ content, size = 13, className = '' }: InfoTooltipProps) {
  const [visible, setVisible] = useState(false);
  const [coords, setCoords] = useState({ top: 0, left: 0, flip: false });
  const iconRef = useRef<HTMLSpanElement>(null);
  const tipRef = useRef<HTMLDivElement>(null);
  const hideTimer = useRef<ReturnType<typeof setTimeout>>();

  const reposition = useCallback(() => {
    if (!iconRef.current) return;
    const rect = iconRef.current.getBoundingClientRect();
    const tipW = 256; // w-64 = 16rem = 256px
    const spaceBelow = window.innerHeight - rect.bottom;
    const flip = spaceBelow < 140;

    // Center horizontally on icon, clamp to viewport
    let left = rect.left + rect.width / 2 - tipW / 2;
    left = Math.max(8, Math.min(left, window.innerWidth - tipW - 8));

    const top = flip
      ? rect.top + window.scrollY - 6 // will be positioned above via bottom anchor
      : rect.bottom + window.scrollY + 6;

    setCoords({ top, left, flip });
  }, []);

  const show = useCallback(() => {
    clearTimeout(hideTimer.current);
    reposition();
    setVisible(true);
  }, [reposition]);

  const hide = useCallback(() => {
    hideTimer.current = setTimeout(() => setVisible(false), 150);
  }, []);

  // Reposition on scroll / resize while visible
  useEffect(() => {
    if (!visible) return;
    const handler = () => reposition();
    window.addEventListener('scroll', handler, true);
    window.addEventListener('resize', handler);
    return () => {
      window.removeEventListener('scroll', handler, true);
      window.removeEventListener('resize', handler);
    };
  }, [visible, reposition]);

  const tooltip = visible
    ? createPortal(
        <div
          ref={tipRef}
          onMouseEnter={show}
          onMouseLeave={hide}
          style={{
            position: 'absolute',
            top: coords.flip ? undefined : coords.top,
            bottom: coords.flip ? `${document.documentElement.scrollHeight - coords.top}px` : undefined,
            left: coords.left,
            zIndex: 9999,
          }}
          className="w-64 max-w-[calc(100vw-16px)] px-3 py-2 text-xs leading-relaxed text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg"
        >
          {content}
        </div>,
        document.body,
      )
    : null;

  return (
    <>
      <span
        ref={iconRef}
        className={`inline-flex items-center ${className}`}
        onMouseEnter={show}
        onMouseLeave={hide}
        onFocus={show}
        onBlur={hide}
        onClick={() => { if (visible) hide(); else show(); }}
        role="button"
        tabIndex={0}
        aria-label="详情"
      >
        <Info
          size={size}
          className="text-gray-400 hover:text-blue-500 dark:hover:text-blue-400 transition-colors cursor-help shrink-0"
        />
      </span>
      {tooltip}
    </>
  );
}
