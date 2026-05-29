export interface Task {
  taskId: string;
  type: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  error?: string;
  name?: string;
  command?: string;
  workDir?: string;
  project?: string;
  isLongRunning?: boolean;
  accessUrl?: string;
}

export interface SearchMatch {
  path: string;
  line_num: number;
  text: string;
  context_before?: string[];
  context_after?: string[];
}

export interface SubmitTaskRequest {
  command: string;
  env?: Record<string, string>;
  type?: 'taskfile' | 'infra';
  project?: string;
  name?: string;
  is_long_running?: boolean;
}

export interface RegisterConnectionRequest {
  name: string;
  url: string;
  ns?: string;
  db?: string;
  auth?: string;
}
