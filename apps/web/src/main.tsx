import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { HashRouter } from 'react-router';
import { Workspace } from './Workspace';
import './workspace.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ConfigProvider locale={zhCN} theme={{ token: { colorPrimary: '#2254c5', borderRadius: 10 } }}>
      <HashRouter>
        <Workspace />
      </HashRouter>
    </ConfigProvider>
  </StrictMode>,
);
