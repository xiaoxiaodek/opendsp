import { useEffect, useState } from 'react';
import { Button, Table, Modal, Form, Input, Switch, message, Tag } from 'antd';
import { PlusOutlined, EditOutlined } from '@ant-design/icons';
import api from '../../services/api';

interface MediaItem {
  id: number;
  name: string;
  code: string;
  domain: string;
  status: number;
}

export default function MediaPage() {
  const [media, setMedia] = useState<MediaItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<MediaItem | null>(null);
  const [form] = Form.useForm();

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await api.get('/media');
      setMedia(res.data.media || []);
    } finally { setLoading(false); }
  };

  useEffect(() => { fetchData(); }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: MediaItem) => {
    setEditing(record);
    form.setFieldsValue({ name: record.name, domain: record.domain });
    setModalOpen(true);
  };

  const handleSubmit = async (values: { name: string; code?: string; domain: string }) => {
    if (editing) {
      await api.patch(`/media/${editing.id}`, { name: values.name, domain: values.domain });
      message.success('Updated');
    } else {
      await api.post('/media', values);
      message.success('Created');
    }
    setModalOpen(false);
    form.resetFields();
    fetchData();
  };

  const handleToggleStatus = async (id: number, currentStatus: number) => {
    await api.patch(`/media/${id}/status`, { status: currentStatus === 1 ? 2 : 1 });
    message.success('Status updated');
    fetchData();
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Code', dataIndex: 'code', key: 'code' },
    { title: 'Domain', dataIndex: 'domain', key: 'domain' },
    {
      title: 'Status', dataIndex: 'status', key: 'status',
      render: (v: number) => <Tag color={v === 1 ? 'green' : 'red'}>{v === 1 ? 'Active' : 'Disabled'}</Tag>,
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: MediaItem) => (
        <>
          <Switch size="small" checked={record.status === 1} onChange={() => handleToggleStatus(record.id, record.status)} style={{ marginRight: 8 }} />
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
        </>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Media Management</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Media</Button>
      </div>
      <Table dataSource={media} columns={columns} rowKey="id" loading={loading} />

      <Modal
        title={editing ? 'Edit Media' : 'Add Media'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          {!editing && <Form.Item name="code" label="Code" rules={[{ required: true }]}><Input /></Form.Item>}
          <Form.Item name="domain" label="Domain"><Input /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
