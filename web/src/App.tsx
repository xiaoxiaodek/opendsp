import { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, Link, useLocation, useNavigate } from 'react-router-dom';
import { Layout, Menu, Button, Space, Spin } from 'antd';
import { DashboardOutlined, FundOutlined, AppstoreOutlined, PictureOutlined, BarChartOutlined, LogoutOutlined, TeamOutlined, SettingOutlined, GlobalOutlined, UsergroupAddOutlined, DollarOutlined, SafetyOutlined, AuditOutlined, ExperimentOutlined, ApiOutlined, ShoppingOutlined } from '@ant-design/icons';
import Dashboard from './pages/Dashboard';
import Campaign from './pages/Campaign';
import AdGroup from './pages/AdGroup';
import Creative from './pages/Creative';
import Report from './pages/Report';
import Login from './pages/Login';
import Advertiser from './pages/Advertiser';
import Admin from './pages/Admin';
import Media from './pages/Media';
import Audience from './pages/Audience';
import ROI from './pages/ROI';
import AntiFraud from './pages/AntiFraud';
import Settlement from './pages/Settlement';
import DPA from './pages/DPA';
import RTA from './pages/RTA';
import ABTest from './pages/ABTest';
import ChatWidget from './components/AIChat';

const { Sider, Content, Header } = Layout;

interface MenuItem {
  key: string;
  icon?: React.ReactNode;
  label: React.ReactNode;
  children?: MenuItem[];
  roles: string[];
}

const allMenuItems: MenuItem[] = [
  { key: '/', icon: <DashboardOutlined />, label: <Link to="/">Dashboard</Link>, roles: ['admin', 'operator', 'viewer'] },
  {
    key: 'delivery', icon: <FundOutlined />, label: 'Ad Delivery', roles: ['admin', 'operator'],
    children: [
      { key: '/campaigns', icon: <FundOutlined />, label: <Link to="/campaigns">Campaigns</Link>, roles: ['admin', 'operator'] },
      { key: '/adgroups', icon: <AppstoreOutlined />, label: <Link to="/adgroups">Ad Groups</Link>, roles: ['admin', 'operator'] },
      { key: '/creatives', icon: <PictureOutlined />, label: <Link to="/creatives">Creatives</Link>, roles: ['admin', 'operator'] },
    ],
  },
  {
    key: 'analytics', icon: <BarChartOutlined />, label: 'Analytics', roles: ['admin', 'operator', 'viewer'],
    children: [
      { key: '/reports', icon: <BarChartOutlined />, label: <Link to="/reports">Reports</Link>, roles: ['admin', 'operator', 'viewer'] },
      { key: '/roi', icon: <DollarOutlined />, label: <Link to="/roi">ROI</Link>, roles: ['admin', 'operator', 'viewer'] },
      { key: '/settlement', icon: <AuditOutlined />, label: <Link to="/settlement">Settlement</Link>, roles: ['admin'] },
    ],
  },
  {
    key: 'tools', icon: <ExperimentOutlined />, label: 'Tools', roles: ['admin'],
    children: [
      { key: '/dpa', icon: <ShoppingOutlined />, label: <Link to="/dpa">DPA</Link>, roles: ['admin', 'operator'] },
      { key: '/rta', icon: <ApiOutlined />, label: <Link to="/rta">RTA</Link>, roles: ['admin'] },
      { key: '/abtest', icon: <ExperimentOutlined />, label: <Link to="/abtest">A/B Test</Link>, roles: ['admin'] },
      { key: '/antifraud', icon: <SafetyOutlined />, label: <Link to="/antifraud">Anti-Fraud</Link>, roles: ['admin'] },
    ],
  },
  {
    key: 'settings', icon: <SettingOutlined />, label: 'Settings', roles: ['admin', 'operator'],
    children: [
      { key: '/advertisers', icon: <TeamOutlined />, label: <Link to="/advertisers">Advertisers</Link>, roles: ['admin', 'operator'] },
      { key: '/audiences', icon: <UsergroupAddOutlined />, label: <Link to="/audiences">Audiences</Link>, roles: ['admin', 'operator'] },
      { key: '/media', icon: <GlobalOutlined />, label: <Link to="/media">Media</Link>, roles: ['admin'] },
      { key: '/admin', icon: <SettingOutlined />, label: <Link to="/admin">Admin</Link>, roles: ['admin'] },
    ],
  },
];

function filterMenuItems(items: MenuItem[], role: string): MenuItem[] {
  return items
    .filter(item => item.roles.includes(role))
    .map(item => ({
      ...item,
      children: item.children ? item.children.filter(c => c.roles.includes(role)) : undefined,
    }))
    .filter(item => !item.children || item.children.length > 0);
}

function ProtectedLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const token = localStorage.getItem('token');

  useEffect(() => {
    if (!token) {
      navigate('/login', { replace: true });
    }
  }, [token, navigate]);

  if (!token) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  const user = (() => {
    try {
      const raw = localStorage.getItem('user');
      if (!raw || raw === 'undefined') return {};
      return JSON.parse(raw);
    } catch {
      return {};
    }
  })();

  const role: string = user.role || 'viewer';
  const menuItems = filterMenuItems(allMenuItems, role);

  const selectedKey = location.pathname === '/' ? '/' : location.pathname;
  const allOpenKeys = ['delivery', 'analytics', 'tools', 'settings'].filter(k =>
    filterMenuItems(allMenuItems, role).some(m => m.key === k)
  );
  const [openKeys, setOpenKeys] = useState<string[]>(allOpenKeys);

  const handleOpenChange = (keys: string[]) => {
    if (keys.length === 0) return;
    setOpenKeys(keys);
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} theme="dark">
        <div style={{ color: 'white', fontSize: 20, fontWeight: 'bold', padding: '16px 24px' }}>OpenDSP</div>
        <Menu theme="dark" mode="inline" selectedKeys={[selectedKey]} openKeys={openKeys} onOpenChange={handleOpenChange} items={menuItems} />
      </Sider>
      <Layout>
        <Header style={{ background: '#fff', padding: '0 24px', display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
          <Space>
            <span>{user.name || user.email}</span>
            <span style={{ color: '#999' }}>({role})</span>
            <Button icon={<LogoutOutlined />} onClick={() => { localStorage.clear(); navigate('/login'); }}>Logout</Button>
          </Space>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: '#fff', borderRadius: 8 }}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/reports" element={<Report />} />
            <Route path="/roi" element={<ROI />} />
            <Route path="/dpa" element={<DPA />} />
            {role !== 'viewer' && (
              <>
                <Route path="/campaigns" element={<Campaign />} />
                <Route path="/adgroups" element={<AdGroup />} />
                <Route path="/creatives" element={<Creative />} />
                <Route path="/advertisers" element={<Advertiser />} />
                <Route path="/audiences" element={<Audience />} />
              </>
            )}
            {role === 'admin' && (
              <>
                <Route path="/media" element={<Media />} />
                <Route path="/admin" element={<Admin />} />
                <Route path="/antifraud" element={<AntiFraud />} />
                <Route path="/settlement" element={<Settlement />} />
                <Route path="/rta" element={<RTA />} />
                <Route path="/abtest" element={<ABTest />} />
              </>
            )}
          </Routes>
        </Content>
      </Layout>
      <ChatWidget />
    </Layout>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/*" element={<ProtectedLayout />} />
      </Routes>
    </BrowserRouter>
  );
}
