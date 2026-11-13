import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import { AppHeader, type ThemeControls } from './AppHeader';
import { ApiError, errorMessage, setDesktopToken } from './api/client';
import { changePassword, getIdentity, login, logout, type Identity } from './api/organization';

type Session =
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

export function AuthLayout({ theme, children }: { theme: ThemeControls; children: React.ReactNode }) {
  return (
    <div className="auth-shell">
      <AppHeader {...theme} />
      <main className="auth-main">{children}</main>
    </div>
  );
}

export function SessionStatus({ session }: { session: Exclude<Session, { status: 'ready' }> }) {
  if (session.status === 'error') {
    return <Alert type="error" showIcon title={session.message} action={
      <Button onClick={() => window.dispatchEvent(new Event('zwoa-session-refresh'))}>重试</Button>
    } />;
  }
  return <Spin aria-label="正在验证登录状态" />;
}

export function LoginPage() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function submit(values: { username: string; password: string }) {
    setBusy(true);
    setError('');
    try {
      await login(values);
      window.dispatchEvent(new Event('zwoa-session-refresh'));
      sessionEvents.postMessage('signed-in');
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card className="auth-card">
      <div className="auth-heading">
        <span className="brand-mark" aria-hidden="true">中</span>
        <h1>登录中网联合</h1>
        <p>使用公司分配的账户登录 OA / CRM。</p>
      </div>
      {error && <Alert className="form-alert" type="error" showIcon title={error} />}
      <Form layout="vertical" requiredMark={false} onFinish={submit}>
        <Form.Item name="username" label="登录名" rules={[{ required: true, whitespace: true, message: '请输入登录名' }]}>
          <Input size="large" prefix={<UserOutlined />} autoComplete="username" maxLength={40} autoFocus />
        </Form.Item>
        <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
          <Input.Password size="large" prefix={<LockOutlined />} autoComplete="current-password" maxLength={256} />
        </Form.Item>
        <Button size="large" type="primary" htmlType="submit" block loading={busy}>登录</Button>
      </Form>
    </Card>
  );
}

export function PasswordPage({ required }: { required: boolean }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  async function submit(values: { current: string; next: string }) {
    setBusy(true);
    setError('');
    try {
      await changePassword({ current: values.current, next: values.next });
      await endSession();
      if (!required) void navigate('/', { replace: true });
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card className="auth-card">
      <div className="auth-heading">
        <h1>修改密码</h1>
        <p>{required ? '首次登录或密码被重置后，需要设置自己的密码。' : '密码更新后，请使用新密码重新登录。'}</p>
      </div>
      {error && <Alert className="form-alert" type="error" showIcon title={error} />}
      <Form layout="vertical" requiredMark={false} onFinish={submit}>
        <Form.Item name="current" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}>
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Form.Item name="next" label="新密码" rules={[{ required: true, min: 12, message: '请输入至少 12 个字符的新密码' }]}>
          <Input.Password autoComplete="new-password" maxLength={256} />
        </Form.Item>
        <Form.Item name="confirmation" label="确认新密码" dependencies={['next']} rules={[
          { required: true, message: '请再次输入新密码' },
          ({ getFieldValue }) => ({ validator: (_, value) => !value || value === getFieldValue('next') ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致')) }),
        ]}>
          <Input.Password autoComplete="new-password" maxLength={256} />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={busy}>更新密码</Button>
      </Form>
    </Card>
  );
}
