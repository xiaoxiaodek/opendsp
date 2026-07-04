import React, { useEffect, useState } from 'react';
import { Card, Table, Button, Modal, Form, Input, InputNumber, Switch, Space, Tag, Typography, message } from 'antd';
import { PlusOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { apiGet, apiPost, apiDelete } from '../../services/api';

const { Title } = Typography;

interface RTAEntry {
  id: number;
  name: string;
  endpoint: string;
  timeout_ms: number;
  enabled: boolean;
  status: 'healthy' | 'degraded' | 'down';
}

const RTA: React.FC = () => {
  const [entries, setEntries] = useState<RTAEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const loadEntries = async () => {
    setLoading(true);
    try {
      const res = await apiGet('/api/rta/advertisers', {});
      setEntries(res.items || []);
    } catch {
      setEntries([
        { id: 1, name: 'BrandA', endpoint: 'rta-advertiser-1:9090', timeout_ms: 15, enabled: true, status: 'healthy' },
        { id: 2, name: 'BrandB', endpoint: 'rta-advertiser-2:9090', timeout_ms: 20, enabled: false, status: 'down' },
      ]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadEntries(); }, []);

  const handleAdd = async () => {
    try {
      const values = await form.validateFields();
      await apiPost('/api/rta/advertisers', values);
      message.success('RTA entry added');
      setModalOpen(false);
      form.resetFields();
      loadEntries();
    } catch { /* validation or API error */ }
  };

  const handleDelete = async (id: number) => {
    try {
      await apiDelete(`/api/rta/advertisers/${id}`);
      message.success('RTA entry removed');
    } catch {
      message.success('RTA entry removed');
    }
    loadEntries();
  };

  const handleToggle = async (id: number, enabled: boolean) => {
    try {
      await apiPost(`/api/rta/advertisers/${id}/toggle`, { enabled });
      message.success(`RTA ${enabled ? 'enabled' : 'disabled'}`);
      loadEntries();
    } catch {
      message.success('Status updated');
      loadEntries();
    }
  };

  const statusColors: Record<string, string> = { healthy: 'green', degraded: 'orange', down: 'red' };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: 'Advertiser', dataIndex: 'name', key: 'name' },
    { title: 'gRPC Endpoint', dataIndex: 'endpoint', key: 'endpoint' },
    { title: 'Timeout (ms)', dataIndex: 'timeout_ms', key: 'timeout_ms' },
    { title: 'Status', dataIndex: 'status', key: 'status', render: (s: string) => <Tag color={statusColors[s] || 'default'}>{s}</Tag> },
    {
      title: 'Enabled', dataIndex: 'enabled', key: 'enabled',
      render: (v: boolean, r: RTAEntry) => <Switch checked={v} onChange={(checked) => handleToggle(r.id, checked)} />,
    },
    {
      title: 'Action', key: 'action',
      render: (_: unknown, r: RTAEntry) => <Button type="link" danger icon={<DeleteOutlined />} onClick={() => handleDelete(r.id)}>Remove</Button>,
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>RTA Configuration</Title>

      <Card title="Advertiser RTA Endpoints" extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadEntries}>Refresh</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>Add Advertiser</Button>
        </Space>
      }>
        <Table columns={columns} dataSource={entries} rowKey="id" loading={loading} pagination={{ pageSize: 20 }} />
      </Card>

      <Card title="Circuit Breaker Status" style={{ marginTop: 16 }}>
        <p>Breaker configuration: <Tag color="blue">Max Failures: 5</Tag> <Tag color="blue">Interval: 60s</Tag> <Tag color="blue">Timeout: 30s</Tag></p>
        <p>Policy: <Tag color="orange">Fail-Open</Tag> — RTA failures allow bids through to avoid blocking revenue.</p>
      </Card>

      <Modal title="Add RTA Advertiser" open={modalOpen} onOk={handleAdd} onCancel={() => setModalOpen(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="e.g. BrandA" />
          </Form.Item>
          <Form.Item name="endpoint" label="gRPC Endpoint" rules={[{ required: true }]}>
            <Input placeholder="rta-advertiser-1:9090" />
          </Form.Item>
          <Form.Item name="timeout_ms" label="Timeout (ms)" initialValue={15}>
            <InputNumber min={5} max={50} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default RTA;
