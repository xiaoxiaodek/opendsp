import React, { useEffect, useState } from 'react';
import { Card, DatePicker, Space, Statistic, Table, Typography, Row, Col } from 'antd';
import { DollarOutlined, WarningOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { fetchSettlement } from '../../services/settlement';
import type { Discrepancy, SettlementSummary } from '../../services/settlement';

const { Title } = Typography;
const { RangePicker } = DatePicker;

const Settlement: React.FC = () => {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'), dayjs(),
  ]);
  const [summary, setSummary] = useState<SettlementSummary | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => { loadData(); }, [dateRange]);

  const loadData = async () => {
    setLoading(true);
    try {
      const data = await fetchSettlement({
        advertiser_id: 0,
        start_date: dateRange[0].format('YYYY-MM-DD'),
        end_date: dateRange[1].format('YYYY-MM-DD'),
      });
      setSummary(data);
    } catch (err) {
      console.error('Failed to load settlement data:', err);
    } finally { setLoading(false); }
  };

  const formatMicros = (micros: number) => `¥${(micros / 1_000_000).toFixed(2)}`;

  const columns = [
    { title: 'Date', dataIndex: 'date', key: 'date' },
    { title: 'DSP Cost', dataIndex: 'dsp_cost', key: 'dsp_cost', render: (v: number) => formatMicros(v) },
    { title: 'ADX Cost', dataIndex: 'adx_cost', key: 'adx_cost', render: (v: number) => formatMicros(v) },
    {
      title: 'Difference', dataIndex: 'difference', key: 'difference',
      render: (v: number) => <span style={{ color: Math.abs(v) > 100000 ? '#ff4d4f' : '#52c41a' }}>{formatMicros(v)}</span>,
    },
    { title: 'Diff %', dataIndex: 'difference_pct', key: 'difference_pct', render: (v: number) => `${v.toFixed(2)}%` },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>Settlement Report</Title>
      <Card style={{ marginBottom: 16 }}>
        <Space><RangePicker value={dateRange} onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])} /></Space>
      </Card>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}><Card><Statistic title="DSP Total Cost" value={summary ? formatMicros(summary.total_dsp_cost) : '¥0.00'} prefix={<DollarOutlined />} loading={loading} /></Card></Col>
        <Col span={8}><Card><Statistic title="ADX Total Cost" value={summary ? formatMicros(summary.total_adx_cost) : '¥0.00'} prefix={<DollarOutlined />} loading={loading} /></Card></Col>
        <Col span={8}>
          <Card>
            <Statistic title="Total Discrepancy" value={summary ? formatMicros(summary.total_discrepancy) : '¥0.00'}
              prefix={<WarningOutlined />}
              valueStyle={{ color: summary && Math.abs(summary.total_discrepancy) > 100000 ? '#ff4d4f' : '#52c41a' }}
              loading={loading} />
          </Card>
        </Col>
      </Row>
      <Card title="Daily Reconciliation">
        <Table columns={columns} dataSource={summary?.discrepancies ?? []} rowKey="date" loading={loading} pagination={{ pageSize: 10 }} />
      </Card>
    </div>
  );
};

export default Settlement;
