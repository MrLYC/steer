import React, { useEffect, useState } from 'react';
import { Table, Button, Tag, Space, DialogPlugin, Dialog, Form, Input, Select, MessagePlugin, Drawer } from 'tdesign-react';
import { AddIcon, RefreshIcon, DeleteIcon, FileIcon } from 'tdesign-icons-react';
import { helmTestJobApi, helmReleaseApi, HelmTestJob, HelmRelease } from '../api/client';

const HelmTestJobs: React.FC = () => {
  const [jobs, setJobs] = useState<HelmTestJob[]>([]);
  const [releases, setReleases] = useState<HelmRelease[]>([]);
  const [loading, setLoading] = useState(true);
  const [visible, setVisible] = useState(false);
  const [logVisible, setLogVisible] = useState(false);
  const [currentJob, setCurrentJob] = useState<HelmTestJob | null>(null);
  const [form] = Form.useForm();
  const [scheduleType, setScheduleType] = useState<'once' | 'cron'>('once');

  useEffect(() => {
    loadJobs();
    loadReleases();
  }, []);

  const loadJobs = async () => {
    setLoading(true);
    try {
      const response = await helmTestJobApi.list();
      setJobs(response.data);
    } catch (error) {
      MessagePlugin.error('Failed to load jobs');
    } finally {
      setLoading(false);
    }
  };

  const loadReleases = async () => {
    try {
      const response = await helmReleaseApi.list();
      setReleases(response.data);
    } catch (error) {
      console.error('Failed to load releases');
    }
  };

  const handleDelete = async (row: HelmTestJob) => {
    const confirmDialog = DialogPlugin.confirm({
      header: 'Confirm Delete',
      body: `Are you sure you want to delete job ${row.metadata.name}?`,
      onConfirm: async () => {
        try {
          await helmTestJobApi.delete(row.metadata.name);
          MessagePlugin.success('Job deleted successfully');
          loadJobs();
          confirmDialog.hide();
        } catch (error) {
          MessagePlugin.error('Failed to delete job');
        }
      },
    });
  };

  const handleSubmit = async (context: any) => {
    if (context.validateResult === true) {
      const values = form.getFieldsValue(true);

      const schedule: HelmTestJob['spec']['schedule'] = {
        type: values.scheduleType,
      };

      if (values.scheduleType === 'once') {
        schedule.delay = values.delay;
      }

      if (values.scheduleType === 'cron') {
        schedule.cron = values.cron;
        schedule.timezone = values.timezone;
      }
       
      const newJob: HelmTestJob = {
        apiVersion: 'steer.io/v1alpha1',
        kind: 'HelmTestJob',
        metadata: {
          name: values.name,
        },
        spec: {
          helmReleaseRef: {
            name: values.release,
          },
          schedule,
          test: {
            timeout: '10m',
            logs: true,
          },
        },
      };

      try {
        await helmTestJobApi.create(newJob);
        MessagePlugin.success('Job created successfully');
        setVisible(false);
        form.reset();
        loadJobs();
      } catch (error) {
        MessagePlugin.error('Failed to create job');
      }
    }
  };

  const showLogs = (row: HelmTestJob) => {
    setCurrentJob(row);
    setLogVisible(true);
  };

  const columns = [
    { colKey: 'metadata.name', title: 'Name' },
    { 
      colKey: 'spec.helmReleaseRef.name', 
      title: 'Target Release',
      cell: ({ row }: { row: HelmTestJob }) => row.spec.helmReleaseRef.name
    },
    { 
      colKey: 'spec.schedule.type', 
      title: 'Schedule',
      cell: ({ row }: { row: HelmTestJob }) => (
        <Space>
          <Tag variant="light">{row.spec.schedule.type}</Tag>
          {row.spec.schedule.type === 'cron' && <span>{row.spec.schedule.cron}</span>}
          {row.spec.schedule.type === 'once' && row.spec.schedule.delay && <span>(Delay: {row.spec.schedule.delay})</span>}
        </Space>
      )
    },
    { 
      colKey: 'status.phase', 
      title: 'Status',
      cell: ({ row }: { row: HelmTestJob }) => {
        const phase = row.status?.phase || 'Unknown';
        const theme = phase === 'Succeeded' ? 'success' : 
                      phase === 'Failed' ? 'danger' : 
                      phase === 'Running' ? 'warning' : 'primary';
        return <Tag theme={theme}>{phase}</Tag>;
      }
    },
    {
      colKey: 'op',
      title: 'Operation',
      cell: ({ row }: { row: HelmTestJob }) => (
        <Space>
          <Button 
            theme="primary" 
            variant="text" 
            icon={<FileIcon />} 
            onClick={() => showLogs(row)}
          >
            Logs
          </Button>
          <Button 
            theme="danger" 
            variant="text" 
            icon={<DeleteIcon />} 
            onClick={() => handleDelete(row)}
          />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Button
          icon={<AddIcon />}
          onClick={() => {
            setScheduleType('once');
            setVisible(true);
          }}
        >
          Create Test Job
        </Button>
        <Button icon={<RefreshIcon />} variant="outline" onClick={loadJobs}>Refresh</Button>
      </div>

      <Table
        data={jobs}
        columns={columns}
        rowKey="metadata.name"
        loading={loading}
      />

      <Dialog
        header="Create Test Job"
        visible={visible}
        onClose={() => setVisible(false)}
        onConfirm={() => form.submit()}
        width={600}
      >
          <Form form={form} onSubmit={handleSubmit} labelWidth={120}>
            <Form.FormItem name="name" label="Name" rules={[{ required: true }]}>
              <Input placeholder="Job name" />
            </Form.FormItem>
            <Form.FormItem name="release" label="Target Release" rules={[{ required: true }]}>
              <Select placeholder="Select a release">
                {releases.map((r: HelmRelease) => (
                  <Select.Option 
                    key={r.metadata.name} 
                    value={r.metadata.name} 
                    label={r.metadata.name} 
                  />
                ))}
              </Select>
            </Form.FormItem>
            <Form.FormItem name="scheduleType" label="Schedule Type" initialData="once">
              <Select onChange={(v: unknown) => setScheduleType(v as 'once' | 'cron')}>
                <Select.Option value="once" label="Once" />
                <Select.Option value="cron" label="Cron" />
              </Select>
            </Form.FormItem>
            {scheduleType === 'once' && (
              <Form.FormItem
                name="delay"
                label="Delay"
                help="Delay execution (e.g. 5m, 1h). Only for 'once' type."
              >
                <Input placeholder="e.g. 5m" />
              </Form.FormItem>
            )}
            {scheduleType === 'cron' && (
              <>
                <Form.FormItem
                  name="cron"
                  label="Cron Expression"
                  help="Standard cron expression. Only for 'cron' type."
                >
                  <Input placeholder="e.g. 0 2 * * *" />
                </Form.FormItem>
                <Form.FormItem name="timezone" label="Timezone" initialData="Asia/Shanghai">
                  <Input placeholder="Asia/Shanghai" />
                </Form.FormItem>
              </>
            )}
          </Form>
        </Dialog>

      <Drawer
        header={`Logs: ${currentJob?.metadata.name}`}
        visible={logVisible}
        onClose={() => setLogVisible(false)}
        size="large"
      >
        {currentJob && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <strong>Status: </strong>
              <Tag theme={(currentJob.status?.phase || '') === 'Succeeded' ? 'success' : 'danger'}>
                {currentJob.status?.phase || 'Unknown'}
              </Tag>
            </div>
            
            {currentJob.status?.message && (
              <div style={{ marginBottom: 16, padding: 12, background: 'var(--td-bg-color-secondary)', borderRadius: 4 }}>
                {currentJob.status.message}
              </div>
            )}

            <h3>Test Results</h3>
            {currentJob.status?.testResults?.map((result, index) => (
              <div key={index} style={{ marginBottom: 12, padding: 12, border: '1px solid var(--td-border-level-1-color)', borderRadius: 4 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <strong>{result.name}</strong>
                  <Tag theme={result.phase === 'Succeeded' ? 'success' : 'danger'}>{result.phase}</Tag>
                </div>
                {result.logs && (
                  <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{result.logs}</pre>
                )}
                {(result.startedAt || result.completedAt) && (
                  <div style={{ fontSize: 12, color: 'var(--td-text-color-secondary)', marginTop: 4 }}>
                    {result.startedAt ? new Date(result.startedAt).toLocaleString() : '-'} - {result.completedAt ? new Date(result.completedAt).toLocaleString() : '-'}
                  </div>
                )}
              </div>
            ))}

            <h3>Hook Results</h3>
            <h4>PreTest</h4>
            {currentJob.status?.hookResults?.preTest?.map((result, index) => (
              <div key={`pre-${index}`} style={{ marginBottom: 12, padding: 12, border: '1px solid var(--td-border-level-1-color)', borderRadius: 4 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <strong>{result.name}</strong>
                  <Tag theme={result.phase === 'Succeeded' ? 'success' : 'danger'}>{result.phase}</Tag>
                </div>
                {result.message && <div>{result.message}</div>}
              </div>
            ))}

            <h4>PostTest</h4>
            {currentJob.status?.hookResults?.postTest?.map((result, index) => (
              <div key={`post-${index}`} style={{ marginBottom: 12, padding: 12, border: '1px solid var(--td-border-level-1-color)', borderRadius: 4 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <strong>{result.name}</strong>
                  <Tag theme={result.phase === 'Succeeded' ? 'success' : 'danger'}>{result.phase}</Tag>
                </div>
                {result.message && <div>{result.message}</div>}
              </div>
            ))}
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default HelmTestJobs;
