import { useEffect, useState } from 'react';
import { Button, Table, Modal, Form, Input, InputNumber, Select, Space, message, Tag, Image, Upload, Popconfirm, Tooltip, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { PlusOutlined, EditOutlined, UploadOutlined, CloudSyncOutlined, ReloadOutlined, SyncOutlined } from '@ant-design/icons';
import { listCreatives, createCreative, updateCreative, listCampaigns, listAdGroups, syncCreativeToPlatform, refreshCreativeSyncStatus, type Creative, type AdGroup, type Campaign } from '../../services/api';
import EntityDrawer from '../../components/EntityDrawer';

const creativeTypeLabels: Record<number, string> = { 1: 'Image', 2: 'Video', 3: 'Native', 4: 'Audio', 5: 'HTML' };
const auditStatusColors: Record<number, string> = { 0: 'orange', 1: 'green', 2: 'red' };
const auditStatusLabels: Record<number, string> = { 0: 'Pending', 1: 'Approved', 2: 'Rejected' };
const syncStatusColors: Record<number, string> = { 0: 'default', 1: 'processing', 2: 'orange', 3: 'green', 4: 'red' };
const syncStatusLabels: Record<number, string> = { 0: 'Not Synced', 1: 'Uploading', 2: 'Pending', 3: 'Approved', 4: 'Rejected' };
const SUPPORTED_PLATFORMS = [
  { key: 'iqiyi', label: 'iQiyi' },
  // { key: 'funshion', label: 'FengXing' },  // future
];

export default function CreativePage() {
  const [creatives, setCreatives] = useState<Creative[]>([]);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [adGroups, setAdGroups] = useState<AdGroup[]>([]);
  const [selectedCampaign, setSelectedCampaign] = useState<number | null>(null);
  const [selectedAdGroup, setSelectedAdGroup] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Creative | null>(null);
  const [uploading, setUploading] = useState(false);
type SyncStateMap = Record<number, Record<string, { status: number; reason: string }>>;

  const [syncState, setSyncState] = useState<SyncStateMap>({});
  const [syncingId, setSyncingId] = useState<string | null>(null);
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

  useEffect(() => {
    if (selectedCampaign == null) return;
    listAdGroups(selectedCampaign).then(res => {
      const list = res.data.adGroups || [];
      setAdGroups(list);
      if (list.length > 0) setSelectedAdGroup(list[0].id);
      else setSelectedAdGroup(null);
    });
  }, [selectedCampaign]);

  const fetchData = () => {
    if (selectedAdGroup == null) { setCreatives([]); return; }
    setLoading(true);
    listCreatives(selectedAdGroup).then(res => setCreatives(res.data.creatives || [])).finally(() => setLoading(false));
  };

  useEffect(() => { fetchData(); }, [selectedAdGroup]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: Creative) => {
    setEditing(record);
    form.setFieldsValue({
      name: record.name,
      creative_type: record.creativeType,
      asset_url: record.assetUrl,
      asset_width: record.assetWidth,
      asset_height: record.assetHeight,
      asset_duration: record.assetDuration,
      title: record.title,
      description: record.description,
      landing_url: record.landingUrl,
      imp_tracker: record.impTracker,
      click_tracker: record.clickTracker,
    });
    setModalOpen(true);
  };

  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const res = await fetch('/api/v1/upload/creative', {
        method: 'POST',
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
        body: formData,
      });
      const data = await res.json();
      if (data.public_path) {
        // Build file-gateway accessible URL (proxied via Vite)
        const url = data.public_path.startsWith('/') ? data.public_path : `/${data.public_path}`;
        form.setFieldsValue({ asset_url: url });
        message.success('Uploaded — URL auto-filled');
      } else {
        message.error(data.error || 'Upload failed');
      }
    } catch {
      message.error('Upload error');
    } finally {
      setUploading(false);
    }
    return false;
  };

  const handleSyncToPlatform = async (record: Creative, platform: string) => {
    setSyncingId(`${record.id}:${platform}`);
    try {
      const res = await syncCreativeToPlatform(record.id, platform);
      setSyncState(prev => ({
        ...prev,
        [record.id]: { ...prev[record.id], [platform]: { status: res.data.status, reason: res.data.reason || '' } },
      }));
      message.success(`${platform} sync: ${syncStatusLabels[res.data.status]}`);
    } catch (err: any) {
      message.error(err?.response?.data?.error || 'Sync failed');
    } finally {
      setSyncingId(null);
    }
  };

  const handleRefreshSync = async (record: Creative, platform: string) => {
    try {
      const res = await refreshCreativeSyncStatus(record.id, platform);
      setSyncState(prev => ({
        ...prev,
        [record.id]: { ...prev[record.id], [platform]: { status: res.data.status, reason: res.data.reason || '' } },
      }));
      message.success(`${platform} refreshed`);
    } catch (err: any) {
      message.error(err?.response?.data?.error || 'Refresh failed');
    }
  };

  const handleSubmit = async (values: any) => {
    const data = {
      name: values.name,
      creative_type: values.creative_type,
      asset_url: values.asset_url,
      asset_width: values.asset_width,
      asset_height: values.asset_height,
      asset_duration: values.asset_duration || 0,
      asset_mime: values.creative_type === 2 ? 'video/mp4' : 'image/jpeg',
      title: values.title,
      description: values.description,
      cta_text: values.cta_text || '',
      brand_name: values.brand_name || '',
      landing_url: values.landing_url,
      imp_tracker: values.imp_tracker || '',
      click_tracker: values.click_tracker || '',
    };

    if (editing) {
      await updateCreative(editing.id, data);
      message.success('Creative updated');
    } else {
      await createCreative({ ...data, ad_group_id: selectedAdGroup! });
      message.success('Creative created');
    }
    setModalOpen(false);
    form.resetFields();
    fetchData();
  };

  const columns = [
    { title: 'Preview', dataIndex: 'assetUrl', key: 'preview', render: (url: string) => <Image src={url} width={80} height={60} style={{ objectFit: 'cover', borderRadius: 4 }} fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=" /> },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Type', dataIndex: 'creativeType', key: 'type', render: (v: number) => creativeTypeLabels[v] || v },
    { title: 'Size', key: 'size', render: (_: any, r: Creative) => `${r.assetWidth}x${r.assetHeight}` },
    { title: 'Title', dataIndex: 'title', key: 'title' },
    {
      title: 'DSP Review', dataIndex: 'auditStatus', key: 'audit',
      render: (v: number) => <Tag color={auditStatusColors[v]}>{auditStatusLabels[v]}</Tag>,
    },
    {
      title: 'Platform Sync', key: 'sync',
      render: (_: any, record: Creative) => {
        const state = syncState[record.id];
        if (!state) return <Tag>Not Synced</Tag>;
        return (
          <Space size={4} wrap>
            {SUPPORTED_PLATFORMS.map(p => {
              const s = state[p.key];
              if (!s) return null;
              return (
                <Tooltip key={p.key} title={`${p.label}: ${syncStatusLabels[s.status]}${s.reason ? ' - ' + s.reason : ''}`}>
                  <Tag color={syncStatusColors[s.status]}>{p.label}</Tag>
                </Tooltip>
              );
            })}
          </Space>
        );
      },
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: Creative) => {
        const syncMenu: MenuProps['items'] = [
          { key: 'sync-header', label: 'Sync to', disabled: true },
          ...SUPPORTED_PLATFORMS.map(p => ({
            key: `sync-${p.key}`,
            label: p.label,
            icon: <CloudSyncOutlined />,
          })),
          { type: 'divider' },
          { key: 'refresh', label: 'Refresh all', icon: <ReloadOutlined /> },
        ];

        const onMenuClick: MenuProps['onClick'] = ({ key }) => {
          if (key === 'refresh') {
            SUPPORTED_PLATFORMS.forEach(p => handleRefreshSync(record, p.key));
            return;
          }
          const platform = key.replace('sync-', '');
          handleSyncToPlatform(record, platform);
        };

        return (
          <Space size={4} onClick={e => e.stopPropagation()}>
            <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
            <Dropdown menu={{ items: syncMenu, onClick: onMenuClick }} trigger={['click']}>
              <Button size="small" icon={<SyncOutlined />} loading={!!syncingId}>
                Sync
              </Button>
            </Dropdown>
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Space>
          <h2 style={{ margin: 0 }}>Creatives</h2>
          <Select style={{ width: 180 }} value={selectedCampaign} onChange={setSelectedCampaign}
            options={campaigns.map(c => ({ value: c.id, label: c.name }))} placeholder="Campaign" />
          <Select style={{ width: 180 }} value={selectedAdGroup} onChange={setSelectedAdGroup}
            options={adGroups.map(ag => ({ value: ag.id, label: ag.name }))} placeholder="Ad Group" />
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} disabled={!selectedAdGroup}>Create Creative</Button>
      </div>
      <Table
        dataSource={creatives}
        columns={columns}
        rowKey="id"
        loading={loading}
        onRow={(record) => ({
          onClick: () => { setSelectedEntity({ id: record.id, name: record.name }); setDrawerOpen(true); },
          style: { cursor: 'pointer' },
        })}
      />
      <Modal title={editing ? 'Edit Creative' : 'Create Creative'} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => form.submit()} width={640} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="creative_type" label="Type" rules={[{ required: true }]}>
            <Select options={[{ value: 1, label: 'Image' }, { value: 2, label: 'Video' }, { value: 3, label: 'Native' }]} />
          </Form.Item>
          <Form.Item name="asset_url" label="Asset URL" tooltip="Paste a URL or upload a file below" help="Enter URL manually, or upload a file to auto-fill this field">
            <Input placeholder="https://cdn.example.com/creative.jpg  — or upload below —" />
          </Form.Item>
          <Form.Item label="Upload File">
            <Upload beforeUpload={handleUpload} showUploadList={false} accept="image/*,video/*">
              <Button icon={<UploadOutlined />} loading={uploading}>Select File (auto-fills URL above)</Button>
            </Upload>
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="asset_width" label="Width" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item>
            <Form.Item name="asset_height" label="Height" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item>
          </Space>
          <Form.Item name="asset_duration" label="Duration (seconds)"><InputNumber min={0} /></Form.Item>
          <Form.Item name="title" label="Title"><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="landing_url" label="Landing URL" rules={[{ required: true }]}><Input placeholder="https://advertiser.com/landing" /></Form.Item>
          <Form.Item name="imp_tracker" label="Impression Tracker URL"><Input placeholder="https://tracker.example.com/imp" /></Form.Item>
          <Form.Item name="click_tracker" label="Click Tracker URL"><Input placeholder="https://tracker.example.com/click" /></Form.Item>
        </Form>
      </Modal>
      <EntityDrawer
        open={drawerOpen}
        entityType="creative"
        entityId={selectedEntity?.id || 0}
        entityName={selectedEntity?.name || ''}
        onClose={() => setDrawerOpen(false)}
      />
    </div>
  );
}
