import { MoonOutlined, SunOutlined } from '@ant-design/icons';
import type { ReactNode } from 'react';
import { DesktopWindowControls, TitlebarDragRegion } from './DesktopTitlebar';

export interface ThemeControls { dark: boolean; onToggleTheme: () => void }

export function Brand({ caption }: { caption?: string }) {
  return (
    <div className="brand">
      <span className="brand-mark" aria-hidden="true">中</span>
      <span className="brand-name"><strong>中网联合</strong>{caption && <small>{caption}</small>}</span>
    </div>
  );
}

export function ThemeToggle({ dark, onToggleTheme }: ThemeControls) {
  const label = dark ? '切换为浅色' : '切换为深色';
  return (
    <button className="theme-toggle" type="button" title={label} aria-label={label} onClick={onToggleTheme}>
      {dark ? <SunOutlined /> : <MoonOutlined />}
    </button>
  );
}

export function AppHeader({ dark, onToggleTheme, caption = 'OA / CRM', children }: ThemeControls & {
  caption?: string; children?: ReactNode;
}) {
  return (
    <header className="workspace-header">
      <Brand caption={caption} />
      <TitlebarDragRegion />
      <div className="header-actions">
        {children}
        <ThemeToggle dark={dark} onToggleTheme={onToggleTheme} />
      </div>
      <DesktopWindowControls />
    </header>
  );
}
