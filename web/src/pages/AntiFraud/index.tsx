import React from 'react';
import { Tabs, Typography } from 'antd';
import { SafetyOutlined, UnorderedListOutlined } from '@ant-design/icons';
import BlacklistTable from './BlacklistTable';
import EventLog from './EventLog';

const { Title } = Typography;

const AntiFraud: React.FC = () => {
  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>Anti-Fraud Management</Title>
      <Tabs
        defaultActiveKey="blacklist"
        items={[
          { key: 'blacklist', label: <span><UnorderedListOutlined /> Blacklist</span>, children: <BlacklistTable /> },
          { key: 'events', label: <span><SafetyOutlined /> Fraud Events</span>, children: <EventLog /> },
        ]}
      />
    </div>
  );
};

export default AntiFraud;
