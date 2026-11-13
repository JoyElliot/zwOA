import { MoonOutlined, SunOutlined } from '@ant-design/icons';
import type { ReactNode } from 'react';
import { DesktopWindowControls, TitlebarDragRegion } from './DesktopTitlebar';

export interface ThemeControls { dark: boolean; onToggleTheme: () => void }

export function AppHeader({ dark, onToggleTheme, caption = 'OA / CRM', children }: ThemeControls & {
  caption?: string; children?: ReactNode;
}) {
  return (
    <header className="workspace-header">
      <div className="brand">
        <span className="brand-mark" aria-hidden="true">中</span>
        <span><strong>中网联合</strong><small>{caption}</small></span>
      </div>
      <TitlebarDragRegion />
      <div className="header-actions">
        {children}
        <button className="theme-toggle" type="button"
          title={dark ? '切换为浅色' : '切换为深色'}
          aria-label={dark ? '切换为浅色' : '切换为深色'} onClick={onToggleTheme}>
          {dark ? <SunOutlined /> : <MoonOutlined />}
        </button>
      </div>
      <DesktopWindowControls />
    </header>
  );
}
