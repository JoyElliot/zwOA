import { Alert, Form, Input, Modal, Segmented, Select } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { errorMessage } from '../api/client';
import { createConversation, findContacts, newRequestKey, type Contact } from './api';

export function CreateConversation({ onClose, onCreate }: { onClose: () => void; onCreate: (id: number) => void }) {
  const [form] = Form.useForm();
  const [kind, setKind] = useState<'direct' | 'group'>('direct');
  const [query, setQuery] = useState('');
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [chosen, setChosen] = useState<{ value: number; label: React.ReactNode }[]>([]);
  const [searching, setSearching] = useState(true);
  const [searchError, setSearchError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const lastRequest = useRef<{ payload: string; key: string } | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    setSearching(true);
    setSearchError('');
    const timer = window.setTimeout(() => {
      findContacts(query, controller.signal).then((values) => {
        if (!controller.signal.aborted) setContacts(values);
      }).catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setContacts([]);
          setSearchError(errorMessage(cause));
        }
      }).finally(() => {
        if (!controller.signal.aborted) setSearching(false);
      });
    }, 180);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [query]);

  async function submit(values: { title?: string; members: typeof chosen | (typeof chosen)[number] }) {
    const members = Array.isArray(values.members) ? values.members : [values.members];
    const title = kind === 'group' ? values.title?.trim() ?? '' : '';
    const memberIds = members.map(({ value }) => value);
    const payload = JSON.stringify([title, memberIds]);
    if (lastRequest.current?.payload !== payload) lastRequest.current = { payload, key: newRequestKey() };
    setBusy(true);
    setError('');
    try {
      const { id } = await createConversation({ title, memberIds, requestKey: lastRequest.current.key });
      onCreate(id);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  const options = new Map(chosen.map((item) => [item.value, item]));
  for (const { id, name, username } of contacts) options.set(id, { value: id, label: `${name}（${username}）` });

  return (
    <Modal title="发起会话" open onCancel={busy ? undefined : onClose} onOk={() => form.submit()}
      okText="创建" confirmLoading={busy} closable={!busy} keyboard={!busy} mask={{ closable: !busy }} cancelButtonProps={{ disabled: busy }}>
      <Form form={form} layout="vertical" requiredMark={false} onFinish={submit} disabled={busy}>
        {error && <Alert className="form-alert" type="error" showIcon title={error} />}
        <Form.Item label="会话类型">
          <Segmented<'direct' | 'group'> value={kind} options={[{ label: '单聊', value: 'direct' }, { label: '群聊', value: 'group' }]} onChange={(value) => {
            setKind(value);
            form.resetFields(['members', 'title']);
            setChosen([]);
            setQuery('');
          }} />
        </Form.Item>
        {kind === 'group' && <Form.Item name="title" label="群名称" rules={[{ required: true, whitespace: true, message: '请输入群名称' }]}>
          <Input maxLength={80} />
        </Form.Item>}
        <Form.Item name="members" label={kind === 'group' ? '群成员' : '联系人'} rules={[{ required: true, message: '请选择联系人' }]}
          help={searchError || undefined} validateStatus={searchError ? 'error' : undefined}>
          <Select mode={kind === 'group' ? 'multiple' : undefined} labelInValue allowClear placement="topLeft"
            showSearch={{ filterOption: false, onSearch: setQuery }}
            placeholder="搜索姓名或登录名" loading={searching} options={[...options.values()]}
            notFoundContent={searching ? '正在查找…' : '没有符合条件的联系人'}
            maxCount={kind === 'group' ? 99 : undefined}
            onChange={(value: typeof chosen | (typeof chosen)[number] | undefined) => {
              setChosen(Array.isArray(value) ? value : value ? [value] : []);
              setQuery('');
            }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
