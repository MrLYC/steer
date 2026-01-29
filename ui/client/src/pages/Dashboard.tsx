import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Loading } from 'tdesign-react';
import { 
  ServerIcon, 
  TaskIcon, 
  CheckCircleIcon, 
  ErrorCircleIcon 
} from 'tdesign-icons-react';
import { helmReleaseApi, helmTestJobApi } from '../api/client';

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState({
    releases: 0,
    jobs: 0,
    successRate: 0,
    failedJobs: 0
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchStats = async () => {
    try {
      const [releasesRes, jobsRes] = await Promise.all([
        helmReleaseApi.list(),
        helmTestJobApi.list()
      ]);

      const releases = releasesRes.data;
      const jobs = jobsRes.data;
      
      const succeededJobs = jobs.filter(j => j.status.phase === 'Succeeded').length;
      const failedJobs = jobs.filter(j => j.status.phase === 'Failed').length;
      const totalCompletedJobs = succeededJobs + failedJobs;
      
      setStats({
        releases: releases.length,
        jobs: jobs.length,
        successRate: totalCompletedJobs > 0 ? Math.round((succeededJobs / totalCompletedJobs) * 100) : 0,
        failedJobs
      });
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }}><Loading /></div>;
  }

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col span={3}>
          <Card>
            <Statistic
              title="Total Releases"
              value={stats.releases}
              extra={<ServerIcon />}
            />
          </Card>
        </Col>
        <Col span={3}>
          <Card>
            <Statistic
              title="Total Jobs"
              value={stats.jobs}
              extra={<TaskIcon />}
            />
          </Card>
        </Col>
        <Col span={3}>
          <Card>
            <Statistic
              title="Success Rate"
              value={stats.successRate}
              unit="%"
              extra={<CheckCircleIcon style={{ color: 'var(--td-success-color)' }} />}
            />
          </Card>
        </Col>
        <Col span={3}>
          <Card>
            <Statistic
              title="Failed Jobs"
              value={stats.failedJobs}
              extra={<ErrorCircleIcon style={{ color: 'var(--td-error-color)' }} />}
            />
          </Card>
        </Col>
      </Row>
      
      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col span={12}>
          <Card title="Recent Activity" bordered>
            <div style={{ padding: 20, textAlign: 'center', color: 'var(--td-text-color-secondary)' }}>
              Activity log will be displayed here
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
