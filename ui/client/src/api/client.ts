import axios from 'axios';

// API 基础 URL
// In-cluster / same-origin (served by operator's embedded web server)
const API_BASE_URL = '/api/v1';

// 创建 axios 实例
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 响应拦截器处理错误
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error);
    return Promise.reject(error);
  }
);

// 类型定义
export interface HelmRelease {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    namespace: string;
    labels?: Record<string, string>;
  };
  spec: {
    chart: {
      name: string;
      version?: string;
      repository?: string;
      git?: {
        url: string;
        ref?: string;
        path?: string;
        branch?: string;
      };
    };
    values?: any;
    deployment: {
      namespace: string;
      timeout?: string;
      maxRetries?: number;
      waitAfterDeployment?: string;
      autoUninstallAfter?: string;
    };
    cleanup?: {
      deleteNamespace?: boolean;
      deleteImages?: boolean;
    };
  };
  status: {
    phase: string;
    message?: string;
    deployedAt?: string;
  };
}

export interface HelmTestJob {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    namespace: string;
    labels?: Record<string, string>;
  };
  spec: {
    helmReleaseRef: {
      name: string;
      namespace: string;
    };
    schedule: {
      type: 'once' | 'cron';
      delay?: string;
      cron?: string;
      timezone?: string;
    };
    test: {
      timeout?: string;
      logs?: boolean;
      filter?: string;
    };
    hooks?: {
      preTest?: Hook[];
      postTest?: Hook[];
    };
    cleanup?: {
      deleteNamespace?: boolean;
      deleteImages?: boolean;
    };
  };
  status: {
    phase: string;
    message?: string;
    startTime?: string;
    completionTime?: string;
    testResults?: TestResult[];
    hookResults?: HookResult[];
  };
}

export interface Hook {
  name: string;
  type: 'script' | 'kubernetes';
  env?: EnvVar[];
  script?: string;
}

export interface EnvVar {
  name: string;
  value?: string;
  valueFrom?: {
    fieldPath?: string;
    helmReleaseRef?: {
      fieldPath: string;
    };
  };
}

export interface TestResult {
  name: string;
  phase: string;
  startedAt: string;
  completedAt: string;
  message?: string;
}

export interface HookResult {
  name: string;
  phase: string;
  message?: string;
}

// API 方法
export const helmReleaseApi = {
  list: () => apiClient.get<HelmRelease[]>('/helmreleases'),
  create: (data: HelmRelease) => apiClient.post<HelmRelease>('/helmreleases', data),
  get: (namespace: string, name: string) => apiClient.get<HelmRelease>(`/helmreleases/${namespace}/${name}`),
  delete: (namespace: string, name: string) => apiClient.delete(`/helmreleases/${namespace}/${name}`),
};

export const helmTestJobApi = {
  list: () => apiClient.get<HelmTestJob[]>('/helmtestjobs'),
  create: (data: HelmTestJob) => apiClient.post<HelmTestJob>('/helmtestjobs', data),
  get: (namespace: string, name: string) => apiClient.get<HelmTestJob>(`/helmtestjobs/${namespace}/${name}`),
  delete: (namespace: string, name: string) => apiClient.delete(`/helmtestjobs/${namespace}/${name}`),
};
