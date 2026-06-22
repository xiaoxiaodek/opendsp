import { useEffect, useState } from 'react';
import { Card, Col, Row, Statistic, Alert, Typography, Skeleton, Segmented, Table } from 'antd';
import { BulbOutlined } from '@ant-design/icons';
import { getDashboard, getDashboardBreakdown, type DimensionItem } from '../../services/api';
import { getDashboardInsight, type DashboardInsight } from '../../services/ai';
import type { Dashboard as DashboardData } from '../../services/api';

const { Text, Paragraph } = Typography;

const DIMENSIONS = ['All', 'Campaign', 'AdGroup', 'Creative'] as const;
type Dimension = typeof DIMENSIONS[number];

export default function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [insight, setInsight] = useState<DashboardInsight | null>(null);
  const [insightLoading, setInsightLoading] = useState(true);
  const [dimension, setDimension] = useState<Dimension>('All');
  const [items, setItems] = useState<DimensionItem[]>([]);
  const [itemsLoading, setItemsLoading] = useState(false);

  useEffect(() => {
    getDashboard(1).then(res => setData(res.data)).catch(() => {});
    getDashboardInsight(1)
      .then(setInsight)
      .catch(() => {})
      .finally(() => setInsightLoading(false));
  }, []);

  useEffect(() => {
    if (dimension === 'All') { setItems([]); return; }
    setItemsLoading(true);
    getDashboardBreakdown(1, dimension.toLowerCase(), 10)
      .then(res => setItems(res.data.items || []))
      .catch(() => {})
      .finally(() => setItemsLoading(false));
  }, [dimension]);

  const breakdownColumns = [
    { title: '#', key: 'rank', render: (_: any, __: any, i: number) => i + 1, width: 50 },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Impressions', dataIndex: 'impressions', key: 'impressions', render: (v: number) => v.toLocaleString() },
    { title: 'Clicks', dataIndex: 'clicks', key: 'clicks', render: (v: number) => v.toLocaleString() },
    { title: 'CTR', dataIndex: 'ctr', key: 'ctr', render: (v: number) => `${v?.toFixed(2)}%` },
    { title: 'Cost', dataIndex: 'cost', key: 'cost', render: (v: number) => `¥${v?.toFixed(2)}` },
    { title: 'CPM', dataIndex: 'cpm', key: 'cpm', render: (v: number) => `¥${v?.toFixed(2)}` },
  ];

  return (
    <div>
      <h2>Dashboard</h2>

      {insightLoading ? (
        <Card style={{ marginBottom: 24 }}>
          <Skeleton active paragraph={{ rows: 2 }} />
        </Card>
      ) : insight ? (
        <Alert
          type="info"
          icon={<BulbOutlined />}
          message="AI Insights"
          description={
            <div>
              <Paragraph style={{ marginBottom: 8 }}>{insight.summary}</Paragraph>
              {insight.pacingAlert && (
                <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>
                  {insight.pacingAlert}
                </Text>
              )}
              {insight.recommendation && (
                <Text strong style={{ color: '#1677ff' }}>💡 {insight.recommendation}</Text>
              )}
            </div>
          }
          style={{ marginBottom: 24 }}
        />
      ) : null}

      <Segmented
        options={DIMENSIONS.map(d => d === 'AdGroup' ? { label: 'Ad Group', value: d } : d)}
        value={dimension}
        onChange={v => setDimension(v as Dimension)}
        style={{ marginBottom: 24 }}
      />

      <Row gutter={16}>
        <Col span={6}><Card><Statistic title="Today Cost" value={data?.todayCost || 0} precision={2} prefix="¥" /></Card></Col>
        <Col span={6}><Card><Statistic title="Impressions" value={data?.todayImpressions || 0} /></Card></Col>
        <Col span={6}><Card><Statistic title="Clicks" value={data?.todayClicks || 0} /></Card></Col>
        <Col span={6}><Card><Statistic title="CTR" value={data?.todayCtr || 0} precision={2} suffix="%" /></Card></Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col span={6}><Card><Statistic title="Balance" value={data?.balance || 0} precision={2} prefix="¥" /></Card></Col>
        <Col span={6}><Card><Statistic title="Active Campaigns" value={data?.activeCampaigns || 0} /></Card></Col>
        <Col span={6}><Card><Statistic title="Active Ad Groups" value={data?.activeAdGroups || 0} /></Card></Col>
      </Row>

      {dimension !== 'All' && (
        <Card title={`Top ${dimension}s`} style={{ marginTop: 24 }}>
          <Table
            dataSource={items}
            columns={breakdownColumns}
            rowKey="id"
            loading={itemsLoading}
            pagination={false}
            size="small"
          />
        </Card>
      )}
    </div>
  );
}
