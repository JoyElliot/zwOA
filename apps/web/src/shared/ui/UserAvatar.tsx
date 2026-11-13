import { TeamOutlined, UserOutlined } from '@ant-design/icons';
import { Avatar } from 'antd';
import { useEffect, useState } from 'react';
import { request } from '../api/client';

// 同一头像可能同时出现在侧栏和多条消息中，只请求一次，并限制缓存大小。
const images = new Map<string, Promise<string>>();

function avatarImage(version: string) {
  const cached = images.get(version);
  if (cached) return cached;
  const pending = request<{ image: string }>(`/auth/avatars/${encodeURIComponent(version)}`)
    .then(({ image }) => `data:image/jpeg;base64,${image}`)
    .catch((error: unknown) => {
      if (images.get(version) === pending) images.delete(version);
      throw error;
    });
  images.set(version, pending);
  if (images.size > 128) {
    const oldest = images.keys().next().value;
    if (oldest !== undefined) images.delete(oldest);
  }
  return pending;
}

export function UserAvatar({ name, version, preview, size, className, group, generic }: {
  name: string;
  version?: string;
  preview?: string;
  size: number;
  className?: string;
  group?: boolean;
  generic?: boolean;
}) {
  const [loaded, setLoaded] = useState<{ version: string; src: string }>();

  useEffect(() => {
    if (!version || preview) return;
    let current = true;
    void avatarImage(version).then((src) => {
      if (current) setLoaded({ version, src });
    }).catch(() => { /* 加载失败时显示默认头像。 */ });
    return () => { current = false; };
  }, [preview, version]);

  const src = preview ?? (loaded?.version === version ? loaded?.src : undefined);
  const initial = Array.from(name.trim())[0]?.toUpperCase();
  const icon = group ? <TeamOutlined /> : generic || !initial ? <UserOutlined /> : undefined;
  return (
    <Avatar shape="circle" size={size} className={className} aria-hidden="true" icon={icon}
      src={src}>
      {!icon && initial}
    </Avatar>
  );
}
