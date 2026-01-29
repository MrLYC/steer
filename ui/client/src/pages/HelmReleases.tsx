import React, { useEffect, useState } from 'react';
import { Table, Button, Tag, Space, DialogPlugin, Dialog, Form, Input, MessagePlugin } from 'tdesign-react';
import { AddIcon, RefreshIcon, DeleteIcon } from 'tdesign-icons-react';
import { helmReleaseApi, HelmRelease } from '../api/client';

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
          await helmReleaseApi.delete(row.metadata.namespace, row.metadata.name);
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
      const newRelease: HelmRelease = {
        apiVersion: 'steer.io/v1alpha1',
        kind: 'HelmRelease',
        metadata: {
          name: values.name,
          namespace: values.namespace,
        },
        spec: {
          chart: {
            name: values.chartName,
            repository: values.repository,
            version: values.version,
          },
          deployment: {
            namespace: values.targetNamespace,
          },
        },
        status: {
          phase: 'Pending',
        },
      };

      try {
        await helmReleaseApi.create(newRelease);
        MessagePlugin.success('Release created successfully');
        setVisible(false);
        form.reset();
        loadReleases();
      } catch (error) {
        MessagePlugin.error('Failed to create release');
      }
    }
  };

  const columns = [
    { colKey: 'metadata.name', title: 'Name' },
    { colKey: 'metadata.namespace', title: 'Namespace' },
    { 
      colKey: 'spec.chart.name', 
      title: 'Chart',
      cell: ({ row }: { row: HelmRelease }) => `${row.spec.chart.name} (${row.spec.chart.version || 'latest'})`
    },
    { 
      colKey: 'status.phase', 
      title: 'Status',
      cell: ({ row }: { row: HelmRelease }) => {
        const theme = row.status.phase === 'Installed' ? 'success' : 
                      row.status.phase === 'Failed' ? 'danger' : 
                      row.status.phase === 'Installing' ? 'warning' : 'primary';
        return <Tag theme={theme}>{row.status.phase}</Tag>;
      }
    },
    { 
      colKey: 'status.deployedAt', 
      title: 'Deployed At',
      cell: ({ row }: { row: HelmRelease }) => row.status.deployedAt ? new Date(row.status.deployedAt).toLocaleString() : '-'
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
          <Form.FormItem name="namespace" label="Namespace" rules={[{ required: true }]}>
            <Input placeholder="Namespace" defaultValue="default" />
          </Form.FormItem>
          <Form.FormItem name="chartName" label="Chart Name" rules={[{ required: true }]}>
            <Input placeholder="Chart name (e.g. nginx)" />
          </Form.FormItem>
          <Form.FormItem name="repository" label="Repository">
            <Input placeholder="Chart repository URL" />
          </Form.FormItem>
          <Form.FormItem name="version" label="Version">
            <Input placeholder="Chart version" />
          </Form.FormItem>
          <Form.FormItem name="targetNamespace" label="Target NS" rules={[{ required: true }]}>
            <Input placeholder="Target deployment namespace" />
          </Form.FormItem>
        </Form>
      </Dialog>
    </div>
  );
};

export default HelmReleases;
