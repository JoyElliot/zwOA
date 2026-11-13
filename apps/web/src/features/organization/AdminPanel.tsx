import { ApartmentOutlined, AppstoreOutlined, IdcardOutlined, PlusOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Checkbox, Form, Input, Modal, Popconfirm, Select, Space, Spin, Table, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { Navigate, NavLink, Route, Routes } from 'react-router';
import { AppHeader, type ThemeControls } from '../../shared/ui/AppHeader';
import { ApiError, errorMessage } from '../../shared/api/client';
import {
  deleteRecord, getAdministration, saveModuleAdmins, saveRecord,
  type AccountInput, type Administration, type ManagedAccount, type ManagedModule, type PermissionGroup, type Team,
} from './api';
import './admin.css';

const sections = [
  { key: 'accounts', title: '账户', Icon: IdcardOutlined },
  { key: 'groups', title: '权限组', Icon: SafetyCertificateOutlined },
  { key: 'teams', title: '团队与项目', Icon: ApartmentOutlined },
  { key: 'modules', title: '模块管理员', Icon: AppstoreOutlined },
] as const;

type Editor =
  | { kind: 'accounts'; record?: ManagedAccount }
  | { kind: 'groups'; record?: PermissionGroup }
  | { kind: 'teams'; record?: Team }
  | { kind: 'modules'; record: ManagedModule };

export default function AdminPanel({ theme }: { theme: ThemeControls }) {
  const [data, setData] = useState<Administration | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [editor, setEditor] = useState<Editor | null>(null);
  const [revision, setRevision] = useState(0);
  const [loadedRevision, setLoadedRevision] = useState(-1);

  useEffect(() => {
    const controller = new AbortController();
    getAdministration(controller.signal).then((result) => {
      if (!controller.signal.aborted) {
        setData(result);
        setLoadedRevision(revision);
        setError('');
      }
    }).catch((cause: unknown) => {
      if (!controller.signal.aborted) setError(errorMessage(cause));
    });
    return () => controller.abort();
  }, [revision]);

  function refresh() {
    setError('');
    setRevision((current) => current + 1);
  }

  async function remove(kind: 'groups' | 'teams', record: PermissionGroup | Team) {
    if (!data) return;
    setNotice('');
    try {
      await deleteRecord(kind, record.id, data.revision);
      setNotice(`已删除「${record.name}」`);
      refresh();
    } catch (cause) {
      setError(errorMessage(cause));
    }
  }

  const accounts = new Map(data?.accounts.map((account) => [account.id, account]));
  const groupNames = new Map(data?.groups.map((group) => [group.id, group.name]));
  const permissions = new Map(data?.modules.map((module) => [module.permission, module.name]));
  const loading = revision !== loadedRevision && !error;

  function accountNames(ids: number[]) {
    return ids.map((id) => {
      const account = accounts.get(id);
      return account ? `${account.name}${account.active ? '' : '（停用）'}` : '';
    }).filter(Boolean).join('、') || '—';
  }

  function create(kind: 'accounts' | 'groups' | 'teams') {
    setNotice('');
    setEditor({ kind });
  }

  return (
    <div className="shell admin-shell">
      <AppHeader {...theme} caption="管理面板" />
      <aside className="sidebar admin-sidebar">
        <nav className="navigation" aria-label="管理导航">
          {sections.map(({ key, title, Icon }) => <NavLink key={key} to={`/admin/${key}`} aria-label={title} title={title}>
            <Icon /><span className="admin-nav-label">{title}</span>
          </NavLink>)}
        </nav>
      </aside>
      <main className="workspace-main admin-main">
        {error && <Alert className="form-alert" type="error" title={error} showIcon action={<Button size="small" onClick={refresh}>重试</Button>} />}
        {notice && <Alert className="form-alert" type="success" title={notice} showIcon closable onClose={() => setNotice('')} />}
        {!data && !error && <Spin aria-label="正在加载管理面板" />}
        {data && <Routes>
          <Route path="accounts" element={
            <AdminSection title="账户" description="创建公司账户，分配权限组，或停用离职账户。" action="添加账户" onAdd={() => create('accounts')}>
              <Table rowKey="id" dataSource={data.accounts} loading={loading} scroll={{ x: 640 }} columns={[
                { title: '姓名', dataIndex: 'name' },
                { title: '登录名', dataIndex: 'username' },
                { title: '权限组', render: (_, account) => account.superAdmin ? <Tag color="blue">超级管理员</Tag> : account.groupIds.map((id) => groupNames.get(id)).join('、') || '—' },
                { title: '状态', render: (_, account) => account.active ? <Tag color="green">启用</Tag> : <Tag>停用</Tag> },
                { title: '操作', render: (_, account) => !account.superAdmin && <Button type="link" onClick={() => setEditor({ kind: 'accounts', record: account })}>编辑</Button> },
              ]} />
            </AdminSection>
          } />
          <Route path="groups" element={
            <AdminSection title="权限组" description="组合功能模块的使用权限。账户可以加入多个权限组。" action="创建权限组" onAdd={() => create('groups')}>
              <Table rowKey="id" dataSource={data.groups} loading={loading} scroll={{ x: 590 }} columns={[
                { title: '名称', dataIndex: 'name' },
                { title: '描述', dataIndex: 'description', render: (description: string) => description || '—' },
                { title: '模块权限', render: (_, group) => group.permissions.map((key) => permissions.get(key)).join('、') || '—' },
                { title: '操作', render: (_, group) => <Space>
                  <Button type="link" onClick={() => setEditor({ kind: 'groups', record: group })}>编辑</Button>
                  <Popconfirm title="删除这个权限组？" description="该权限组授予账户的权限将被移除。" onConfirm={() => remove('groups', group)}>
                    <Button type="link" danger>删除</Button>
                  </Popconfirm>
                </Space> },
              ]} />
            </AdminSection>
          } />
          <Route path="teams" element={
            <AdminSection title="团队与项目" description="设置团队或项目的成员，并从成员中指定管理员。" action="创建团队或项目" onAdd={() => create('teams')}>
              <Table rowKey="id" dataSource={data.teams} loading={loading} scroll={{ x: 640 }} columns={[
                { title: '名称', dataIndex: 'name' },
                { title: '类型', render: (_, team) => team.kind === 'project' ? '项目' : '团队' },
                { title: '成员', render: (_, team) => team.memberIds.length },
                { title: '管理员', render: (_, team) => accountNames(team.adminIds) },
                { title: '操作', render: (_, team) => <Space>
                  <Button type="link" onClick={() => setEditor({ kind: 'teams', record: team })}>编辑</Button>
                  <Popconfirm title="删除这个团队或项目？" description="其成员关系和管理员设置也会移除。" onConfirm={() => remove('teams', team)}>
                    <Button type="link" danger>删除</Button>
                  </Popconfirm>
                </Space> },
              ]} />
            </AdminSection>
          } />
          <Route path="modules" element={
            <AdminSection title="模块管理员" description="为每个功能模块指定跨团队和项目的管理员。">
              <Table rowKey="key" dataSource={data.modules} loading={loading} pagination={false} scroll={{ x: 440 }} columns={[
                { title: '功能模块', dataIndex: 'name' },
                { title: '管理员', render: (_, module) => accountNames(module.adminIds) },
                { title: '操作', render: (_, module) => <Button type="link" onClick={() => setEditor({ kind: 'modules', record: module })}>设置管理员</Button> },
              ]} />
            </AdminSection>
          } />
          <Route path="*" element={<Navigate to="/admin/accounts" replace />} />
        </Routes>}
      </main>
      {data && editor && <AdminEditor data={data} editor={editor} onClose={() => setEditor(null)} onRefresh={() => {
        setEditor(null);
        refresh();
      }} onSaved={() => {
        setEditor(null);
        setNotice('设置已保存');
        refresh();
      }} />}
    </div>
  );
}

function AdminSection({ title, description, action, onAdd, children }: {
  title: string; description: string; action?: string; onAdd?: () => void; children: React.ReactNode;
}) {
  return <>
    <section className="admin-heading">
      <div><h1>{title}</h1><p>{description}</p></div>
      {action && <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>{action}</Button>}
    </section>
    <Card className="admin-table">{children}</Card>
  </>;
}

function AdminEditor({ editor, data, onClose, onRefresh, onSaved }: { editor: Editor; data: Administration; onClose: () => void; onRefresh: () => void; onSaved: () => void }) {
  const [form] = Form.useForm();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [stale, setStale] = useState(false);
  const [revision] = useState(data.revision);
  const members = (Form.useWatch('memberIds', form) as number[] | undefined) ?? [];
  const accountOptions = data.accounts.map((account) => ({ label: `${account.name}（${account.username}${account.active ? '' : '，已停用'}）`, value: account.id }));
  const title = editor.kind === 'modules' ? `设置${editor.record.name}管理员` :
    `${editor.record ? '编辑' : '创建'}${editor.kind === 'accounts' ? '账户' : editor.kind === 'groups' ? '权限组' : '团队或项目'}`;
  const initialValues = editor.record ?? (editor.kind === 'accounts' ? { active: true, groupIds: [] } :
    editor.kind === 'groups' ? { description: '', permissions: [] } : { kind: 'team', description: '', memberIds: [], adminIds: [] });
  const nameRule = { required: true, whitespace: true, message: '请输入名称' };

  async function submit(values: AccountInput & PermissionGroup & Team) {
    setBusy(true);
    setError('');
    try {
      switch (editor.kind) {
        case 'accounts':
          await saveRecord('accounts', editor.record?.id, revision, { username: values.username, name: values.name, password: values.password, active: values.active, groupIds: values.groupIds ?? [] });
          break;
        case 'groups':
          await saveRecord('groups', editor.record?.id, revision, { name: values.name, description: values.description ?? '', permissions: values.permissions ?? [] });
          break;
        case 'teams':
          await saveRecord('teams', editor.record?.id, revision, { name: values.name, kind: values.kind, description: values.description ?? '', memberIds: values.memberIds ?? [], adminIds: values.adminIds ?? [] });
          break;
        case 'modules':
          await saveModuleAdmins(editor.record.key, revision, values.adminIds ?? []);
          break;
      }
      onSaved();
    } catch (cause) {
      setError(errorMessage(cause));
      setStale(cause instanceof ApiError && cause.status === 412);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal title={title} open onCancel={busy ? undefined : onClose} onOk={() => form.submit()} confirmLoading={busy} closable={!busy} mask={{ closable: !busy }} keyboard={!busy} cancelButtonProps={{ disabled: busy }} okButtonProps={{ disabled: stale }}>
      {error && <Alert className="form-alert" type="error" title={error} showIcon action={stale && <Button size="small" onClick={onRefresh}>刷新数据</Button>} />}
      <Form form={form} layout="vertical" initialValues={initialValues} onFinish={submit} disabled={busy}>
        {editor.kind === 'accounts' && <>
          <Form.Item name="username" label="登录名" rules={[{ required: true, pattern: /^[a-z0-9][a-z0-9._-]{2,39}$/i, message: '请输入 3–40 位字母、数字或 . _ -，以字母或数字开头' }]}>
            <Input autoComplete="off" maxLength={40} disabled={!!editor.record} />
          </Form.Item>
          <Form.Item name="name" label="姓名" rules={[nameRule]}><Input maxLength={80} /></Form.Item>
          <Form.Item name="password" label={editor.record ? '重置密码' : '密码'} extra={editor.record ? '留空保留当前密码。重置后，账户需用新密码重新登录。' : undefined}
            rules={[
              { required: !editor.record, message: '请输入密码' },
              { validator: (_, value?: string) => !value || new TextEncoder().encode(value).length <= 256
                ? Promise.resolve()
                : Promise.reject(new Error('密码不能超过 256 字节')) },
            ]}>
            <Input.Password autoComplete="new-password" maxLength={256} />
          </Form.Item>
          <Form.Item name="groupIds" label="权限组"><Select mode="multiple" options={data.groups.map(({ name, id }) => ({ label: name, value: id }))} /></Form.Item>
          <Form.Item name="active" valuePropName="checked"><Checkbox>启用账户</Checkbox></Form.Item>
        </>}
        {(editor.kind === 'groups' || editor.kind === 'teams') && <>
          <Form.Item name="name" label="名称" rules={[nameRule]}><Input maxLength={80} /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea maxLength={300} rows={2} /></Form.Item>
        </>}
        {editor.kind === 'groups' && <Form.Item name="permissions" label="允许使用的模块">
          <Checkbox.Group options={data.modules.map(({ name, permission }) => ({ label: name, value: permission }))} />
        </Form.Item>}
        {editor.kind === 'teams' && <>
          <Form.Item name="kind" label="类型" rules={[{ required: true }]}>
            <Select options={[{ label: '团队', value: 'team' }, { label: '项目', value: 'project' }]} />
          </Form.Item>
          <Form.Item name="memberIds" label="成员">
            <Select mode="multiple" options={accountOptions} showSearch={{ optionFilterProp: 'label' }} onChange={(ids: number[]) => {
              const admins = (form.getFieldValue('adminIds') ?? []) as number[];
              form.setFieldValue('adminIds', admins.filter((id) => ids.includes(id)));
            }} />
          </Form.Item>
          <Form.Item name="adminIds" label="团队管理员" extra="管理员从该团队或项目的成员中选择。">
            <Select mode="multiple" options={accountOptions.filter(({ value }) => members.includes(value))} showSearch={{ optionFilterProp: 'label' }} />
          </Form.Item>
        </>}
        {editor.kind === 'modules' && <Form.Item name="adminIds" label="模块管理员">
          <Select mode="multiple" options={accountOptions} showSearch={{ optionFilterProp: 'label' }} />
        </Form.Item>}
      </Form>
    </Modal>
  );
}
