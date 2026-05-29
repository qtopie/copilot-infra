import React, { useEffect, useState } from 'react';
import {
  Drawer,
  DrawerHeader,
  DrawerHeaderTitle,
  DrawerBody,
  Button,
  makeStyles,
  shorthands,
  tokens,
  Text,
  Badge,
  Spinner,
} from '@fluentui/react-components';
import { DismissRegular, ArrowClockwiseRegular } from '@fluentui/react-icons';
import { api } from '../api/client';
import type { Task } from '../api/types';

const useStyles = makeStyles({
  body: {
    display: 'flex',
    flexDirection: 'column',
    gap: '15px',
  },
  section: {
    display: 'flex',
    flexDirection: 'column',
    gap: '5px',
  },
  label: {
    color: tokens.colorNeutralForeground4,
    fontWeight: tokens.fontWeightSemibold,
  },
  logContainer: {
    backgroundColor: tokens.colorNeutralBackground3,
    color: tokens.colorNeutralForeground3,
    fontFamily: 'monospace',
    ...shorthands.padding('10px'),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    overflowX: 'auto',
    whiteSpace: 'pre-wrap',
    maxHeight: '400px',
    overflowY: 'auto',
    fontSize: '12px',
  },
  metadata: {
    display: 'grid',
    gridTemplateColumns: '100px 1fr',
    gap: '10px',
  }
});

interface TaskDetailsDrawerProps {
  task: Task | null;
  isOpen: boolean;
  onClose: () => void;
}

export const TaskDetailsDrawer: React.FC<TaskDetailsDrawerProps> = ({ task, isOpen, onClose }) => {
  const styles = useStyles();
  const [logs, setLogs] = useState<string>('');
  const [loadingLogs, setLoadingLogs] = useState(false);

  useEffect(() => {
    if (isOpen && task) {
      fetchLogs();
    } else {
      setLogs('');
    }
  }, [isOpen, task]);

  const fetchLogs = async () => {
    if (!task) return;
    setLoadingLogs(true);
    try {
      const resp = await api.getTaskLogs(task.taskId);
      setLogs(resp.content || 'No logs available');
    } catch (err) {
      setLogs('Error loading logs: ' + err);
    } finally {
      setLoadingLogs(false);
    }
  };

  if (!task) return null;

  return (
    <Drawer
      position="end"
      separator
      open={isOpen}
      onOpenChange={(_, { open }) => !open && onClose()}
      size="medium"
    >
      <DrawerHeader>
        <DrawerHeaderTitle
          action={
            <Button
              appearance="subtle"
              aria-label="Close"
              icon={<DismissRegular />}
              onClick={onClose}
            />
          }
        >
          Task Details
        </DrawerHeaderTitle>
      </DrawerHeader>

      <DrawerBody className={styles.body}>
        <div className={styles.section}>
          <div className={styles.metadata}>
            <Text className={styles.label}>ID</Text>
            <Text>{task.taskId}</Text>
            
            <Text className={styles.label}>Status</Text>
            <div>
               <Badge appearance="filled" color={task.status === 'succeeded' ? 'success' : task.status === 'failed' ? 'danger' : 'warning'}>
                 {task.status}
               </Badge>
            </div>

            <Text className={styles.label}>Type</Text>
            <Text>{task.type}</Text>

            <Text className={styles.label}>Created</Text>
            <Text>{task.createdAt ? new Date(task.createdAt).toLocaleString() : 'N/A'}</Text>
          </div>
        </div>

        <div className={styles.section}>
          <Text className={styles.label}>Command</Text>
          <div className={styles.logContainer}>{task.command}</div>
        </div>

        {task.error && (
          <div className={styles.section}>
            <Text className={styles.label} style={{ color: tokens.colorPaletteRedForeground1 }}>Error</Text>
            <div className={styles.logContainer} style={{ color: tokens.colorPaletteRedForeground1 }}>
              {task.error}
            </div>
          </div>
        )}

        <div className={styles.section}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Text className={styles.label}>Logs</Text>
            <Button 
              size="small" 
              appearance="subtle" 
              icon={<ArrowClockwiseRegular />} 
              onClick={fetchLogs}
              disabled={loadingLogs}
            >
              Refresh
            </Button>
          </div>
          <div className={styles.logContainer}>
            {loadingLogs ? <Spinner size="tiny" label="Loading logs..." /> : logs}
          </div>
        </div>
      </DrawerBody>
    </Drawer>
  );
};
