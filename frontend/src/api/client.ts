const BASE_URL = '/api/task.v1.TaskService';

async function request<T>(method: string, body?: any): Promise<T> {
  const options: RequestInit = {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  };
  if (body) {
    options.body = JSON.stringify(body);
  }
  const response = await fetch(`${BASE_URL}/${method}`, options);
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || 'API request failed');
  }
  return response.json();
}

export const api = {
  submitTask: (req: any) => request<any>('SubmitTask', req),
  getTask: (taskId: string) => request<any>('GetTask', { task_id: taskId }),
  getTaskLogs: (taskId: string) => request<any>('GetTaskLogs', { task_id: taskId }),
  searchLogs: (req: any) => request<any>('SearchLogs', req),
  globalSearch: (req: any) => request<any>('GlobalSearch', req),
  listTasks: () => request<any>('ListTasks', {}),
  restartTask: (taskId: string) => request<any>('RestartTask', { task_id: taskId }),
  cancelTask: (taskId: string) => request<any>('CancelTask', { task_id: taskId }),
  registerConnection: (req: any) => request<any>('RegisterConnection', req),
};
