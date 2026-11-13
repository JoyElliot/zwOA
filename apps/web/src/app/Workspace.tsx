import { DashboardOutlined, SettingOutlined } from '@ant-design/icons';
import { Alert, Button, Spin } from 'antd';
import { invoke } from '@tauri-apps/api/core';
import { lazy, Suspense, useEffect, useState } from 'react';
import { Link, Navigate, NavLink, Route, Routes, useMatch } from 'react-router';
import { Brand, type ThemeControls } from '../shared/ui/AppHeader';
import { WorkspaceTitlebar } from '../shared/ui/DesktopTitlebar';
import { AccountMenu } from '../features/account/AccountMenu';
import { AccountPasswordPage, ProfilePage, SettingsPage } from '../features/account/AccountPages';
import { logout, type Identity } from '../features/auth/api';
import { desktop, desktopSessionScope, errorMessage } from '../shared/api/client';
import { getWorkbenchOverview, type WorkbenchOverview } from '../features/workbench/api';
import { ModulePage, Workbench } from '../features/workbench/Workbench';
import { modules } from '../features/workbench/modules';

const ChatPage = lazy(() => import('../features/chat/ChatPage'));

type Connection =
  | { state: 'loading' | 'offline' }
  | { state: 'online'; overview: WorkbenchOverview };

export function Workspace({ theme, identity }: { theme: ThemeControls; identity: Identity }) {
  const [connection, setConnection] = useState<Connection>({ state: 'loading' });
  const [attempt, setAttempt] = useState(0);
  const [actionError, setActionError] = useState('');
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

  return (
    <div className="shell workspace-shell">
      <aside className="sidebar workspace-sidebar">
        <Brand />
        <nav className="navigation" aria-label="业务导航">
          <NavLink to="/" end aria-label="工作台" title="工作台"><DashboardOutlined /><span className="workspace-nav-label">工作台</span></NavLink>
          {availableModules.map(({ path, title, Icon }) => (
            <NavLink key={path} to={path} aria-label={title} title={title}><Icon /><span className="workspace-nav-label">{title}</span></NavLink>
          ))}
          {identity.superAdmin && (desktop ? (
            <button type="button" aria-label="管理面板" title="管理面板"
              onClick={() => void invoke('open_admin', { scope: desktopSessionScope }).catch(() => setActionError('无法打开管理窗口，请重试。'))}>
              <SettingOutlined /><span className="workspace-nav-label">管理面板</span>
            </button>
          ) : (
            <Link to="/admin/accounts" target="_blank" rel="noopener" aria-label="管理面板" title="管理面板">
              <SettingOutlined /><span className="workspace-nav-label">管理面板</span>
            </Link>
          ))}
        </nav>
        <div className="workspace-sidebar-footer">
          <AccountMenu name={identity.name} username={identity.username} avatarVersion={identity.avatarVersion} {...theme}
            onLogout={() => void logout().catch((error: unknown) => setActionError(errorMessage(error)))} />
        </div>
      </aside>
      <div className="workspace-body">
        {!chatOpen && <WorkspaceTitlebar />}
        {actionError && <Alert className="workspace-action-error" type="error" showIcon title={actionError} closable onClose={() => setActionError('')} />}
        <main className={chatOpen ? 'workspace-main workspace-main-chat' : 'workspace-main'}>
          <Routes>
            <Route index element={<>
              {connection.state === 'offline' && <Alert className="form-alert" type="warning" showIcon title="服务未连接" action={
                <Button size="small" onClick={() => {
                  setConnection({ state: 'loading' });
                  setAttempt((current) => current + 1);
                }}>重试</Button>
              } />}
              <Workbench overview={connection.state === 'online' ? connection.overview : undefined} availableModules={availableModules} />
            </>} />
            <Route path="/account/profile" element={<ProfilePage identity={identity} />} />
            <Route path="/account/settings" element={<SettingsPage {...theme} />} />
            <Route path="/account/password" element={<AccountPasswordPage />} />
            {availableModules.map((module) => module.key === 'chat' ? (
              <Route key={module.path} path="/chat/:conversationId?" element={
                <Suspense fallback={<>
                  <WorkspaceTitlebar />
                  <div className="workspace-chat-loading"><Spin aria-label="正在打开消息" /></div>
                </>}>
                  <ChatPage key={identity.id} actor={identity.id} />
                </Suspense>
              } />
            ) : <Route key={module.path} path={module.path} element={<ModulePage module={module} />} />)}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>
      </div>
    </div>
  );
}
