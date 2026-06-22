import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Select, Tabs, message, Space, Tag, InputNumber } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { listTags, createTag, deleteTag, listAudiences, createAudience, deleteAudience, createLookalike } from '../../services/api'

export default function Audience() {
  const [tags, setTags] = useState<any[]>([])
  const [audiences, setAudiences] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [tagModalOpen, setTagModalOpen] = useState(false)
  const [audienceModalOpen, setAudienceModalOpen] = useState(false)
  const [lookalikeModalOpen, setLookalikeModalOpen] = useState(false)
  const [tagForm] = Form.useForm()
  const [audienceForm] = Form.useForm()
  const [lookalikeForm] = Form.useForm()

  const fetchTags = async () => {
    setLoading(true)
    try {
      const res = await listTags(1)
      setTags(res.tags || [])
    } finally { setLoading(false) }
  }

  const fetchAudiences = async () => {
    setLoading(true)
    try {
      const res = await listAudiences(1)
      setAudiences(res.audiences || [])
    } finally { setLoading(false) }
  }

  useEffect(() => { fetchTags(); fetchAudiences() }, [])

  const handleCreateTag = async (values: any) => {
    const deviceIds = values.device_ids ? values.device_ids.split('\n').filter(Boolean) : []
    await createTag({ name: values.name, tag_type: 1, device_ids: deviceIds, device_type: 'idfa' })
    message.success('Tag created')
    setTagModalOpen(false)
    tagForm.resetFields()
    fetchTags()
  }

  const handleDeleteTag = async (id: number) => {
    await deleteTag(id)
    message.success('Tag deleted')
    fetchTags()
  }

  const handleCreateAudience = async (values: any) => {
    const rules = JSON.stringify({
      operator: 'AND',
      include: values.tag_ids.map((id: number) => ({ tag_id: id })),
    })
    await createAudience({ name: values.name, audience_type: 1, rules })
    message.success('Audience created')
    setAudienceModalOpen(false)
    audienceForm.resetFields()
    fetchAudiences()
  }

  const handleDeleteAudience = async (id: number) => {
    await deleteAudience(id)
    message.success('Audience deleted')
    fetchAudiences()
  }

  const handleCreateLookalike = async (values: any) => {
    await createLookalike({ seed_audience_id: values.seed_audience_id, name: values.name, expansion_factor: values.expansion_factor })
    message.success('Lookalike task created')
    setLookalikeModalOpen(false)
    lookalikeForm.resetFields()
    fetchTags()
  }

  const tagColumns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: 'Name', dataIndex: 'name' },
    { title: 'Type', dataIndex: 'tag_type', render: (t: number) => t === 1 ? 'Upload' : t === 2 ? 'Behavior' : 'Lookalike' },
    { title: 'Devices', dataIndex: 'device_count', render: (c: number) => c?.toLocaleString() },
    { title: 'Status', dataIndex: 'status', render: (s: number) => <Tag color={s === 2 ? 'green' : 'orange'}>{s === 2 ? 'Ready' : 'Computing'}</Tag> },
    { title: 'Action', render: (_: any, r: any) => <Button danger icon={<DeleteOutlined />} onClick={() => handleDeleteTag(r.id)} /> },
  ]

  const audienceColumns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: 'Name', dataIndex: 'name' },
    { title: 'Type', dataIndex: 'audience_type', render: (t: number) => ['Tag Combo', 'Upload', 'Behavior', 'Lookalike'][t - 1] },
    { title: 'Devices', dataIndex: 'device_count', render: (c: number) => c?.toLocaleString() },
    { title: 'Status', dataIndex: 'status', render: (s: number) => <Tag color={s === 2 ? 'green' : 'orange'}>{s === 2 ? 'Ready' : 'Computing'}</Tag> },
    { title: 'Action', render: (_: any, r: any) => <Button danger icon={<DeleteOutlined />} onClick={() => handleDeleteAudience(r.id)} /> },
  ]

  return (
    <div style={{ padding: 24 }}>
      <h2>Audience Management</h2>
      <Tabs items={[
        {
          key: 'tags', label: 'Tags',
          children: (
            <>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setTagModalOpen(true)} style={{ marginBottom: 16 }}>Create Tag</Button>
              <Table rowKey="id" columns={tagColumns} dataSource={tags} loading={loading} />
              <Modal title="Create Tag" open={tagModalOpen} onCancel={() => setTagModalOpen(false)} onOk={() => tagForm.submit()}>
                <Form form={tagForm} onFinish={handleCreateTag} layout="vertical">
                  <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
                  <Form.Item name="device_ids" label="Device IDs (one per line)"><Input.TextArea rows={6} placeholder="IDFA-001&#10;IDFA-002" /></Form.Item>
                </Form>
              </Modal>
            </>
          ),
        },
        {
          key: 'audiences', label: 'Audiences',
          children: (
            <>
              <Space style={{ marginBottom: 16 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => setAudienceModalOpen(true)}>Create Audience</Button>
              </Space>
              <Table rowKey="id" columns={audienceColumns} dataSource={audiences} loading={loading} />
              <Modal title="Create Audience" open={audienceModalOpen} onCancel={() => setAudienceModalOpen(false)} onOk={() => audienceForm.submit()}>
                <Form form={audienceForm} onFinish={handleCreateAudience} layout="vertical">
                  <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
                  <Form.Item name="tag_ids" label="Tag IDs (comma separated)" rules={[{ required: true }]}>
                    <Select mode="tags" placeholder="1,2,3" options={tags.map(t => ({ label: `${t.name} (${t.id})`, value: t.id }))} />
                  </Form.Item>
                </Form>
              </Modal>
            </>
          ),
        },
        {
          key: 'lookalike', label: 'Lookalike',
          children: (
            <>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setLookalikeModalOpen(true)} style={{ marginBottom: 16 }}>Create Lookalike</Button>
              <Modal title="Create Lookalike" open={lookalikeModalOpen} onCancel={() => setLookalikeModalOpen(false)} onOk={() => lookalikeForm.submit()}>
                <Form form={lookalikeForm} onFinish={handleCreateLookalike} layout="vertical">
                  <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
                  <Form.Item name="seed_audience_id" label="Seed Audience ID" rules={[{ required: true }]}>
                    <Select options={audiences.map(a => ({ label: `${a.name} (${a.id})`, value: a.id }))} />
                  </Form.Item>
                  <Form.Item name="expansion_factor" label="Expansion Factor" rules={[{ required: true }]}>
                    <InputNumber min={1} max={20} />
                  </Form.Item>
                </Form>
              </Modal>
            </>
          ),
        },
      ]} />
    </div>
  )
}
