import React from 'react';
import { Layout as TLayout, Menu, Button } from 'tdesign-react';
import { 
  DashboardIcon, 
  ServerIcon, 
  TaskIcon, 
  LogoGithubIcon 
} from 'tdesign-icons-react';
import { useLocation, useRoute } from 'wouter';

const { Header, Content, Aside, Footer } = TLayout;
const { MenuItem } = Menu;

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [location, setLocation] = useLocation();

  return (
    <TLayout style={{ minHeight: '100vh' }}>
      <Aside>
        <Menu
          value={location}
          onChange={(value) => setLocation(value as string)}
          style={{ marginRight: 0, height: '100%' }}
          logo={
            <div style={{ 
              height: 64, 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              fontSize: 24,
              fontWeight: 'bold',
              color: 'var(--td-brand-color)'
            }}>
              STEER
            </div>
          }
        >
          <MenuItem value="/" icon={<DashboardIcon />}>
            Dashboard
          </MenuItem>
          <MenuItem value="/releases" icon={<ServerIcon />}>
            Helm Releases
          </MenuItem>
          <MenuItem value="/jobs" icon={<TaskIcon />}>
            Test Jobs
          </MenuItem>
        </Menu>
      </Aside>
      <TLayout>
        <Header style={{ background: 'var(--td-bg-color-container)', padding: '0 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontSize: 18, fontWeight: 600 }}>
            {location === '/' ? 'Dashboard' : 
             location === '/releases' ? 'Helm Releases' : 
             location === '/jobs' ? 'Test Jobs' : 'Steer'}
          </div>
          <Button 
            variant="text" 
            shape="square" 
            icon={<LogoGithubIcon />} 
            onClick={() => window.open('https://github.com/MrLYC/steer', '_blank')}
          />
        </Header>
        <Content style={{ padding: 24, overflowY: 'auto' }}>
          {children}
        </Content>
        <Footer style={{ textAlign: 'center', padding: '16px 0' }}>
          Steer Operator &copy; 2026 Created by Manus
        </Footer>
      </TLayout>
    </TLayout>
  );
};

export default Layout;
