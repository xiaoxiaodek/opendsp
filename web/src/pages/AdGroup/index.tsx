import { useEffect, useState } from 'react';
import { Button, Table, Modal, Form, Input, InputNumber, Select, Switch, message, Space, Tag } from 'antd';
import { PlusOutlined, EditOutlined } from '@ant-design/icons';
import { listAdGroups, createAdGroup, updateAdGroup, updateAdGroupStatus, listCampaigns, type AdGroup, type Campaign } from '../../services/api';
import EntityDrawer from '../../components/EntityDrawer';

const bidTypeLabels: Record<number, string> = { 1: 'CPM', 2: 'CPC', 3: 'CPV', 4: 'CPA' };

export default function AdGroupPage() {
  const [adGroups, setAdGroups] = useState<AdGroup[]>([]);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [selectedCampaign, setSelectedCampaign] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<AdGroup | null>(null);
  const [form] = Form.useForm();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedEntity, setSelectedEntity] = useState<{id: number; name: string} | null>(null);

  useEffect(() => {
    listCampaigns(1).then(res => {
      const list = res.data.campaigns || [];
      setCampaigns(list);
      if (list.length > 0) setSelectedCampaign(list[0].id);
    });
  }, []);

  const fetchData = () => {
    if (selectedCampaign == null) return;
    setLoading(true);
    listAdGroups(selectedCampaign).then(res => setAdGroups(res.data.adGroups || [])).finally(() => setLoading(false));
  };

  useEffect(() => { fetchData(); }, [selectedCampaign]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ campaign_id: selectedCampaign });
    setModalOpen(true);
  };

  const openEdit = (record: AdGroup) => {
    setEditing(record);
    form.setFieldsValue({
      campaign_id: record.campaignId,
      name: record.name,
      bid_type: record.bidType,
      bid_price: record.bidPrice,
      daily_budget: record.dailyBudget,
      freq_cap: record.freqCap,
      targeting: record.targeting,
    });
    setModalOpen(true);
  };

  const handleSubmit = async (values: any) => {
    let targeting = values.targeting;
    if (typeof targeting === 'string') {
      try { targeting = JSON.parse(targeting); } catch { targeting = {}; }
    }
    const data = {
      campaign_id: values.campaign_id,
      name: values.name,
      bid_type: values.bid_type,
      bid_price: values.bid_price,
      daily_budget: values.daily_budget,
      freq_cap: values.freq_cap,
      targeting: JSON.stringify(targeting),
    };

    if (editing) {
      await updateAdGroup(editing.id, data);
      message.success('Ad Group updated');
    } else {
      await createAdGroup(data);
      message.success('Ad Group created');
    }
    setModalOpen(false);
    form.resetFields();
    fetchData();
  };

  const handleToggleStatus = async (id: number, currentStatus: number) => {
    await updateAdGroupStatus(id, currentStatus === 1 ? 2 : 1);
    message.success('Status updated');
    fetchData();
  };

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Bid Type', dataIndex: 'bidType', key: 'bidType', render: (v: number) => bidTypeLabels[v] || v },
    { title: 'Bid Price', dataIndex: 'bidPrice', key: 'bidPrice', render: (v: number) => `¥${v}` },
    { title: 'Daily Budget', dataIndex: 'dailyBudget', key: 'dailyBudget', render: (v: number) => v ? `¥${v.toLocaleString()}` : '-' },
    { title: 'Freq Cap', dataIndex: 'freqCap', key: 'freqCap', render: (v: number) => v || '-' },
    {
      title: 'Status', dataIndex: 'status', key: 'status',
      render: (v: number) => {
        const map: Record<number, { text: string; color: string }> = {
          0: { text: 'Draft', color: 'default' },
          1: { text: 'Active', color: 'green' },
          2: { text: 'Paused', color: 'orange' },
          3: { text: 'Completed', color: 'blue' },
        };
        const s = map[v] ?? { text: 'Unknown', color: 'default' };
        return <Tag color={s.color}>{s.text}</Tag>;
      },
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: AdGroup) => (
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
        <Space>
          <h2 style={{ margin: 0 }}>Ad Groups</h2>
          <Select style={{ width: 200 }} value={selectedCampaign} onChange={setSelectedCampaign}
            options={campaigns.map(c => ({ value: c.id, label: c.name }))} placeholder="Select Campaign" />
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} disabled={!selectedCampaign}>Create Ad Group</Button>
      </div>
      <Table
        dataSource={adGroups}
        columns={columns}
        rowKey="id"
        loading={loading}
        onRow={(record) => ({
          onClick: () => { setSelectedEntity({ id: record.id, name: record.name }); setDrawerOpen(true); },
          style: { cursor: 'pointer' },
        })}
      />
      <Modal title={editing ? 'Edit Ad Group' : 'Create Ad Group'} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => form.submit()} width={640} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="campaign_id" label="Campaign" rules={[{ required: true }]}>
            <Select options={campaigns.map(c => ({ value: c.id, label: c.name }))} placeholder="Select Campaign" />
          </Form.Item>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="bid_type" label="Bid Type" rules={[{ required: true }]}>
            <Select options={[{ value: 1, label: 'CPM' }, { value: 2, label: 'CPC' }, { value: 3, label: 'CPV' }, { value: 4, label: 'CPA' }]} />
          </Form.Item>
          <Form.Item name="bid_price" label="Bid Price" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} min={0} step={0.01} prefix="¥" /></Form.Item>
          <Form.Item name="daily_budget" label="Daily Budget"><InputNumber style={{ width: '100%' }} min={0} prefix="¥" /></Form.Item>
          <Form.Item name="freq_cap" label="Frequency Cap"><InputNumber style={{ width: '100%' }} min={0} /></Form.Item>
          <Form.Item name="targeting" label="Targeting (JSON)">
            <Input.TextArea rows={6} placeholder='{"geo":{"city":["110000"]},"device":{"os":["ios","android"]}}' />
          </Form.Item>
        </Form>
      </Modal>
      <EntityDrawer
        open={drawerOpen}
        entityType="adgroup"
        entityId={selectedEntity?.id || 0}
        entityName={selectedEntity?.name || ''}
        onClose={() => setDrawerOpen(false)}
      />
    </div>
  );
}
