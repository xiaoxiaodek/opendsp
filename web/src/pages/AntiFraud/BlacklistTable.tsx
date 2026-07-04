import React, { useEffect, useState } from 'react';
import { Table, Button, Modal, Form, Input, Select, Space, Tag, message } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { fetchBlacklist, addBlacklist, removeBlacklist } from '../../services/antifraud';
import type { BlacklistEntry } from '../../services/antifraud';

const ruleTypeColors: Record<string, string> = { ip: 'blue', device_id: 'orange', ua_pattern: 'purple', geo: 'green' };

const BlacklistTable: React.FC = () => {
  const [data, setData] = useState<BlacklistEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const loadData = async () => {
    setLoading(true);
    try {
      const result = await fetchBlacklist({ page, page_size: 20 });
      setData(result.items);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadData(); }, [page]);

  const handleAdd = async () => {
    try {
      const values = await form.validateFields();
      await addBlacklist(values);
      message.success('Blacklist entry added');
      setModalOpen(false);
      form.resetFields();
      loadData();
    } catch { /* validation or API error */ }
  };

  const handleDelete = async (id: number) => {
    await removeBlacklist(id);
    message.success('Blacklist entry removed');
    loadData();
  };

  const columns = [
    { title: 'Type', dataIndex: 'rule_type', key: 'rule_type', render: (t: string) => <Tag color={ruleTypeColors[t] || 'default'}>{t}</Tag> },
    { title: 'Value', dataIndex: 'rule_value', key: 'rule_value' },
    { title: 'Reason', dataIndex: 'reason', key: 'reason' },
    { title: 'Created', dataIndex: 'created_at', key: 'created_at' },
    {
      title: 'Action', key: 'action',
      render: (_: unknown, record: BlacklistEntry) => (
        <Button type="link" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record.id)}>Remove</Button>
      ),
    },
  ];

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>Add Entry</Button>
      </Space>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading}
        pagination={{ current: page, total, pageSize: 20, onChange: setPage }} />
      <Modal title="Add Blacklist Entry" open={modalOpen} onOk={handleAdd} onCancel={() => setModalOpen(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="rule_type" label="Type" rules={[{ required: true }]}>
            <Select options={[
              { label: 'IP Address', value: 'ip' },
              { label: 'Device ID', value: 'device_id' },
              { label: 'UA Pattern', value: 'ua_pattern' },
              { label: 'Geo', value: 'geo' },
            ]} />
          </Form.Item>
          <Form.Item name="rule_value" label="Value" rules={[{ required: true }]}>
            <Input placeholder="e.g. 192.168.1.1 or device-hash-abc123" />
          </Form.Item>
          <Form.Item name="reason" label="Reason">
            <Input placeholder="Why this entry is blacklisted" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default BlacklistTable;
