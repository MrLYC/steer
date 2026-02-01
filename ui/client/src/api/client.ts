import axios, { type AxiosResponse } from 'axios';

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
  (response: AxiosResponse) => response,
  (error: unknown) => {
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
    // CR namespace is enforced by the server (operator namespace).
    namespace?: string;
    labels?: Record<string, string>;
  };
  spec: {
    chart: {
      source?: 'repository' | 'git' | 'local';
      repository?: {
        name: string;
        url: string;
        version?: string;
      };
      git?: {
        url: string;
        ref?: string;
        path: string;
      };
      local?: {
        path: string;
      };
    };
    values?: {
      inline?: string;
      valuesFrom?: Array<{
        configMapKeyRef?: { name: string; key: string };
        secretKeyRef?: { name: string; key: string };
      }>;
    };
    deployment: {
      namespace: string;
      createNamespace?: boolean;
      timeout?: string;
      retries?: number;
      waitAfterDeploy?: string;
      autoUninstallAfter?: string;
    };
    cleanup?: {
      deleteNamespace?: boolean;
      deleteImages?: boolean;
    };
  };
  status?: {
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
    // CR namespace is enforced by the server (operator namespace).
    namespace?: string;
    labels?: Record<string, string>;
  };
  spec: {
    helmReleaseRef: {
      name: string;
      // Server enforces HelmReleaseRef namespace = operator namespace.
      namespace?: string;
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
  status?: {
    phase?: string;
    message?: string;
    startTime?: string;
    completionTime?: string;
    testResults?: TestResult[];
    hookResults?: {
      preTest?: HookResult[];
      postTest?: HookResult[];
    };
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
  startedAt?: string;
  completedAt?: string;
  logs?: string;
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
  get: (name: string) => apiClient.get<HelmRelease>(`/helmreleases/${name}`),
  delete: (name: string) => apiClient.delete(`/helmreleases/${name}`),
};

export const helmTestJobApi = {
  list: () => apiClient.get<HelmTestJob[]>('/helmtestjobs'),
  create: (data: HelmTestJob) => apiClient.post<HelmTestJob>('/helmtestjobs', data),
  get: (name: string) => apiClient.get<HelmTestJob>(`/helmtestjobs/${name}`),
  delete: (name: string) => apiClient.delete(`/helmtestjobs/${name}`),
};
