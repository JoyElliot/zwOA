import { ArrowRightOutlined, DashboardOutlined, SettingOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, Badge, Button, Card, Dropdown, Spin } from 'antd';
import { invoke } from '@tauri-apps/api/core';
import { lazy, Suspense, useEffect, useState } from 'react';
import { Link, Navigate, NavLink, Route, Routes, useMatch, useNavigate } from 'react-router';
import { AppHeader, type ThemeControls } from './AppHeader';
import { signOut } from './Authentication';
import { desktop, errorMessage } from './api/client';
import type { Identity } from './api/organization';
import { getWorkbenchOverview, type WorkbenchOverview } from './api/workbench';
import { modules, type WorkspaceModule } from './modules';

const ChatPage = lazy(() => import('./chat/ChatPage'));

type Connection =
  | { state: 'loading' | 'offline' }
  | { state: 'online'; overview: WorkbenchOverview };

const connectionLabels = {
  loading: { status: 'processing', label: '正在连接' },
  online: { status: 'success', label: '服务在线' },
  offline: { status: 'error', label: '服务未连接' },
} as const;

export function Workspace({ theme, identity }: { theme: ThemeControls; identity: Identity }) {
  const [connection, setConnection] = useState<Connection>({ state: 'loading' });
  const [attempt, setAttempt] = useState(0);
  const [actionError, setActionError] = useState('');
  const navigate = useNavigate();
  const chatRoute = useMatch('/chat/*');
  const availableModules = modules.filter(({ key }) => identity.superAdmin ||
    identity.permissions.includes(`${key}.access`) || identity.adminModules.includes(key));
  const chatOpen = !!chatRoute && availableModules.some(({ key }) => key === 'chat');

  useEffect(() => {
    const controller = new AbortController();
    getWorkbenchOverview(controller.signal)
      .then((overview) => {
        if (!controller.signal.aborted) setConnection({ state: 'online', overview });
      })
      .catch(() => {
        if (!controller.signal.aborted) setConnection({ state: 'offline' });
      });
    return () => controller.abort();
  }, [attempt]);

  const connectionLabel = connectionLabels[connection.state];

  return (
    <div className={chatOpen ? 'shell chat-shell' : 'shell'}>
      <AppHeader {...theme}>
          <div className="service-status" role="status">
            <Badge status={connectionLabel.status} text={connectionLabel.label} />
            {connection.state === 'offline' && (
              <Button size="small" onClick={() => {
                setConnection({ state: 'loading' });
                setAttempt((current) => current + 1);
              }}>重试</Button>
            )}
          </div>
          <Dropdown trigger={['click']} menu={{ items: [
            { key: 'password', label: '修改密码' },
            { key: 'logout', label: '退出登录' },
          ], onClick: ({ key }) => {
            if (key === 'password') void navigate('/account/password');
            else void signOut().catch((error: unknown) => setActionError(errorMessage(error)));
          } }}>
            <Button type="text" icon={<UserOutlined />} className="account-trigger" title={identity.name}>{identity.name}</Button>
          </Dropdown>
      </AppHeader>
      <aside className="sidebar">
        <nav className="navigation" aria-label="业务导航">
          <NavLink to="/" end><DashboardOutlined /> 工作台</NavLink>
          {availableModules.map(({ path, title, Icon }) => (
            <NavLink key={path} to={path}><Icon /> {title}</NavLink>
          ))}
          {identity.superAdmin && (desktop ? (
            <button type="button" onClick={() => void invoke('open_admin').catch(() => setActionError('无法打开管理窗口，请重试。'))}>
              <SettingOutlined /> 管理面板
            </button>
          ) : (
            <Link to="/admin/accounts" target="_blank" rel="noopener"><SettingOutlined /> 管理面板</Link>
          ))}
        </nav>
      </aside>
      <main className={chatOpen ? 'workspace-main workspace-main-chat' : 'workspace-main'}>
        {actionError && <Alert className="form-alert" type="error" showIcon title={actionError} closable onClose={() => setActionError('')} />}
        <Routes>
          <Route index element={<Workbench connection={connection} availableModules={availableModules} />} />
          {availableModules.map((module) => module.key === 'chat' ? (
            <Route key={module.path} path="/chat/:conversationId?" element={
              <Suspense fallback={<div className="workspace-chat-loading"><Spin aria-label="正在打开消息" /></div>}>
                <ChatPage key={identity.id} actor={identity.id} />
              </Suspense>
            } />
          ) : <Route key={module.path} path={module.path} element={<ModulePage module={module} />} />)}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}

function Workbench({ connection, availableModules }: { connection: Connection; availableModules: WorkspaceModule[] }) {
  return (
    <>
      <section className="welcome">
        <h1>工作台</h1>
        <p>{availableModules.length > 0 ? '选择一个功能模块。' : '当前账户没有可用的功能模块，请联系管理员分配权限。'}</p>
      </section>
      <div className="module-grid">
        {availableModules.map(({ path, title, summary, Icon, color }) => (
          <Link className="module-link" key={path} to={path}>
            <Card className="module-card">
              <span className={`module-icon ${color}`} aria-hidden="true"><Icon /></span>
              <h2>{title}</h2>
              <p>{summary}</p>
              <span className="module-action">查看模块 <ArrowRightOutlined /></span>
            </Card>
          </Link>
        ))}
      </div>
      {connection.state === 'online' && (
        <p className="service-version">{connection.overview.name} · v{connection.overview.version}</p>
      )}
    </>
  );
}

function ModulePage({ module }: { module: WorkspaceModule }) {
  const { Icon } = module;
  return (
    <>
      <Link className="back-link" to="/">返回工作台</Link>
      <section className="module-heading">
        <span className={`module-icon ${module.color}`} aria-hidden="true"><Icon /></span>
        <div><h1>{module.title}</h1><p>{module.summary}</p></div>
      </section>
      <Card className="scope-card" title="待建设功能">
        <ul>{module.features.map((feature) => <li key={feature}>{feature}</li>)}</ul>
      </Card>
    </>
  );
}
