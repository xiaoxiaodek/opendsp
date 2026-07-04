import React, { useEffect, useState } from 'react';
import { Card, Col, DatePicker, Row, Statistic, Table, Space, Typography } from 'antd';
import { RiseOutlined, FallOutlined, SwapOutlined, DollarOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { fetchROISummary, fetchROIByCampaign } from '../../services/roi';
import type { ROIMetrics, ROISummary } from '../../services/roi';

const { Title } = Typography;
const { RangePicker } = DatePicker;

const ROI: React.FC = () => {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'), dayjs(),
  ]);
  const [summary, setSummary] = useState<ROISummary | null>(null);
  const [campaignMetrics, setCampaignMetrics] = useState<ROIMetrics[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadData();
  }, [dateRange]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [summaryData, campaignData] = await Promise.all([
        fetchROISummary({
          advertiser_id: 0,
          start_date: dateRange[0].format('YYYY-MM-DD'),
          end_date: dateRange[1].format('YYYY-MM-DD'),
        }),
        fetchROIByCampaign({
          advertiser_id: 0,
          start_date: dateRange[0].format('YYYY-MM-DD'),
          end_date: dateRange[1].format('YYYY-MM-DD'),
        }),
      ]);
      setSummary(summaryData);
      setCampaignMetrics(campaignData);
    } catch (err) {
      console.error('Failed to load ROI data:', err);
    } finally {
      setLoading(false);
    }
  };

  const formatMicros = (micros: number) => (micros / 1_000_000).toFixed(2);

  const columns = [
    { title: 'Date', dataIndex: 'date', key: 'date' },
    { title: 'Cost', dataIndex: 'cost_micros', key: 'cost', render: (v: number) => `¥${formatMicros(v)}` },
    { title: 'Revenue', dataIndex: 'revenue_micros', key: 'revenue', render: (v: number) => `¥${formatMicros(v)}` },
    { title: 'Conversions', dataIndex: 'conversions', key: 'conversions' },
    {
      title: 'ROAS',
      dataIndex: 'roas',
      key: 'roas',
      render: (v: number) => <span style={{ color: v >= 1 ? '#52c41a' : '#ff4d4f' }}>{v.toFixed(2)}x</span>,
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>ROI Dashboard</Title>
      <Card style={{ marginBottom: 16 }}>
        <Space>
          <RangePicker
            value={dateRange}
            onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
          />
        </Space>
      </Card>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card><Statistic title="Total Cost" value={summary ? formatMicros(summary.total_cost) : '0'} prefix={<DollarOutlined />} loading={loading} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="Total Revenue" value={summary ? formatMicros(summary.total_revenue) : '0'} prefix={<DollarOutlined />} valueStyle={{ color: '#3f8600' }} loading={loading} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="Conversions" value={summary?.total_conversions ?? 0} prefix={<SwapOutlined />} loading={loading} /></Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Overall ROAS"
              value={summary?.overall_roas ?? 0}
              suffix="x"
              precision={2}
              prefix={summary && summary.overall_roas >= 1 ? <RiseOutlined /> : <FallOutlined />}
              valueStyle={{ color: summary && summary.overall_roas >= 1 ? '#3f8600' : '#cf1322' }}
              loading={loading}
            />
          </Card>
        </Col>
      </Row>

      <Card title="Daily Breakdown">
        <Table columns={columns} dataSource={campaignMetrics} rowKey="date" loading={loading} pagination={{ pageSize: 10 }} />
      </Card>
    </div>
  );
};

export default ROI;
