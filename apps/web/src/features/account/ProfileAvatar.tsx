import { CameraOutlined } from '@ant-design/icons';
import { Alert, Button, Modal, Upload } from 'antd';
import { useState } from 'react';
import { errorMessage, request, sessionChangedEvent } from '../../shared/api/client';
import { UserAvatar } from '../../shared/ui/UserAvatar';
import type { Identity } from '../auth/api';

const acceptedImages = ['image/jpeg', 'image/png', 'image/webp'];

async function prepareAvatar(file: File) {
  const bitmap = await createImageBitmap(file);
  try {
    const canvas = document.createElement('canvas');
    canvas.width = canvas.height = 256;
    const context = canvas.getContext('2d');
    if (!context) throw new Error('无法处理图片。');
    const crop = Math.min(bitmap.width, bitmap.height);
    context.fillStyle = '#fff';
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.drawImage(bitmap, (bitmap.width - crop) / 2, (bitmap.height - crop) / 2, crop, crop,
      0, 0, canvas.width, canvas.height);
    return canvas.toDataURL('image/jpeg', 0.88);
  } finally {
    bitmap.close();
  }
}

export function ProfileAvatar({ identity }: { identity: Identity }) {
  const [open, setOpen] = useState(false);
  const [preview, setPreview] = useState('');
  const [error, setError] = useState('');
  const [processing, setProcessing] = useState(false);
  const [saving, setSaving] = useState(false);
  const busy = processing || saving;

  async function choose(file: File) {
    setPreview('');
    setError('');
    if (!acceptedImages.includes(file.type) || file.size > 8 * 1024 * 1024) {
      setError('请选择不超过 8 MB 的 JPG、PNG 或 WebP 图片。');
      return;
    }
    setProcessing(true);
    try {
      setPreview(await prepareAvatar(file));
    } catch {
      setError('无法读取这张图片，请重新选择。');
    } finally {
      setProcessing(false);
    }
  }

  async function save() {
    if (busy || !preview) return;
    setSaving(true);
    setError('');
    try {
      await request('/auth/avatar', { method: 'PUT', body: { image: preview.split(',')[1] } });
      window.dispatchEvent(new Event(sessionChangedEvent));
      setOpen(false);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <div className="profile-heading">
        <UserAvatar name={identity.name} version={identity.avatarVersion} size={52} className="profile-avatar" generic />
        <div className="profile-heading-details">
          <h2>{identity.name}</h2>
          <Button size="small" type="link" className="profile-avatar-edit" icon={<CameraOutlined />} onClick={() => {
            setPreview('');
            setError('');
            setOpen(true);
          }}>更换头像</Button>
        </div>
      </div>
      {open && <Modal title="更换头像" open onCancel={() => setOpen(false)} onOk={() => void save()}
        okText="保存" confirmLoading={saving} okButtonProps={{ disabled: !preview || processing }}
        closable={!busy} keyboard={!busy} mask={{ closable: !busy }} cancelButtonProps={{ disabled: busy }}>
        {error && <Alert type="error" showIcon title={error} />}
        <div className="profile-avatar-preview">
          <UserAvatar name={identity.name} version={identity.avatarVersion} preview={preview || undefined}
            size={96} className="profile-avatar" generic />
          <Upload accept={acceptedImages.join(',')} showUploadList={false} disabled={busy} beforeUpload={(file) => {
            void choose(file);
            return Upload.LIST_IGNORE;
          }}>
            <Button disabled={busy} loading={processing}>选择图片</Button>
          </Upload>
        </div>
      </Modal>}
    </>
  );
}
