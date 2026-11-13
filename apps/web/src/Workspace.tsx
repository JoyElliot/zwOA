import { ArrowRightOutlined, DashboardOutlined, MoonOutlined, SunOutlined } from '@ant-design/icons';
import { Badge, Button, Card } from 'antd';
import { useEffect, useState } from 'react';
import { Link, Navigate, NavLink, Route, Routes } from 'react-router';
import { getWorkbenchOverview, type WorkbenchOverview } from './api/workbench';
import { modules, type WorkspaceModule } from './modules';

type Connection =
  | { state: 'loading' | 'offline' }
  | { state: 'online'; overview: WorkbenchOverview };

const connectionLabels = {
  loading: { status: 'processing', label: '正在连接' },
  online: { status: 'success', label: '服务在线' },
  offline: { status: 'error', label: '服务未连接' },
} as const;

export function Workspace({ dark, onToggleTheme }: { dark: boolean; onToggleTheme: () => void }) {
  const [connection, setConnection] = useState<Connection>({ state: 'loading' });
  const [attempt, setAttempt] = useState(0);

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
    <div className="shell">
      <header className="workspace-header">
        <Link className="brand" to="/">
          <span className="brand-mark" aria-hidden="true">中</span>
          <span><strong>中网联合</strong><small>OA / CRM</small></span>
        </Link>
        <div className="header-spacer" />
        <div className="header-actions">
          <div className="service-status" role="status">
            <Badge status={connectionLabel.status} text={connectionLabel.label} />
            {connection.state === 'offline' && (
              <Button size="small" onClick={() => {
                setConnection({ state: 'loading' });
                setAttempt((current) => current + 1);
              }}>重试</Button>
            )}
          </div>
          <button
            className="theme-toggle"
            type="button"
            title={dark ? '切换为浅色' : '切换为深色'}
            aria-label={dark ? '切换为浅色' : '切换为深色'}
            onClick={onToggleTheme}
          >
            {dark ? <SunOutlined /> : <MoonOutlined />}
          </button>
        </div>
      </header>
      <aside className="sidebar">
        <nav className="navigation" aria-label="业务导航">
          <NavLink to="/" end><DashboardOutlined /> 工作台</NavLink>
          {modules.map(({ path, title, Icon }) => (
            <NavLink key={path} to={path}><Icon /> {title}</NavLink>
          ))}
        </nav>
      </aside>
      <main className="workspace-main">
        <Routes>
          <Route index element={<Workbench connection={connection} />} />
          {modules.map((module) => (
            <Route key={module.path} path={module.path} element={<ModulePage module={module} />} />
          ))}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}

function Workbench({ connection }: { connection: Connection }) {
  return (
    <>
      <section className="welcome">
        <h1>工作台</h1>
        <p>从这里进入客户管理、办公审批和组织管理。</p>
      </section>
      <div className="module-grid">
        {modules.map(({ path, title, summary, Icon, color }) => (
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
