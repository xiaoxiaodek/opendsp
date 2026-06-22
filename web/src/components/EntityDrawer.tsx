import { useEffect, useState } from 'react';
import { Drawer, Card, Statistic, Row, Col, Table, Spin, Empty } from 'antd';
import { Line } from '@ant-design/charts';
import dayjs from 'dayjs';
import { getEntityReport, type EntityReport } from '../services/api';

interface Props {
  open: boolean;
  entityType: 'campaign' | 'adgroup' | 'creative';
  entityId: number;
  entityName: string;
  onClose: () => void;
}

const subLabel: Record<string, string> = {
  campaign: 'Ad Groups',
  adgroup: 'Creatives',
};

export default function EntityDrawer({ open, entityType, entityId, entityName, onClose }: Props) {
  const [data, setData] = useState<EntityReport | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || !entityId) return;
    setLoading(true);
    const endTime = dayjs().toISOString();
    const startTime = dayjs().subtract(7, 'day').startOf('day').toISOString();
    getEntityReport(1, entityType, entityId, startTime, endTime)
      .then(res => setData(res.data))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [open, entityId, entityType]);

  const chartData = (data?.hourly || []).map(h => ({
    hour: dayjs(h.hour).format('MM-DD HH:mm'),
    Impressions: h.impressions,
    Cost: h.cost,
  }));

  const subColumns = [
    { title: '#', key: 'rank', render: (_: unknown, __: unknown, i: number) => i + 1, width: 50 },
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Impressions', dataIndex: 'impressions', key: 'impressions', render: (v: number) => v.toLocaleString() },
    { title: 'Clicks', dataIndex: 'clicks', key: 'clicks', render: (v: number) => v.toLocaleString() },
    { title: 'CTR', dataIndex: 'ctr', key: 'ctr', render: (v: number) => `${v?.toFixed(2)}%` },
    { title: 'Cost', dataIndex: 'cost', key: 'cost', render: (v: number) => `¥${v?.toFixed(2)}` },
    { title: 'CPM', dataIndex: 'cpm', key: 'cpm', render: (v: number) => `¥${v?.toFixed(2)}` },
  ];

  const hourlyColumns = [
    { title: 'Hour', dataIndex: 'hour', key: 'hour', render: (v: string) => dayjs(v).format('MM-DD HH:mm') },
    { title: 'Impressions', dataIndex: 'impressions', key: 'impressions', render: (v: number) => v.toLocaleString() },
    { title: 'Clicks', dataIndex: 'clicks', key: 'clicks', render: (v: number) => v.toLocaleString() },
    { title: 'CTR', dataIndex: 'ctr', key: 'ctr', render: (v: number) => `${v?.toFixed(2)}%` },
    { title: 'Cost', dataIndex: 'cost', key: 'cost', render: (v: number) => `¥${v?.toFixed(2)}` },
    { title: 'CPM', dataIndex: 'cpm', key: 'cpm', render: (v: number) => `¥${v?.toFixed(2)}` },
  ];

  return (
    <Drawer
      title={`${entityType.charAt(0).toUpperCase() + entityType.slice(1)}: ${entityName}`}
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" /></div>
      ) : !data ? (
        <Empty description="No data available" />
      ) : (
        <>
          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col span={6}><Card size="small"><Statistic title="Today Cost" value={data.todayCost} precision={2} prefix="¥" /></Card></Col>
            <Col span={6}><Card size="small"><Statistic title="Impressions" value={data.todayImpressions} /></Card></Col>
            <Col span={6}><Card size="small"><Statistic title="Clicks" value={data.todayClicks} /></Card></Col>
            <Col span={6}><Card size="small"><Statistic title="CTR" value={data.todayCtr} precision={2} suffix="%" /></Card></Col>
          </Row>

          <Card title="7-Day Trend" size="small" style={{ marginBottom: 24 }}>
            {chartData.length > 0 ? (
              <Line
                data={chartData}
                xField="hour"
                yField={['Impressions', 'Cost']}
                height={200}
                smooth
                legend={{ position: 'top' }}
              />
            ) : (
              <Empty description="No trend data" />
            )}
          </Card>

          {data.subItems && data.subItems.length > 0 && (
            <Card title={`Top ${subLabel[entityType] || 'Items'}`} size="small" style={{ marginBottom: 24 }}>
              <Table
                dataSource={data.subItems}
                columns={subColumns}
                rowKey="id"
                size="small"
                pagination={false}
              />
            </Card>
          )}

          <Card title="Hourly Breakdown" size="small">
            <Table
              dataSource={data.hourly || []}
              columns={hourlyColumns}
              rowKey="hour"
              size="small"
              pagination={{ pageSize: 24 }}
            />
          </Card>
        </>
      )}
    </Drawer>
  );
}
