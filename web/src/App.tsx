import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Link, useLocation, useNavigate } from 'react-router-dom';
import { Layout, Menu, Button, Space, Spin } from 'antd';
import { DashboardOutlined, FundOutlined, AppstoreOutlined, PictureOutlined, BarChartOutlined, LogoutOutlined, TeamOutlined, SettingOutlined, GlobalOutlined, UsergroupAddOutlined } from '@ant-design/icons';
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
import ChatWidget from './components/AIChat';

const { Sider, Content, Header } = Layout;

const allMenuItems = [
  { key: '/', icon: <DashboardOutlined />, label: <Link to="/">Dashboard</Link>, roles: ['admin', 'operator', 'viewer'] },
  { key: '/campaigns', icon: <FundOutlined />, label: <Link to="/campaigns">Campaigns</Link>, roles: ['admin', 'operator'] },
  { key: '/adgroups', icon: <AppstoreOutlined />, label: <Link to="/adgroups">Ad Groups</Link>, roles: ['admin', 'operator'] },
  { key: '/creatives', icon: <PictureOutlined />, label: <Link to="/creatives">Creatives</Link>, roles: ['admin', 'operator'] },
  { key: '/reports', icon: <BarChartOutlined />, label: <Link to="/reports">Reports</Link>, roles: ['admin', 'operator', 'viewer'] },
  { key: '/advertisers', icon: <TeamOutlined />, label: <Link to="/advertisers">Advertisers</Link>, roles: ['admin', 'operator'] },
  { key: '/media', icon: <GlobalOutlined />, label: <Link to="/media">Media</Link>, roles: ['admin'] },
  { key: '/audiences', icon: <UsergroupAddOutlined />, label: <Link to="/audiences">Audiences</Link>, roles: ['admin', 'operator'] },
  { key: '/admin', icon: <SettingOutlined />, label: <Link to="/admin">Admin</Link>, roles: ['admin'] },
];

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
  const menuItems = allMenuItems.filter(item => item.roles.includes(role));

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} theme="dark">
        <div style={{ color: 'white', fontSize: 20, fontWeight: 'bold', padding: '16px 24px' }}>OpenDSP</div>
        <Menu theme="dark" mode="inline" selectedKeys={[location.pathname]} items={menuItems} />
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
