import React from 'react';
import { Route, Switch } from 'wouter';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import HelmReleases from './pages/HelmReleases';
import HelmTestJobs from './pages/HelmTestJobs';

const App: React.FC = () => {
  return (
    <Layout>
      <Switch>
        <Route path="/" component={Dashboard} />
        <Route path="/releases" component={HelmReleases} />
        <Route path="/jobs" component={HelmTestJobs} />
        <Route>404: Page Not Found</Route>
      </Switch>
    </Layout>
  );
};

export default App;
