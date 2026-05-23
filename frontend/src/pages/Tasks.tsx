import React, { useState } from 'react';
import useSWR from 'swr';
import { 
  Title1, 
  DataGrid, 
  DataGridBody, 
  DataGridRow, 
  DataGridHeader, 
  DataGridHeaderCell, 
  DataGridCell,
  type TableColumnDefinition,
  createTableColumn,
  Badge,
  Button,
  Spinner,
  makeStyles,
  shorthands
} from '@fluentui/react-components';
import { PlayRegular, StopRegular, EyeRegular } from '@fluentui/react-icons';
import { api } from '../api/client';
import type { Task } from '../api/types';
import { TaskDetailsDrawer } from '../components/TaskDetailsDrawer';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
  },
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    ...shorthands.padding('10px', '0'),
  },
});

export const Tasks: React.FC = () => {
  const styles = useStyles();
  const { data, error, mutate, isLoading } = useSWR('ListTasks', () => api.listTasks());
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const tasks: Task[] = data?.tasks || [];

  const columns: TableColumnDefinition<Task>[] = [
    createTableColumn<Task>({
      columnId: 'id',
      renderHeaderCell: () => 'ID',
      renderCell: (item) => <span style={{ fontFamily: 'monospace' }}>{item.taskId ? item.taskId.substring(0, 8) : 'N/A'}...</span>,
    }),
    createTableColumn<Task>({
      columnId: 'status',
      renderHeaderCell: () => 'Status',
      renderCell: (item) => {
        let color: 'success' | 'warning' | 'danger' | 'informative' = 'warning';
        if (item.status === 'succeeded') color = 'success';
        if (item.status === 'failed') color = 'danger';
        if (item.status === 'running') color = 'informative';
        return <Badge appearance="filled" color={color}>{item.status}</Badge>;
      },
    }),
    createTableColumn<Task>({
      columnId: 'createdAt',
      renderHeaderCell: () => 'Created At',
      renderCell: (item) => item.createdAt ? new Date(item.createdAt).toLocaleString() : 'N/A',
    }),
    createTableColumn<Task>({
      columnId: 'actions',
      renderHeaderCell: () => 'Actions',
      renderCell: (item) => (
        <div style={{ display: 'flex', gap: '5px' }}>
          <Button 
            size="small" 
            icon={<EyeRegular />} 
            onClick={() => { setSelectedTask(item); setIsDrawerOpen(true); }}
          >
            View
          </Button>
          <Button 
            size="small" 
            icon={<PlayRegular />} 
            onClick={() => api.restartTask(item.taskId).then(() => mutate())}
          >
            Restart
          </Button>
          <Button 
            size="small" 
            icon={<StopRegular />} 
            onClick={() => api.cancelTask(item.taskId).then(() => mutate())}
          >
            Cancel
          </Button>
        </div>
      ),
    }),
  ];

  if (isLoading) return <Spinner label="Loading tasks..." />;
  if (error) return <div>Error loading tasks: {error.message}</div>;

  return (
    <div className={styles.container}>
      <div className={styles.toolbar}>
        <Title1>Tasks</Title1>
        <Button appearance="primary" onClick={() => mutate()}>Refresh</Button>
      </div>
      
      <DataGrid items={tasks} columns={columns}>
        <DataGridHeader>
          <DataGridRow>
            {({ renderHeaderCell }) => (
              <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>
            )}
          </DataGridRow>
        </DataGridHeader>
        <DataGridBody<Task>>
          {({ item, rowId }) => (
            <DataGridRow<Task> key={rowId}>
              {({ renderCell }) => (
                <DataGridCell>{renderCell(item)}</DataGridCell>
              )}
            </DataGridRow>
          )}
        </DataGridBody>
      </DataGrid>

      <TaskDetailsDrawer 
        task={selectedTask} 
        isOpen={isDrawerOpen} 
        onClose={() => setIsDrawerOpen(false)} 
      />
    </div>
  );
};
