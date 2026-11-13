import { ArrowRightOutlined } from '@ant-design/icons';
import { Card } from 'antd';
import { Link } from 'react-router';
import type { WorkspaceModule } from './modules';
import type { WorkbenchOverview } from './api';
import './workbench.css';

export function Workbench({ overview, availableModules }: { overview: WorkbenchOverview | undefined; availableModules: WorkspaceModule[] }) {
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
      {overview && (
        <p className="service-version">{overview.name} · v{overview.version}</p>
      )}
    </>
  );
}

export function ModulePage({ module }: { module: WorkspaceModule }) {
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
