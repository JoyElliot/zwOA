import { LogoutOutlined, MoonOutlined, SettingOutlined, SunOutlined, UserOutlined } from '@ant-design/icons';
import { Button, Dropdown } from 'antd';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import type { ThemeControls } from '../../shared/ui/AppHeader';
import { UserAvatar } from '../../shared/ui/UserAvatar';
import './account.css';

export function AccountMenu({ name, username, avatarVersion, onLogout, dark, onToggleTheme }: ThemeControls & {
  name: string;
  username: string;
  avatarVersion: string;
  onLogout: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [keyboardOpened, setKeyboardOpened] = useState(false);
  const navigate = useNavigate();

  return (
    <Dropdown
      trigger={['click']}
      placement="topLeft"
      align={{ offset: [0, -16] }}
      getPopupContainer={(trigger) => trigger.parentElement ?? document.body}
      autoFocus={keyboardOpened}
      open={open}
      onOpenChange={setOpen}
      classNames={{ root: 'account-dropdown', menu: 'account-menu', item: 'account-menu-item' }}
      menu={{
        items: [
          { key: '/account/profile', label: '个人主页', icon: <UserOutlined /> },
          { key: '/account/settings', label: '设置', icon: <SettingOutlined /> },
          { key: 'theme', label: dark ? '切换为浅色' : '切换为深色', icon: dark ? <SunOutlined /> : <MoonOutlined /> },
          { type: 'divider' },
          { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, className: 'account-menu-logout' },
        ],
        onClick: ({ key }) => {
          setOpen(false);
          if (key === 'logout') onLogout();
          else if (key === 'theme') onToggleTheme();
          else void navigate(key);
        },
      }}
    >
      <Button type="text" className="account-trigger"
        aria-label={name} aria-haspopup="menu" aria-expanded={open} title={name}
        onClickCapture={(event) => setKeyboardOpened(event.detail === 0)}>
        <UserAvatar name={name} version={avatarVersion} className="account-trigger-avatar" size={34} generic />
        <span className="account-trigger-details">
          <strong className="account-name">{name}</strong>
          <span className="account-login-name">{username}</span>
        </span>
      </Button>
    </Dropdown>
  );
}
