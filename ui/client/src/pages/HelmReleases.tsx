import React, { useEffect, useState } from 'react';
import { Table, Button, Tag, Space, DialogPlugin, Dialog, Form, Input, Textarea, MessagePlugin, Checkbox } from 'tdesign-react';
import { AddIcon, RefreshIcon, DeleteIcon } from 'tdesign-icons-react';
import { helmReleaseApi, HelmRelease } from '../api/client';

type ErrorWithResponse = {
  message?: string;
  response?: {
    data?: unknown;
    status?: number;
  };
};

function getErrorMessage(error: unknown, fallback: string) {
  if (typeof error === 'object' && error !== null) {
    const e = error as ErrorWithResponse;
    const data = e.response?.data;
    if (typeof data === 'string' && data.trim()) return data;
    if (typeof data === 'object' && data !== null && 'error' in data) {
      const errMsg = (data as { error?: unknown }).error;
      if (typeof errMsg === 'string' && errMsg.trim()) return errMsg;
    }
    if (typeof e.message === 'string' && e.message.trim()) return e.message;
  }
  return fallback;
}

const HelmReleases: React.FC = () => {
  const [releases, setReleases] = useState<HelmRelease[]>([]);
  const [loading, setLoading] = useState(true);
  const [visible, setVisible] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    loadReleases();
  }, []);

  const loadReleases = async () => {
    setLoading(true);
    try {
      const response = await helmReleaseApi.list();
      setReleases(response.data);
    } catch (error) {
      MessagePlugin.error('Failed to load releases');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (row: HelmRelease) => {
    const confirmDialog = DialogPlugin.confirm({
      header: 'Confirm Delete',
      body: `Are you sure you want to delete release ${row.metadata.name}?`,
      onConfirm: async () => {
        try {
          await helmReleaseApi.delete(row.metadata.name);
          MessagePlugin.success('Release deleted successfully');
          loadReleases();
          confirmDialog.hide();
        } catch (error) {
          MessagePlugin.error('Failed to delete release');
        }
      },
    });
  };

  const handleSubmit = async (context: any) => {
    if (context.validateResult === true) {
      const values = form.getFieldsValue(true);
      const targetNamespace = values.targetNamespace;
      const createNamespace = values.createNamespace !== false;
      const valuesInline = typeof values.values === 'string' ? values.values : '';

      const newRelease: HelmRelease = {
        apiVersion: 'steer.io/v1alpha1',
        kind: 'HelmRelease',
        metadata: {
          name: values.name,
        },
        spec: {
          chart: {
            source: 'repository',
            repository: {
              name: values.chartName,
              url: values.repository,
              version: values.version,
            },
          },
          values: {
            inline: valuesInline,
          },
          deployment: {
            namespace: targetNamespace,
            createNamespace,
          },
        },
      };

      try {
        await helmReleaseApi.create(newRelease);
        MessagePlugin.success('Release created successfully');
        setVisible(false);
        form.reset();
        loadReleases();
      } catch (error) {
        MessagePlugin.error(getErrorMessage(error, 'Failed to create release'));
      }
    }
  };

  const columns = [
    { colKey: 'metadata.name', title: 'Name' },
    {
      colKey: 'spec.deployment.namespace',
      title: 'Target Namespace',
      cell: ({ row }: { row: HelmRelease }) => row.spec.deployment?.namespace || '-',
    },
    { 
      colKey: 'spec.chart.name', 
      title: 'Chart',
      cell: ({ row }: { row: HelmRelease }) => {
        const repo = row.spec.chart.repository;
        const name = repo?.name || '-';
        const ver = repo?.version || 'latest';
        return `${name} (${ver})`;
      }
    },
    { 
      colKey: 'status.phase', 
      title: 'Status',
      cell: ({ row }: { row: HelmRelease }) => {
        const phase = row.status?.phase || 'Unknown';
        const theme = phase === 'Installed' ? 'success' : 
                      phase === 'Failed' ? 'danger' : 
                      phase === 'Installing' ? 'warning' : 'primary';
        return <Tag theme={theme}>{phase}</Tag>;
      }
    },
    { 
      colKey: 'status.deployedAt', 
      title: 'Deployed At',
      cell: ({ row }: { row: HelmRelease }) => row.status?.deployedAt ? new Date(row.status.deployedAt).toLocaleString() : '-'
    },
    {
      colKey: 'op',
      title: 'Operation',
      cell: ({ row }: { row: HelmRelease }) => (
        <Button 
          theme="danger" 
          variant="text" 
          icon={<DeleteIcon />} 
          onClick={() => handleDelete(row)}
        />
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Button icon={<AddIcon />} onClick={() => setVisible(true)}>Create Release</Button>
        <Button icon={<RefreshIcon />} variant="outline" onClick={loadReleases}>Refresh</Button>
      </div>

      <Table
        data={releases}
        columns={columns}
        rowKey="metadata.name"
        loading={loading}
      />

      <Dialog
        header="Create Helm Release"
        visible={visible}
        onClose={() => setVisible(false)}
        onConfirm={() => form.submit()}
      >
        <Form form={form} onSubmit={handleSubmit} labelWidth={120}>
          <Form.FormItem name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="Release name" />
          </Form.FormItem>
          <Form.FormItem name="targetNamespace" label="Target Namespace" rules={[{ required: true }]}>
            <Input placeholder="Namespace to deploy chart into" defaultValue="default" />
          </Form.FormItem>
          <Form.FormItem name="createNamespace" label="Create Namespace" initialData={true}>
            <Checkbox>Auto-create target namespace if missing</Checkbox>
          </Form.FormItem>
          <Form.FormItem name="chartName" label="Chart Name" rules={[{ required: true }]}>
            <Input placeholder="Chart name (e.g. hello-world)" defaultValue="hello-world" />
          </Form.FormItem>
          <Form.FormItem name="repository" label="Repository" rules={[{ required: true }]}>
            <Input placeholder="Chart repository URL" defaultValue="https://helm.github.io/examples" />
          </Form.FormItem>
          <Form.FormItem name="version" label="Version">
            <Input placeholder="Chart version" defaultValue="0.1.0" />
          </Form.FormItem>
          <Form.FormItem name="values" label="Values (YAML)">
            <Textarea placeholder={'replicaCount: 1\n'} autosize={{ minRows: 3, maxRows: 10 }} />
          </Form.FormItem>
        </Form>
      </Dialog>
    </div>
  );
};

export default HelmReleases;
