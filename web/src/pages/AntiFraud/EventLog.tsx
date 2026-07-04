import React, { useEffect, useState } from 'react';
import { Table, Tag, DatePicker, Space, Statistic, Card, Row, Col } from 'antd';
import dayjs from 'dayjs';
import { fetchFraudEvents, fetchFraudStats } from '../../services/antifraud';
import type { FraudEvent, FraudStats } from '../../services/antifraud';

const { RangePicker } = DatePicker;

const EventLog: React.FC = () => {
  const [data, setData] = useState<FraudEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<FraudStats | null>(null);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(1, 'day'), dayjs(),
  ]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [result, statsData] = await Promise.all([
        fetchFraudEvents({ page, page_size: 20, start_date: dateRange[0].toISOString(), end_date: dateRange[1].toISOString() }),
        fetchFraudStats({ start_date: dateRange[0].toISOString(), end_date: dateRange[1].toISOString() }),
      ]);
      setData(result.items);
      setTotal(result.total);
      setStats(statsData);
    } finally { setLoading(false); }
  };

  useEffect(() => { loadData(); }, [page, dateRange]);

  const columns = [
    { title: 'Time', dataIndex: 'created_at', key: 'created_at' },
    { title: 'Request ID', dataIndex: 'request_id', key: 'request_id', ellipsis: true },
    { title: 'Rule', dataIndex: 'rule_type', key: 'rule_type', render: (t: string, r: FraudEvent) => `${t}: ${r.rule_value}` },
    {
      title: 'Risk Score', dataIndex: 'risk_score', key: 'risk_score',
      render: (v: number) => <span style={{ color: v > 0.8 ? '#ff4d4f' : v > 0.5 ? '#faad14' : '#52c41a' }}>{v?.toFixed(4) ?? '-'}</span>,
    },
    { title: 'Action', dataIndex: 'action', key: 'action', render: (a: string) => <Tag color={a === 'blocked' ? 'red' : 'orange'}>{a}</Tag> },
  ];

  return (
    <>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}><Card size="small"><Statistic title="Total Requests" value={stats?.total_requests ?? 0} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="Blocked" value={stats?.blocked ?? 0} valueStyle={{ color: '#ff4d4f' }} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="Flagged" value={stats?.flagged ?? 0} valueStyle={{ color: '#faad14' }} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="Block Rate" value={stats?.block_rate ?? 0} suffix="%" precision={2} /></Card></Col>
      </Row>
      <Space style={{ marginBottom: 16 }}>
        <RangePicker value={dateRange} onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])} />
      </Space>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} pagination={{ current: page, total, pageSize: 20, onChange: setPage }} />
    </>
  );
};

export default EventLog;
