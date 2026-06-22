import { useEffect, useState } from 'react';
import { Table, DatePicker, Space, Card, Statistic, Row, Col, Tooltip } from 'antd';
import { WarningOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { getReport, getDashboard, type Dashboard as DashboardData } from '../../services/api';
import { getReportAnomalies, type ReportAnomaly } from '../../services/ai';

interface ReportRow {
  hour: string;
  impressions: number;
  clicks: number;
  cost: number;
  ctr: number;
  cpm: number;
}

export default function ReportPage() {
  const [reports, setReports] = useState<ReportRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [, setDashboard] = useState<DashboardData | null>(null);
  const [anomalies, setAnomalies] = useState<ReportAnomaly[]>([]);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day').startOf('day'),
    dayjs().endOf('day'),
  ]);

  const fetchData = async () => {
    if (!dateRange[0] || !dateRange[1]) return;
    setLoading(true);
    try {
      const [reportRes, dashRes] = await Promise.all([
        getReport(1, dateRange[0].toISOString(), dateRange[1].toISOString()),
        getDashboard(1),
      ]);
      setReports(reportRes.data.reports || []);
      setDashboard(dashRes.data);

      getReportAnomalies(dateRange[0].toISOString(), dateRange[1].toISOString(), 1)
        .then(setAnomalies)
        .catch(() => {});
    } finally { setLoading(false); }
  };

  useEffect(() => { fetchData(); }, [dateRange]);

  const anomalyHours = new Set(anomalies.map(a => dayjs(a.hour).format('YYYY-MM-DD HH:mm')));

  const columns = [
    {
      title: 'Hour', dataIndex: 'hour', key: 'hour',
      render: (v: string) => {
        const formatted = dayjs(v).format('YYYY-MM-DD HH:mm');
        const anomaly = anomalies.find(a => dayjs(a.hour).format('YYYY-MM-DD HH:mm') === formatted);
        return (
          <span>
            {formatted}
            {anomaly && (
              <Tooltip title={anomaly.explanation}>
                <WarningOutlined style={{ color: '#faad14', marginLeft: 8 }} />
              </Tooltip>
            )}
          </span>
        );
      },
    },
    { title: 'Impressions', dataIndex: 'impressions', key: 'impressions', render: (v: number) => v.toLocaleString() },
    { title: 'Clicks', dataIndex: 'clicks', key: 'clicks', render: (v: number) => v.toLocaleString() },
    {
      title: 'CTR', dataIndex: 'ctr', key: 'ctr',
      render: (v: number, record: ReportRow) => {
        const formatted = dayjs(record.hour).format('YYYY-MM-DD HH:mm');
        const isAnomaly = anomalyHours.has(formatted);
        return (
          <span style={{ color: isAnomaly ? '#faad14' : undefined, fontWeight: isAnomaly ? 'bold' : undefined }}>
            {v?.toFixed(2)}%
          </span>
        );
      },
    },
    { title: 'Cost', dataIndex: 'cost', key: 'cost', render: (v: number) => `¥${v?.toFixed(2)}` },
    { title: 'CPM', dataIndex: 'cpm', key: 'cpm', render: (v: number) => `¥${v?.toFixed(2)}` },
  ];

  const totals = reports.reduce((acc, r) => ({
    impressions: acc.impressions + r.impressions,
    clicks: acc.clicks + r.clicks,
    cost: acc.cost + r.cost,
  }), { impressions: 0, clicks: 0, cost: 0 });

  return (
    <div>
      <h2>Reports</h2>

      {anomalies.length > 0 && (
        <Card size="small" style={{ marginBottom: 16, background: '#fffbe6', border: '1px solid #ffe58f' }}>
          <WarningOutlined style={{ color: '#faad14', marginRight: 8 }} />
          {anomalies.length} anomaly {anomalies.length > 1 ? 'points' : 'point'} detected in this period.
          Hover over <WarningOutlined style={{ color: '#faad14' }} /> icons for details.
        </Card>
      )}

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}><Card><Statistic title="Total Impressions" value={totals.impressions} /></Card></Col>
        <Col span={6}><Card><Statistic title="Total Clicks" value={totals.clicks} /></Card></Col>
        <Col span={6}><Card><Statistic title="Total Cost" value={totals.cost} precision={2} prefix="¥" /></Card></Col>
        <Col span={6}><Card><Statistic title="CTR" value={totals.impressions > 0 ? (totals.clicks / totals.impressions * 100) : 0} precision={2} suffix="%" /></Card></Col>
      </Row>

      <Space style={{ marginBottom: 16 }}>
        <DatePicker.RangePicker
          value={dateRange}
          onChange={(dates) => { if (dates?.[0] && dates?.[1]) setDateRange([dates[0], dates[1]]); }}
          showTime
        />
      </Space>

      <Table dataSource={reports} columns={columns} rowKey="hour" loading={loading} />
    </div>
  );
}
