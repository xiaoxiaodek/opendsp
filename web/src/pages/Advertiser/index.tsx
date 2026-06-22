import { useEffect, useState } from 'react';
import { Button, Table, Modal, Form, Input, Select, Tag, message, Space, Descriptions, Tabs, Upload, Popconfirm, Dropdown, Image } from 'antd';
import type { MenuProps } from 'antd';
import { EyeOutlined, UploadOutlined, PlusOutlined, DeleteOutlined, CloudSyncOutlined, SyncOutlined } from '@ant-design/icons';
import {
  listAdvertisers, createAdvertiser, getAdvertiser, updateAdvertiser, deleteAdvertiser,
  auditAdvertiser,
  syncAdvertiserToPlatform,
  listProofMaterials,
  getBalance, recharge, listTransactions,
  type Advertiser, type ProofMaterial, type BalanceTransaction,
} from '../../services/api';

const SUPPORTED_PLATFORMS = [
  { key: 'iqiyi', label: 'iQiyi' },
];

export default function AdvertiserPage() {
  const [advertisers, setAdvertisers] = useState<Advertiser[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [selected, setSelected] = useState<Advertiser | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailData, setDetailData] = useState<Advertiser | null>(null);
  const [proofs, setProofs] = useState<ProofMaterial[]>([]);
  const [txs, setTxs] = useState<BalanceTransaction[]>([]);
  const [balance, setBalance] = useState<{ balance: number; creditLimit: number } | null>(null);
  const [proofUploading, setProofUploading] = useState(false);
  const [advertiserSyncingId, setAdvertiserSyncingId] = useState<string | null>(null);
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();
  const [rechargeForm] = Form.useForm();

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await listAdvertisers();
      setAdvertisers(res.data.advertisers || []);
    } finally { setLoading(false); }
  };

  useEffect(() => { fetchData(); }, []);

  const openEdit = (record: Advertiser) => {
    setSelected(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const openDetail = async (record: Advertiser) => {
    setDetailData(record);
    setDetailOpen(true);
    try {
      const [proofRes, balRes, txRes] = await Promise.all([
        listProofMaterials(record.id),
        getBalance(record.id),
        listTransactions(record.id),
      ]);
      setProofs(proofRes.data.materials || []);
      setBalance(balRes.data);
      setTxs(txRes.data.transactions || []);
    } catch {}
  };

  const handleSubmit = async (values: any) => {
    if (selected) {
      await updateAdvertiser(selected.id, values);
      message.success('Updated');
      setModalOpen(false);
      fetchData();
    }
  };

  const handleCreate = async (values: any) => {
    await createAdvertiser(values);
    message.success('Created');
    setCreateModalOpen(false);
    createForm.resetFields();
    fetchData();
  };

  const handleDelete = async (id: number) => {
    await deleteAdvertiser(id);
    message.success('Deleted');
    fetchData();
  };

  const handleAudit = async (id: number, status: number) => {
    const reason = status === 2 ? prompt('Rejection reason:') || '' : '';
    await auditAdvertiser(id, status, reason);
    message.success(status === 1 ? 'Approved' : 'Rejected');
    fetchData();
  };

  const handleRecharge = async (values: { amount: number; description: string }) => {
    if (!detailData) return;
    await recharge(detailData.id, values.amount, values.description);
    message.success('Recharged');
    const balRes = await getBalance(detailData.id);
    setBalance(balRes.data);
    const txRes = await listTransactions(detailData.id);
    setTxs(txRes.data.transactions || []);
    rechargeForm.resetFields();
  };

  const handleProofUpload = async (file: File, materialType: number) => {
    if (!detailData) return false;
    setProofUploading(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('advertiser_id', String(detailData.id));
      formData.append('material_type', String(materialType));
      const res = await fetch('/api/v1/upload/proof', {
        method: 'POST',
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
        body: formData,
      });
      const data = await res.json();
      if (data.file_id) {
        message.success('Proof uploaded');
        const proofRes = await listProofMaterials(detailData.id);
        setProofs(proofRes.data.materials || []);
      } else {
        message.error(data.error || 'Upload failed');
      }
    } catch {
      message.error('Upload error');
    } finally {
      setProofUploading(false);
    }
    return false;
  };

  const handleSyncAdvertiser = async (record: Advertiser, platform: string) => {
    setAdvertiserSyncingId(`${record.id}:${platform}`);
    try {
      await syncAdvertiserToPlatform(record.id, platform);
      message.success(`${platform} sync started`);
    } catch (err: any) {
      message.error(err?.response?.data?.error || 'Sync failed');
    } finally {
      setAdvertiserSyncingId(null);
    }
  };

  const qualStatusMap: Record<number, { color: string; text: string }> = {
    0: { color: 'default', text: 'Pending' },
    1: { color: 'green', text: 'Approved' },
    2: { color: 'red', text: 'Rejected' },
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Contact', dataIndex: 'contactName', key: 'contactName' },
    { title: 'Email', dataIndex: 'contactEmail', key: 'contactEmail' },
    { title: 'Balance', dataIndex: 'balance', key: 'balance', render: (v: number) => `¥${v?.toLocaleString()}` },
    {
      title: 'Qualification', dataIndex: 'qualificationStatus', key: 'qualificationStatus',
      render: (v: number) => <Tag color={qualStatusMap[v]?.color}>{qualStatusMap[v]?.text}</Tag>,
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: Advertiser) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />} onClick={() => openDetail(record)}>Detail</Button>
          <Button size="small" onClick={() => openEdit(record)}>Edit</Button>
          <Dropdown menu={{
            items: [
              { key: 'header', label: 'Sync to', disabled: true },
              ...SUPPORTED_PLATFORMS.map(p => ({ key: p.key, label: p.label, icon: <CloudSyncOutlined /> })),
            ],
            onClick: ({ key }) => handleSyncAdvertiser(record, key),
          }} trigger={['click']}>
            <Button size="small" icon={<SyncOutlined />} loading={!!advertiserSyncingId}>Sync</Button>
          </Dropdown>
          <Popconfirm title="Delete this advertiser?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
          {record.qualificationStatus === 0 && (
            <>
              <Button size="small" type="primary" onClick={() => handleAudit(record.id, 1)}>Approve</Button>
              <Button size="small" danger onClick={() => handleAudit(record.id, 2)}>Reject</Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  const txColumns = [
    { title: 'Time', dataIndex: 'createdAt', key: 'createdAt', render: (v: string) => new Date(v).toLocaleString() },
    { title: 'Type', dataIndex: 'txType', key: 'txType', render: (v: number) => v === 1 ? 'Recharge' : v === 2 ? 'Consume' : 'Refund' },
    { title: 'Amount', dataIndex: 'amount', key: 'amount', render: (v: number) => `¥${v?.toLocaleString()}` },
    { title: 'Before', dataIndex: 'balanceBefore', key: 'balanceBefore', render: (v: number) => `¥${v?.toLocaleString()}` },
    { title: 'After', dataIndex: 'balanceAfter', key: 'balanceAfter', render: (v: number) => `¥${v?.toLocaleString()}` },
    { title: 'Description', dataIndex: 'description', key: 'description' },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Advertisers</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>Create Advertiser</Button>
      </div>
      <Table dataSource={advertisers} columns={columns} rowKey="id" loading={loading} />

      <Modal title="Edit Advertiser" open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="Name"><Input /></Form.Item>
          <Form.Item name="industry" label="Industry"><Input /></Form.Item>
          <Form.Item name="contactName" label="Contact Name"><Input /></Form.Item>
          <Form.Item name="contactEmail" label="Contact Email"><Input /></Form.Item>
          <Form.Item name="address" label="Address"><Input /></Form.Item>
          <Form.Item name="website" label="Website"><Input /></Form.Item>
          <Form.Item name="brandNames" label="Brand Names"><Input /></Form.Item>
        </Form>
      </Modal>

      <Modal title="Create Advertiser" open={createModalOpen} onCancel={() => setCreateModalOpen(false)} onOk={() => createForm.submit()} destroyOnClose>
        <Form form={createForm} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="industry" label="Industry"><Input /></Form.Item>
          <Form.Item name="contactName" label="Contact Name"><Input /></Form.Item>
          <Form.Item name="contactEmail" label="Contact Email"><Input /></Form.Item>
          <Form.Item name="address" label="Address"><Input /></Form.Item>
          <Form.Item name="website" label="Website"><Input /></Form.Item>
          <Form.Item name="brandNames" label="Brand Names"><Input /></Form.Item>
        </Form>
      </Modal>

      <Modal title={`Advertiser Detail - ${detailData?.name}`} open={detailOpen} onCancel={() => setDetailOpen(false)} width={900} footer={null}>
        <Tabs items={[
          {
            key: 'info', label: 'Info',
            children: detailData && (
              <Descriptions column={2} bordered size="small">
                <Descriptions.Item label="ID">{detailData.id}</Descriptions.Item>
                <Descriptions.Item label="Name">{detailData.name}</Descriptions.Item>
                <Descriptions.Item label="Industry">{detailData.industry}</Descriptions.Item>
                <Descriptions.Item label="Contact">{detailData.contactName}</Descriptions.Item>
                <Descriptions.Item label="Email">{detailData.contactEmail}</Descriptions.Item>
                <Descriptions.Item label="Balance">¥{detailData.balance?.toLocaleString()}</Descriptions.Item>
                <Descriptions.Item label="Credit Limit">¥{detailData.creditLimit?.toLocaleString()}</Descriptions.Item>
                <Descriptions.Item label="Qualification">
                  <Tag color={qualStatusMap[detailData.qualificationStatus]?.color}>{qualStatusMap[detailData.qualificationStatus]?.text}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Address">{detailData.address}</Descriptions.Item>
                <Descriptions.Item label="Website">{detailData.website}</Descriptions.Item>
                <Descriptions.Item label="Brands">{detailData.brandNames}</Descriptions.Item>
              </Descriptions>
            ),
          },
          {
            key: 'proofs', label: 'Proof Materials',
            children: (
              <div>
                <Space style={{ marginBottom: 12 }}>
                  <Upload beforeUpload={(f) => handleProofUpload(f, 1)} showUploadList={false} accept="image/*,.pdf">
                    <Button icon={<UploadOutlined />} loading={proofUploading} size="small">License</Button>
                  </Upload>
                  <Upload beforeUpload={(f) => handleProofUpload(f, 2)} showUploadList={false} accept="image/*,.pdf">
                    <Button icon={<UploadOutlined />} loading={proofUploading} size="small">ID Card</Button>
                  </Upload>
                  <Upload beforeUpload={(f) => handleProofUpload(f, 3)} showUploadList={false} accept="image/*,.pdf">
                    <Button icon={<UploadOutlined />} loading={proofUploading} size="small">Tax</Button>
                  </Upload>
                  <Upload beforeUpload={(f) => handleProofUpload(f, 4)} showUploadList={false} accept="image/*,.pdf">
                    <Button icon={<UploadOutlined />} loading={proofUploading} size="small">Other</Button>
                  </Upload>
                </Space>
                <Table dataSource={proofs} columns={[
                  { title: 'Type', dataIndex: 'materialType', key: 'materialType', render: (v: number) => ['', 'License', 'ID Card', 'Tax', 'Other'][v] },
                  { title: 'Preview', dataIndex: 'fileUrl', key: 'preview', render: (url: string) => <Image src={url} width={60} style={{ objectFit: 'cover', borderRadius: 4 }} fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=" /> },
                  { title: 'File', dataIndex: 'fileName', key: 'fileName' },
                  { title: 'Status', dataIndex: 'auditStatus', key: 'auditStatus', render: (v: number) => <Tag color={v === 1 ? 'green' : v === 2 ? 'red' : 'default'}>{v === 1 ? 'Approved' : v === 2 ? 'Rejected' : 'Pending'}</Tag> },
                ]} rowKey="id" size="small" pagination={false} />
              </div>
            ),
          },
          {
            key: 'balance', label: 'Balance',
            children: (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <strong>Balance:</strong> ¥{balance?.balance?.toLocaleString() || 0} &nbsp;
                  <strong>Credit Limit:</strong> ¥{balance?.creditLimit?.toLocaleString() || 0}
                </div>
                <Form form={rechargeForm} layout="inline" onFinish={handleRecharge}>
                  <Form.Item name="amount" label="Amount" rules={[{ required: true }]}>
                    <Input type="number" min={0} prefix="¥" />
                  </Form.Item>
                  <Form.Item name="description" label="Note">
                    <Input placeholder="Recharge note" />
                  </Form.Item>
                  <Form.Item>
                    <Button type="primary" htmlType="submit">Recharge</Button>
                  </Form.Item>
                </Form>
                <Table dataSource={txs} columns={txColumns} rowKey="id" size="small" style={{ marginTop: 16 }} pagination={{ pageSize: 10 }} />
              </div>
            ),
          },
        ]} />
      </Modal>
    </div>
  );
}
