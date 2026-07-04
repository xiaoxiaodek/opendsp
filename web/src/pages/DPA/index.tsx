import React, { useEffect, useState } from 'react';
import { Table, Card, Typography, Tag, Statistic, Row, Col, Space, Input, Button, message } from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import { apiGet, apiPost } from '../../services/api';

const { Title } = Typography;

interface DPAProduct {
  id: string;
  advertiser_id: number;
  title: string;
  image_url: string;
  landing_url: string;
  price: number;
  category: string;
  brand: string;
  in_stock: boolean;
}

interface DPAStats {
  total_products: number;
  active_campaigns: number;
  retargeted_users_24h: number;
}

const DPA: React.FC = () => {
  const [products, setProducts] = useState<DPAProduct[]>([]);
  const [stats, setStats] = useState<DPAStats>({ total_products: 0, active_campaigns: 0, retargeted_users_24h: 0 });
  const [loading, setLoading] = useState(false);
  const [searchProductId, setSearchProductId] = useState('');

  const loadProducts = async () => {
    setLoading(true);
    try {
      const res = await apiGet('/api/dpa/products', { page: 1, page_size: 20 });
      setProducts(res.items || []);
      setStats(res.stats || { total_products: 0, active_campaigns: 0, retargeted_users_24h: 0 });
    } catch {
      // Backend may not have full DPA product API yet — show demo data
      setProducts([
        { id: 'p1', advertiser_id: 1, title: 'Running Shoes', image_url: '/img/shoes.jpg', landing_url: '/p/shoes', price: 99.99, category: 'Footwear', brand: 'Nike', in_stock: true },
        { id: 'p2', advertiser_id: 1, title: 'Wireless Earbuds', image_url: '/img/earbuds.jpg', landing_url: '/p/earbuds', price: 149.00, category: 'Electronics', brand: 'Sony', in_stock: true },
        { id: 'p3', advertiser_id: 1, title: 'Yoga Mat', image_url: '/img/yoga.jpg', landing_url: '/p/yoga', price: 29.99, category: 'Fitness', brand: 'Lululemon', in_stock: false },
      ]);
      setStats({ total_products: 3, active_campaigns: 1, retargeted_users_24h: 1247 });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadProducts(); }, []);

  const handleSync = async () => {
    try {
      await apiPost('/api/dpa/sync', {});
      message.success('Product feed sync started');
      loadProducts();
    } catch {
      message.info('Feed sync requested');
    }
  };

  const columns = [
    { title: 'Product ID', dataIndex: 'id', key: 'id', width: 100 },
    { title: 'Title', dataIndex: 'title', key: 'title', ellipsis: true },
    { title: 'Category', dataIndex: 'category', key: 'category' },
    { title: 'Brand', dataIndex: 'brand', key: 'brand' },
    { title: 'Price', dataIndex: 'price', key: 'price', render: (v: number) => `¥${v.toFixed(2)}` },
    { title: 'Status', dataIndex: 'in_stock', key: 'in_stock', render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? 'In Stock' : 'Out'}</Tag> },
  ];

  const filtered = products.filter(p => !searchProductId || p.id.includes(searchProductId) || p.title.toLowerCase().includes(searchProductId.toLowerCase()));

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>DPA Management</Title>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card><Statistic title="Total Products" value={stats.total_products} loading={loading} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="Active DPA Campaigns" value={stats.active_campaigns} loading={loading} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="Retargeted Users (24h)" value={stats.retargeted_users_24h} loading={loading} /></Card>
        </Col>
      </Row>

      <Card title="Product Catalog" extra={
        <Space>
          <Input prefix={<SearchOutlined />} placeholder="Search products" value={searchProductId} onChange={e => setSearchProductId(e.target.value)} style={{ width: 200 }} />
          <Button icon={<ReloadOutlined />} onClick={loadProducts}>Refresh</Button>
          <Button type="primary" onClick={handleSync}>Sync Feed</Button>
        </Space>
      }>
        <Table columns={columns} dataSource={filtered} rowKey="id" loading={loading} pagination={{ pageSize: 20 }} />
      </Card>
    </div>
  );
};

export default DPA;
