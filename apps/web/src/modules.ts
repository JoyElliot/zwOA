import { ApartmentOutlined, AuditOutlined, TeamOutlined } from '@ant-design/icons';

export const modules = [
  {
    path: '/crm',
    title: '客户管理',
    summary: '客户、联系人和销售商机。',
    features: ['客户档案', '联系人', '销售商机'],
    Icon: TeamOutlined,
    color: 'blue',
  },
  {
    path: '/approvals',
    title: '办公审批',
    summary: '发起申请，处理待办并跟踪审批进度。',
    features: ['申请发起', '审批待办', '流程记录'],
    Icon: AuditOutlined,
    color: 'teal',
  },
  {
    path: '/organization',
    title: '组织与权限',
    summary: '部门、成员、角色和数据访问范围。',
    features: ['组织架构', '成员管理', '角色与数据权限'],
    Icon: ApartmentOutlined,
    color: 'purple',
  },
];

export type WorkspaceModule = (typeof modules)[number];
