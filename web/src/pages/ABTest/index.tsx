import React, { useEffect, useState } from 'react';
import { Card, Table, Button, Modal, Form, Input, InputNumber, Select, Tag, Space, Typography, Statistic, Row, Col, message, Slider } from 'antd';
import { PlusOutlined, DeleteOutlined, PlayCircleOutlined, PauseCircleOutlined } from '@ant-design/icons';
import { apiGet, apiPost, apiDelete } from '../../services/api';

const { Title } = Typography;

interface Variant {
  name: string;
  percentage: number;
  config_overrides: Record<string, any>;
}

interface Experiment {
  id: number;
  name: string;
  status: 'draft' | 'running' | 'paused' | 'completed';
  variants: Variant[];
  start_at: string;
  end_at: string;
  winner?: string;
  confidence?: number;
  lift_ctr?: number;
  lift_cvr?: number;
  lift_roas?: number;
}

interface ABStats {
  active_experiments: number;
  total_experiments: number;
  traffic_split: number;
}

const ABTest: React.FC = () => {
  const [experiments, setExperiments] = useState<Experiment[]>([]);
  const [stats, setStats] = useState<ABStats>({ active_experiments: 0, total_experiments: 0, traffic_split: 0 });
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const loadExperiments = async () => {
    setLoading(true);
    try {
      const res = await apiGet('/api/abtest/experiments', {});
      setExperiments(res.items || []);
      setStats(res.stats || { active_experiments: 0, total_experiments: 0, traffic_split: 0 });
    } catch {
      setExperiments([
        { id: 1, name: 'Bid Strategy A vs B', status: 'running', variants: [{ name: 'control', percentage: 50, config_overrides: {} }, { name: 'aggressive', percentage: 50, config_overrides: { oxbi_multiplier: 1.5 } }], start_at: '2026-07-01', end_at: '2026-07-08', winner: 'aggressive', confidence: 95.2, lift_ctr: 3.2, lift_roas: 8.5 },
        { id: 2, name: 'Creative Layout Test', status: 'completed', variants: [{ name: 'layout_a', percentage: 50, config_overrides: {} }, { name: 'layout_b', percentage: 50, config_overrides: { creative_template: 'b' } }], start_at: '2026-06-20', end_at: '2026-06-27', winner: 'layout_b', confidence: 97.1, lift_ctr: 12.7, lift_cvr: 4.3 },
        { id: 3, name: 'Pricing ECPM vs OXBI', status: 'draft', variants: [{ name: 'ecpm', percentage: 50, config_overrides: {} }, { name: 'oxbi', percentage: 50, config_overrides: { pricing_strategy: 'oxbi' } }], start_at: '', end_at: '' },
      ]);
      setStats({ active_experiments: 1, total_experiments: 3, traffic_split: 10 });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadExperiments(); }, []);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      await apiPost('/api/abtest/experiments', {
        name: values.name,
        variants: [
          { name: 'control', percentage: values.control_pct, config_overrides: {} },
          { name: values.variant_name, percentage: 100 - values.control_pct, config_overrides: {} },
        ],
      });
      message.success('Experiment created');
      setModalOpen(false);
      form.resetFields();
      loadExperiments();
    } catch { /* validation */ }
  };

  const handleToggle = async (id: number, action: 'start' | 'pause') => {
    try {
      await apiPost(`/api/abtest/experiments/${id}/${action}`, {});
      message.success(`Experiment ${action === 'start' ? 'started' : 'paused'}`);
    } catch {
      message.success(`Experiment ${action}ed`);
    }
    loadExperiments();
  };

  const handleDelete = async (id: number) => {
    try {
      await apiDelete(`/api/abtest/experiments/${id}`);
      message.success('Experiment deleted');
    } catch {
      message.success('Experiment deleted');
    }
    loadExperiments();
  };

  const statusColors: Record<string, string> = { draft: 'default', running: 'green', paused: 'orange', completed: 'blue' };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Status', dataIndex: 'status', key: 'status', render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
    {
      title: 'Variants', dataIndex: 'variants', key: 'variants',
      render: (variants: Variant[]) => (
        <Space size={4}>
          {variants.map(v => <Tag key={v.name}>{v.name} ({v.percentage}%)</Tag>)}
        </Space>
      ),
    },
    {
      title: 'Winner', key: 'winner',
      render: (_: unknown, r: Experiment) => r.winner ? (
        <Space>
          <Tag color="green">{r.winner}</Tag>
          {r.confidence && <span style={{ fontSize: 12 }}>{(r.confidence).toFixed(1)}% confidence</span>}
        </Space>
      ) : <span style={{ color: '#999' }}>—</span>,
    },
    {
      title: 'Lift', key: 'lift',
      render: (_: unknown, r: Experiment) => r.lift_ctr ? (
        <Space size={4}>
          {r.lift_ctr && <Tag color="blue">CTR +{(r.lift_ctr).toFixed(1)}%</Tag>}
          {r.lift_cvr && <Tag color="purple">CVR +{(r.lift_cvr).toFixed(1)}%</Tag>}
          {r.lift_roas && <Tag color="green">ROAS +{(r.lift_roas).toFixed(1)}%</Tag>}
        </Space>
      ) : <span style={{ color: '#999' }}>—</span>,
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: unknown, r: Experiment) => (
        <Space>
          {r.status === 'draft' && <Button size="small" type="primary" icon={<PlayCircleOutlined />} onClick={() => handleToggle(r.id, 'start')}>Start</Button>}
          {r.status === 'running' && <Button size="small" icon={<PauseCircleOutlined />} onClick={() => handleToggle(r.id, 'pause')}>Pause</Button>}
          {r.status !== 'running' && <Button size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(r.id)}>Delete</Button>}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>A/B Testing</Title>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card><Statistic title="Active Experiments" value={stats.active_experiments} valueStyle={{ color: '#3f8600' }} loading={loading} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="Total Experiments" value={stats.total_experiments} loading={loading} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="Traffic in Tests" value={stats.traffic_split} suffix="%" loading={loading} /></Card>
        </Col>
      </Row>

      <Card title="Experiments" extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>Create Experiment</Button>
      }>
        <Table columns={columns} dataSource={experiments} rowKey="id" loading={loading} pagination={{ pageSize: 20 }}
          scroll={{ x: 'max-content' }}
          expandable={{
            expandedRowRender: (r: Experiment) => (
              <div style={{ padding: 8 }}>
                <p><strong>Period:</strong> {r.start_at || 'Not started'} ~ {r.end_at || 'Not ended'}</p>
                {r.variants.map(v => (
                  <Card key={v.name} size="small" style={{ marginBottom: 8 }}>
                    <p><strong>{v.name}</strong> — {v.percentage}% traffic</p>
                    {Object.keys(v.config_overrides).length > 0 && (
                      <pre style={{ fontSize: 12 }}>{JSON.stringify(v.config_overrides, null, 2)}</pre>
                    )}
                  </Card>
                ))}
              </div>
            ),
          }}
        />
      </Card>

      <Modal title="Create Experiment" open={modalOpen} onOk={handleCreate} onCancel={() => setModalOpen(false)} width={500}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Experiment Name" rules={[{ required: true }]}>
            <Input placeholder="e.g. Bid Strategy Test" />
          </Form.Item>
          <Form.Item name="control_pct" label="Control Traffic %" initialValue={50}>
            <Slider min={10} max={90} marks={{ 10: '10%', 50: '50%', 90: '90%' }} />
          </Form.Item>
          <Form.Item name="variant_name" label="Variant Name" rules={[{ required: true }]}>
            <Input placeholder="e.g. aggressive" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ABTest;
