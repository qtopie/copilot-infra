import React from 'react';
import useSWR from 'swr';
import { 
  Title1, 
  makeStyles, 
  shorthands, 
  Card, 
  CardHeader, 
  Text,
  tokens
} from '@fluentui/react-components';
import { ListRegular, WarningRegular, CheckmarkCircleRegular } from '@fluentui/react-icons';
import { api } from '../api/client';
import type { Task } from '../api/types';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
  },
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '20px',
  },
  card: {
    ...shorthands.padding('20px'),
  },
  icon: {
    fontSize: '32px',
    marginBottom: '10px',
  },
});

export const Dashboard: React.FC = () => {
  const styles = useStyles();
  const { data } = useSWR('ListTasks', () => api.listTasks());

  const tasks: Task[] = data?.tasks || [];
  const failed = tasks.filter(t => t.status === 'failed').length;
  const succeeded = tasks.filter(t => t.status === 'succeeded').length;

  return (
    <div className={styles.container}>
      <Title1>Dashboard</Title1>
      
      <div className={styles.statsGrid}>
        <Card className={styles.card}>
          <ListRegular className={styles.icon} primaryFill={tokens.colorBrandForeground1} />
          <CardHeader 
            header={<Text weight="bold" size={400}>Total Tasks</Text>}
            description={<Text size={600}>{tasks.length}</Text>}
          />
        </Card>
        <Card className={styles.card}>
          <CheckmarkCircleRegular className={styles.icon} primaryFill={tokens.colorPaletteGreenForeground1} />
          <CardHeader 
            header={<Text weight="bold" size={400}>Succeeded</Text>}
            description={<Text size={600}>{succeeded}</Text>}
          />
        </Card>
        <Card className={styles.card}>
          <WarningRegular className={styles.icon} primaryFill={tokens.colorPaletteRedForeground1} />
          <CardHeader 
            header={<Text weight="bold" size={400}>Failed</Text>}
            description={<Text size={600}>{failed}</Text>}
          />
        </Card>
      </div>
    </div>
  );
};
