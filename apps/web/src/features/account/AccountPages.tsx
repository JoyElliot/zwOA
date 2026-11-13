import { RightOutlined } from '@ant-design/icons';
import { Card, Descriptions, Switch, Tag } from 'antd';
import { Link } from 'react-router';
import type { ThemeControls } from '../../shared/ui/AppHeader';
import type { Identity } from '../auth/api';
import { PasswordPage } from '../auth/Authentication';
import { ProfileAvatar } from './ProfileAvatar';
import './account.css';

export function ProfilePage({ identity }: { identity: Identity }) {
  return (
    <section className="account-page" aria-labelledby="profile-title">
      <h1 id="profile-title">个人主页</h1>
      <Card className="account-panel">
        <ProfileAvatar identity={identity} />
        <Descriptions column={1} items={[
          { key: 'username', label: '登录名', children: identity.username },
          { key: 'role', label: '账户类型', children: identity.superAdmin ? '超级管理员' : '成员' },
        ]} />
        <Link className="account-setting-row account-setting-link profile-password-link" to="/account/password">
          <span>修改密码</span>
          <RightOutlined aria-hidden="true" />
        </Link>
      </Card>
      <Card className="account-panel" title="团队与项目">
        {identity.teams.length ? (
          <ul className="profile-teams">
            {identity.teams.map((team) => (
              <li key={team.id}>
                <span className="profile-team-name">{team.name}</span>
                <div className="profile-team-labels">
                  <Tag>{team.kind === 'project' ? '项目' : '团队'}</Tag>
                  {team.admin && <Tag color="blue">管理员</Tag>}
                </div>
              </li>
            ))}
          </ul>
        ) : <p className="account-muted">尚未加入团队或项目。</p>}
      </Card>
    </section>
  );
}

export function SettingsPage({ dark, onToggleTheme }: ThemeControls) {
  return (
    <section className="account-page" aria-labelledby="settings-title">
      <h1 id="settings-title">设置</h1>
      <Card className="account-panel" title="外观">
        <div className="account-setting-row">
          <span id="theme-setting-label">深色模式</span>
          <Switch checked={dark} onChange={onToggleTheme} aria-labelledby="theme-setting-label" />
        </div>
      </Card>
    </section>
  );
}

export function AccountPasswordPage() {
  return (
    <div className="account-page account-password">
      <Link className="back-link" to="/account/profile">返回个人主页</Link>
      <PasswordPage required={false} />
    </div>
  );
}
