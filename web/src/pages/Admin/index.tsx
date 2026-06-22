import { useEffect, useState } from 'react';
import { Button, Table, Modal, Form, Input, Select, Tag, message, Space, Tabs } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { listUsers, updateUserRole, listPendingAudits, auditCreative, auditAdvertiser, type User, type PendingAudit } from '../../services/api';
import api from '../../services/api';

export default function AdminPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [audits, setAudits] = useState<PendingAudit[]>([]);
  const [loading, setLoading] = useState(false);
  const [roleModal, setRoleModal] = useState(false);
  const [createModal, setCreateModal] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await listUsers();
      setUsers(res.data.users || []);
    } finally { setLoading(false); }
  };

  const fetchAudits = async () => {
    try {
      const res = await listPendingAudits();
      setAudits(res.data.audits || []);
    } catch {}
  };

  useEffect(() => { fetchUsers(); fetchAudits(); }, []);

  const openRoleEdit = (user: User) => {
    setSelectedUser(user);
    form.setFieldsValue({ role: user.role });
    setRoleModal(true);
  };

  const handleRoleUpdate = async (values: { role: string }) => {
    if (selectedUser) {
      await updateUserRole(selectedUser.id, values.role);
      message.success('Role updated');
      setRoleModal(false);
      fetchUsers();
    }
  };

  const handleCreateUser = async (values: { email: string; password: string; name: string; role: string; advertiser_id?: number }) => {
    await api.post('/admin/users', values);
    message.success('User created');
    setCreateModal(false);
    createForm.resetFields();
    fetchUsers();
  };

  const handleAudit = async (record: PendingAudit, status: number) => {
    const reason = status === 2 ? prompt('Rejection reason:') || '' : '';
    if (record.auditType === 1) {
      await auditCreative(record.id, status, reason);
    } else if (record.auditType === 2) {
      await auditAdvertiser(record.id, status, reason);
    }
    message.success(status === 1 ? 'Approved' : 'Rejected');
    fetchAudits();
  };

  const userColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: 'Email', dataIndex: 'email', key: 'email' },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Role', dataIndex: 'role', key: 'role', render: (v: string) => <Tag color={v === 'admin' ? 'blue' : v === 'operator' ? 'green' : 'default'}>{v}</Tag> },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: User) => (
        <Button size="small" onClick={() => openRoleEdit(record)}>Change Role</Button>
      ),
    },
  ];

  const auditColumns = [
    { title: 'Type', dataIndex: 'auditType', key: 'auditType', render: (v: number) => v === 1 ? 'Creative' : 'Advertiser' },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Advertiser', dataIndex: 'advertiserName', key: 'advertiserName' },
    { title: 'Status', key: 'status', render: () => <Tag color="orange">Pending</Tag> },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: PendingAudit) => (
        <Space>
          <Button size="small" type="primary" onClick={() => handleAudit(record, 1)}>Approve</Button>
          <Button size="small" danger onClick={() => handleAudit(record, 2)}>Reject</Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <h2>Admin Panel</h2>
      <Tabs items={[
        {
          key: 'users', label: 'User Management',
          children: (
            <>
              <div style={{ marginBottom: 16 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModal(true)}>Create User</Button>
              </div>
              <Table dataSource={users} columns={userColumns} rowKey="id" loading={loading} />
            </>
          ),
        },
        {
          key: 'audits', label: 'Audit Queue',
          children: <Table dataSource={audits} columns={auditColumns} rowKey="id" />,
        },
      ]} />

      <Modal title="Change Role" open={roleModal} onCancel={() => setRoleModal(false)} onOk={() => form.submit()}>
        <Form form={form} layout="vertical" onFinish={handleRoleUpdate}>
          <Form.Item name="role" label="Role" rules={[{ required: true }]}>
            <Select options={[
              { value: 'admin', label: 'Admin' },
              { value: 'operator', label: 'Operator' },
              { value: 'viewer', label: 'Viewer' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="Create User" open={createModal} onCancel={() => setCreateModal(false)} onOk={() => createForm.submit()} destroyOnClose>
        <Form form={createForm} layout="vertical" onFinish={handleCreateUser}>
          <Form.Item name="email" label="Email" rules={[{ required: true, type: 'email' }]}><Input /></Form.Item>
          <Form.Item name="password" label="Password" rules={[{ required: true, min: 6 }]}><Input.Password /></Form.Item>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="role" label="Role" rules={[{ required: true }]} initialValue="viewer">
            <Select options={[
              { value: 'admin', label: 'Admin' },
              { value: 'operator', label: 'Operator' },
              { value: 'viewer', label: 'Viewer' },
            ]} />
          </Form.Item>
          <Form.Item name="advertiser_id" label="Advertiser ID"><Input type="number" /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
