import { useEffect, useState } from 'react';
import { ApiError, errorMessage, setDesktopToken } from '../../shared/api/client';
import { getIdentity, logout, type Identity } from './api';

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
        if (controller.signal.aborted) return;
        if (error instanceof ApiError && error.status === 401) {
          setSession({ status: 'signed-out' });
        } else {
          setSession((previous) => previous.status === 'ready' ? previous : { status: 'error', message: errorMessage(error) });
        }
      }
    }

    function clear() {
      current?.abort();
      setSession({ status: 'signed-out' });
    }

    function sync(event: MessageEvent) {
      if (event.data === 'signed-out') clear();
      else void refresh();
    }

    void refresh();
    window.addEventListener('focus', refresh);
    window.addEventListener('zwoa-session-expired', clear);
    window.addEventListener('zwoa-session-refresh', refresh);
    sessionEvents.addEventListener('message', sync);
    return () => {
      current?.abort();
      window.removeEventListener('focus', refresh);
      window.removeEventListener('zwoa-session-expired', clear);
      window.removeEventListener('zwoa-session-refresh', refresh);
      sessionEvents.removeEventListener('message', sync);
    };
  }, []);

  return session;
}

export async function endSession() {
  await setDesktopToken(null);
  window.dispatchEvent(new Event('zwoa-session-expired'));
  sessionEvents.postMessage('signed-out');
}

export async function signOut() {
  await logout();
  await endSession();
}

export function refreshSession() {
  window.dispatchEvent(new Event('zwoa-session-refresh'));
}

export function announceSignIn() {
  refreshSession();
  sessionEvents.postMessage('signed-in');
}
