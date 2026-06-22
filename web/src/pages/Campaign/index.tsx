import { useEffect, useState } from 'react';
import { Button, Table, Modal, Form, Input, InputNumber, DatePicker, Select, Switch, message, Space, Tag } from 'antd';
import { PlusOutlined, EditOutlined } from '@ant-design/icons';
import { listCampaigns, createCampaign, updateCampaign, updateCampaignStatus } from '../../services/api';
import type { Campaign } from '../../services/api';
import dayjs from 'dayjs';
import EntityDrawer from '../../components/EntityDrawer';

export default function CampaignPage() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Campaign | null>(null);
  const [form] = Form.useForm();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedEntity, setSelectedEntity] = useState<{id: number; name: string} | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await listCampaigns(1);
      setCampaigns(res.data.campaigns || []);
    } finally { setLoading(false); }
  };

  useEffect(() => { fetchData(); }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: Campaign) => {
    setEditing(record);
    form.setFieldsValue({
      name: record.name,
      budget: record.budget,
      daily_budget: record.dailyBudget,
      time_range: record.startTime && record.endTime ? [dayjs(record.startTime), dayjs(record.endTime)] : undefined,
      pacing: record.pacing,
    });
    setModalOpen(true);
  };

  const handleSubmit = async (values: any) => {
    const data = {
      name: values.name,
      budget: values.budget,
      daily_budget: values.daily_budget,
      start_time: values.time_range?.[0]?.toISOString(),
      end_time: values.time_range?.[1]?.toISOString(),
      pacing: values.pacing || 1,
    };

    if (editing) {
      await updateCampaign(editing.id, data);
      message.success('Campaign updated');
    } else {
      await createCampaign({ ...data, advertiser_id: 1 });
      message.success('Campaign created');
    }
    setModalOpen(false);
    form.resetFields();
    fetchData();
  };

  const handleToggleStatus = async (id: number, currentStatus: number) => {
    await updateCampaignStatus(id, currentStatus === 1 ? 2 : 1);
    message.success('Status updated');
    fetchData();
  };

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Budget', dataIndex: 'budget', key: 'budget', render: (v: number) => v ? `¥${v.toLocaleString()}` : '-' },
    { title: 'Daily Budget', dataIndex: 'dailyBudget', key: 'dailyBudget', render: (v: number) => v ? `¥${v.toLocaleString()}` : '-' },
    {
      title: 'Time Range', key: 'time',
      render: (_: any, r: Campaign) => {
        if (!r.startTime && !r.endTime) return '-';
        const s = r.startTime ? dayjs(r.startTime).format('YYYY-MM-DD') : '...';
        const e = r.endTime ? dayjs(r.endTime).format('YYYY-MM-DD') : '...';
        return `${s} ~ ${e}`;
      },
    },
    {
      title: 'Status', dataIndex: 'status', key: 'status',
      render: (v: number) => <Tag color={v === 1 ? 'green' : v === 2 ? 'orange' : 'default'}>{v === 1 ? 'Active' : v === 2 ? 'Paused' : 'Ended'}</Tag>,
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: Campaign) => (
        <Space onClick={e => e.stopPropagation()}>
          <Switch size="small" checked={record.status === 1} onChange={() => handleToggleStatus(record.id, record.status)} />
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Campaigns</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Create Campaign</Button>
      </div>
      <Table
        dataSource={campaigns}
        columns={columns}
        rowKey="id"
        loading={loading}
        onRow={(record) => ({
          onClick: () => { setSelectedEntity({ id: record.id, name: record.name }); setDrawerOpen(true); },
          style: { cursor: 'pointer' },
        })}
      />
      <Modal title={editing ? 'Edit Campaign' : 'Create Campaign'} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="budget" label="Budget"><InputNumber style={{ width: '100%' }} min={0} prefix="¥" /></Form.Item>
          <Form.Item name="daily_budget" label="Daily Budget"><InputNumber style={{ width: '100%' }} min={0} prefix="¥" /></Form.Item>
          <Form.Item name="time_range" label="Time Range"><DatePicker.RangePicker showTime style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="pacing" label="Pacing"><Select options={[{ value: 1, label: 'Standard' }, { value: 2, label: 'Accelerated' }]} /></Form.Item>
        </Form>
      </Modal>
      <EntityDrawer
        open={drawerOpen}
        entityType="campaign"
        entityId={selectedEntity?.id || 0}
        entityName={selectedEntity?.name || ''}
        onClose={() => setDrawerOpen(false)}
      />
    </div>
  );
}
