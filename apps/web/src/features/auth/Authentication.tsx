import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, Spin } from 'antd';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { AppHeader, type ThemeControls } from '../../shared/ui/AppHeader';
import { errorMessage } from '../../shared/api/client';
import { changePassword, login } from './api';
import { announceSignIn, endSession, refreshSession, type Session } from './session';
import './auth.css';

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
      <Button onClick={refreshSession}>重试</Button>
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
      announceSignIn();
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
