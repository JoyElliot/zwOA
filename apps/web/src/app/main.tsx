import { Alert, Button, ConfigProvider, Spin, theme } from 'antd';
import { getCurrentWindow } from '@tauri-apps/api/window';
import zhCN from 'antd/locale/zh_CN';
import { lazy, StrictMode, Suspense } from 'react';
import { createRoot } from 'react-dom/client';
import { HashRouter, Link, Navigate, Route, Routes } from 'react-router';
import { AuthLayout, LoginPage, PasswordPage, SessionStatus } from '../features/auth/Authentication';
import { useSession } from '../features/auth/session';
import { desktop } from '../shared/api/client';
import type { ThemeControls } from '../shared/ui/AppHeader';
import { Workspace } from './Workspace';
import { useColorTheme } from './theme';
import './styles.css';

function App() {
  const { colorTheme, toggleTheme } = useColorTheme();
  const dark = colorTheme === 'dark';

  return (
    <ConfigProvider locale={zhCN} button={{ autoInsertSpace: false }} theme={{
      algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm,
      token: {
        colorPrimary: dark ? '#5eb2ff' : '#006bd6',
        colorBgContainer: dark ? '#20232c' : '#fdfdfe',
        colorText: dark ? '#edf0f7' : '#1c2435',
        colorTextSecondary: dark ? '#aeb6c7' : '#5b6579',
        borderRadius: 10,
      },
    }}>
      <HashRouter>
        <AppRoutes theme={{ dark, onToggleTheme: toggleTheme }} />
      </HashRouter>
    </ConfigProvider>
  );
}

const adminWindow = desktop && getCurrentWindow().label === 'admin';
const AdminPanel = lazy(() => import('../features/organization/AdminPanel'));

function AppRoutes({ theme }: { theme: ThemeControls }) {
  const session = useSession();
  if (session.status !== 'ready') {
    return <AuthLayout theme={theme}>{session.status === 'signed-out' ? (
      adminWindow ? <Alert showIcon type="info" title="请在工作台重新登录" /> : <LoginPage />
    ) : <SessionStatus session={session} />}</AuthLayout>;
  }

  const { identity } = session;
  if (identity.mustChangePassword) {
    return <AuthLayout theme={theme}>{adminWindow ? <Alert showIcon type="info" title="请在工作台修改密码" /> : <PasswordPage required />}</AuthLayout>;
  }

  return <Routes>
    <Route path="/admin/*" element={identity.superAdmin && (!desktop || adminWindow) ? (
      <Suspense fallback={<AuthLayout theme={theme}><Spin aria-label="正在打开管理面板" /></AuthLayout>}>
        <AdminPanel theme={theme} />
      </Suspense>
    ) : (
      <AuthLayout theme={theme}>
        <Alert showIcon type="warning" title={identity.superAdmin ? '请从工作台打开管理面板' : '此账户无权访问管理面板'}
          action={!adminWindow && <Link to="/"><Button>返回工作台</Button></Link>} />
      </AuthLayout>
    )} />
    <Route path="/*" element={adminWindow ? <Navigate to="/admin/accounts" replace /> : <Workspace theme={theme} identity={identity} />} />
  </Routes>;
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
