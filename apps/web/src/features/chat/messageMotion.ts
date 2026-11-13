export function captureDraft(input: HTMLTextAreaElement | null) {
  if (!input || document.hidden || matchMedia('(prefers-reduced-motion: reduce)').matches) return null;
  const bounds = input.getBoundingClientRect();
  const style = getComputedStyle(input);
  const left = parseFloat(style.paddingLeft);
  const right = parseFloat(style.paddingRight);
  const top = parseFloat(style.paddingTop);
  const bottom = parseFloat(style.paddingBottom);
  return {
    left: bounds.left + input.clientLeft + left,
    top: bounds.top + input.clientTop + top,
    width: input.clientWidth - left - right,
    height: input.clientHeight - top - bottom,
    scrollTop: input.scrollTop,
    text: input.value,
    font: style.font,
  };
}

export function captureVisibleMessages(messages: HTMLElement | null, history: HTMLElement | null) {
  const snapshot = new Map<Element, number>();
  if (!history) return snapshot;
  const viewport = history.getBoundingClientRect();
  for (let group = messages?.lastElementChild; group; group = group.previousElementSibling) {
    const bounds = group.getBoundingClientRect();
    if (bounds.bottom <= viewport.top) break;
    if (bounds.top >= viewport.bottom) continue;
    const avatar = group.querySelector('.chat-message-avatar');
    if (avatar) snapshot.set(avatar, avatar.getBoundingClientRect().top);
    for (let row = group.querySelector('.chat-message-group-rows')?.lastElementChild; row; row = row.previousElementSibling) {
      const rowBounds = row.getBoundingClientRect();
      if (rowBounds.bottom <= viewport.top) break;
      if (rowBounds.top < viewport.bottom) snapshot.set(row, rowBounds.top);
    }
  }
  return snapshot;
}

export function animateSentMessage(source: NonNullable<ReturnType<typeof captureDraft>>, bubble: HTMLElement,
  history: HTMLElement, previousRows?: ReturnType<typeof captureVisibleMessages>) {
  const text = bubble.querySelector('.chat-bubble-text');
  const panel = history.getBoundingClientRect();
  const target = bubble.getBoundingClientRect();
  if (!text || target.bottom <= panel.top || target.bottom > panel.bottom || source.height <= 0 || source.width <= 0) return;

  const style = getComputedStyle(bubble);
  const insetX = parseFloat(style.paddingLeft);
  const insetY = parseFloat(style.paddingTop);
  const left = source.left - insetX;
  const bottom = source.top + source.height + insetY;
  const overlay = document.createElement('div');
  overlay.className = 'chat-send-overlay';
  overlay.setAttribute('aria-hidden', 'true');
  overlay.style.top = `${panel.top}px`;
  const flight = document.createElement('div');
  flight.className = 'chat-send-bubble chat-message-own';
  Object.assign(flight.style, {
    left: `${left}px`,
    top: `${bottom - panel.top}px`,
    color: style.color,
    boxShadow: style.boxShadow,
  });
  const fill = document.createElement('div');
  fill.className = 'chat-send-fill';
  fill.style.background = style.background;
  const copy = document.createElement('div');
  copy.className = 'chat-send-draft';
  copy.textContent = source.text;
  Object.assign(copy.style, {
    left: `${insetX}px`,
    top: `${insetY - source.scrollTop}px`,
    maxWidth: `${source.width}px`,
    font: source.font,
  });
  const destination = bubble.cloneNode(true) as HTMLElement;
  Object.assign(destination.style, {
    position: 'absolute',
    left: '0',
    bottom: '0',
    width: `${target.width}px`,
    maxWidth: 'none',
    background: 'none',
    boxShadow: 'none',
  });
  flight.append(fill, copy, destination);
  overlay.append(flight);
  document.body.append(overlay);

  const copyBounds = copy.getBoundingClientRect();
  const sameLine = source.text === text.textContent && source.scrollTop === 0 &&
    copyBounds.height <= parseFloat(getComputedStyle(copy).lineHeight) && text.getClientRects().length === 1;
  const draftWidth = copyBounds.width + 2 * insetX;
  const startWidth = Math.min(draftWidth, target.right - left);
  const narrowing = startWidth > target.width;
  const dx = target.left - left;
  const dy = target.bottom - bottom;
  // A quadratic arc lets the bubble move sideways first, then swing upward.
  const path: Keyframe[] = Array.from({ length: 13 }, (_, index) => {
    const t = index / 12;
    const bend = 2 * t * (1 - t);
    const horizontal = t * t + .91 * bend;
    const x = dx * horizontal;
    const y = dy * (t * t + .08 * bend);
    const frame: Keyframe = { transform: `translate(${x}px, ${y}px) translateY(-100%)` };
    // Contract wide drafts with the same progress as their horizontal movement.
    if (narrowing) frame.width = `${startWidth + (target.width - startWidth) * horizontal}px`;
    return frame;
  });
  Object.assign(path[0], {
    width: `${startWidth}px`,
    height: `${source.height + 2 * insetY}px`,
    borderRadius: '12px',
  });
  Object.assign(path[path.length - 1], {
    width: `${target.width}px`,
    height: `${target.height}px`,
    borderRadius: style.borderRadius,
  });
  const timing = { duration: 390, easing: 'cubic-bezier(.13, .66, .30, 1)', fill: 'both' } as const;
  const motion = flight.animate(path, timing);
  const layers = [
    bubble.animate([{ opacity: 0 }, { opacity: 0 }], timing),
    fill.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 55, easing: 'ease-out', fill: 'both' }),
  ];
  if (sameLine) {
    copy.hidden = true;
    const meta = destination.querySelector('.chat-bubble-meta-visible');
    if (meta) layers.push(meta.animate([{ opacity: 0 }, { opacity: 0, offset: .65 }, { opacity: 1 }], timing));
  } else {
    layers.push(
      copy.animate([{ opacity: 1 }, { opacity: 1, offset: .42 }, { opacity: 0, offset: .55 }, { opacity: 0 }], timing),
      destination.animate([{ opacity: 0 }, { opacity: 0, offset: .47 }, { opacity: 1, offset: .63 }, { opacity: 1 }], timing),
    );
  }

  // Keep scrollTop and the landing bubble at their final positions. Move the old rows visually.
  const shifts: { row: Element; offset: number }[] = [];
  for (const [row, top] of previousRows ?? []) {
    if (!row.isConnected || row.contains(bubble)) continue;
    const offset = top - row.getBoundingClientRect().top;
    if (Math.abs(offset) >= 1) shifts.push({ row, offset });
  }
  for (const { row, offset } of shifts) {
    layers.push(row.animate([{ transform: `translateY(${offset}px)` }, { transform: 'translateY(0)' }], timing));
  }

  function stop() {
    if (!overlay.isConnected) return;
    window.removeEventListener('resize', stop);
    history.removeEventListener('scroll', checkLanding);
    observer.disconnect();
    overlay.remove();
    motion.cancel();
    for (const layer of layers) layer.cancel();
  }

  function checkLanding() {
    const current = bubble.getBoundingClientRect();
    if (current.left !== target.left || current.bottom !== target.bottom ||
        current.width !== target.width || current.height !== target.height ||
        history.getBoundingClientRect().top !== panel.top) stop();
  }

  const observer = new ResizeObserver(checkLanding);
  observer.observe(history);
  observer.observe(bubble);
  history.addEventListener('scroll', checkLanding, { passive: true });
  window.addEventListener('resize', stop, { once: true });
  void motion.finished.then(stop, stop);
  return stop;
}
