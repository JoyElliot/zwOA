import { useEffect, useState } from 'react';
import { ApiError, errorMessage, sessionChangedEvent } from '../../shared/api/client';
import { getIdentity, type Identity } from './api';

export type Session =
  | { status: 'loading' | 'signed-out' }
  | { status: 'error'; message: string }
  | { status: 'ready'; identity: Identity };

const sessionEvents = new BroadcastChannel('zwoa-session');

export function useSession() {
  const [session, setSession] = useState<Session>({ status: 'loading' });

  useEffect(() => {
    let current: AbortController | undefined;

    async function refresh() {
      current?.abort();
      const controller = new AbortController();
      current = controller;
      try {
        const identity = await getIdentity(controller.signal);
        if (!controller.signal.aborted) setSession({ status: 'ready', identity });
      } catch (error) {
        if (controller.signal.aborted || (error instanceof DOMException && error.name === 'AbortError')) return;
        if (error instanceof ApiError && error.status === 401) {
          setSession({ status: 'signed-out' });
        } else {
          setSession((previous) => previous.status === 'ready' ? previous : { status: 'error', message: errorMessage(error) });
        }
      }
    }

    function notifyWindows() {
      void refresh();
      sessionEvents.postMessage('refresh');
    }

    void refresh();
    window.addEventListener('focus', refresh);
    window.addEventListener(sessionChangedEvent, notifyWindows);
    window.addEventListener('zwoa-session-refresh', refresh);
    sessionEvents.addEventListener('message', refresh);
    return () => {
      current?.abort();
      window.removeEventListener('focus', refresh);
      window.removeEventListener(sessionChangedEvent, notifyWindows);
      window.removeEventListener('zwoa-session-refresh', refresh);
      sessionEvents.removeEventListener('message', refresh);
    };
  }, []);

  return session;
}

export function refreshSession() {
  window.dispatchEvent(new Event('zwoa-session-refresh'));
}

export function announceSignIn() {
  window.dispatchEvent(new Event(sessionChangedEvent));
}
