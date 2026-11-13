import { ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { HashRouter } from 'react-router';
import { Workspace } from './Workspace';
import { useColorTheme } from './theme';
import './workspace.css';

function App() {
  const { colorTheme, toggleTheme } = useColorTheme();
  const dark = colorTheme === 'dark';

  return (
    <ConfigProvider locale={zhCN} theme={{
      algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm,
      token: {
        colorPrimary: dark ? '#8d9eff' : '#4057dc',
        colorBgContainer: dark ? '#20232c' : '#fdfdfe',
        colorText: dark ? '#edf0f7' : '#1c2435',
        colorTextSecondary: dark ? '#aeb6c7' : '#5b6579',
        borderRadius: 10,
      },
    }}>
      <HashRouter>
        <Workspace dark={dark} onToggleTheme={toggleTheme} />
      </HashRouter>
    </ConfigProvider>
  );
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
