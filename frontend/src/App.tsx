import React from 'react';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import { 
  makeStyles, 
  shorthands, 
  tokens,
  Subtitle2
} from '@fluentui/react-components';
import { Dashboard } from './pages/Dashboard';
import { Tasks } from './pages/Tasks';
import { Connections } from './pages/Connections';

const useStyles = makeStyles({
  root: {
    display: 'flex',
    flexDirection: 'column',
    height: '100vh',
    backgroundColor: tokens.colorNeutralBackground1,
  },
  header: {
    ...shorthands.padding('10px', '20px'),
    backgroundColor: tokens.colorNeutralBackground2,
    borderBottom: `1px solid ${tokens.colorNeutralStroke1}`,
    display: 'flex',
    alignItems: 'center',
  },
  container: {
    display: 'flex',
    flexGrow: 1,
  },
  nav: {
    width: '200px',
    backgroundColor: tokens.colorNeutralBackground2,
    borderRight: `1px solid ${tokens.colorNeutralStroke1}`,
    ...shorthands.padding('20px', '10px'),
    display: 'flex',
    flexDirection: 'column',
    gap: '10px',
  },
  content: {
    flexGrow: 1,
    ...shorthands.padding('20px'),
    overflowY: 'auto',
  },
  navLink: {
    ...shorthands.padding('8px', '12px'),
    textDecorationLine: 'none',
    color: tokens.colorNeutralForeground1,
    borderRadius: tokens.borderRadiusMedium,
    ':hover': {
      backgroundColor: tokens.colorNeutralBackground1Hover,
    },
  },
});

const App: React.FC = () => {
  const styles = useStyles();

  return (
    <BrowserRouter>
      <div className={styles.root}>
        <header className={styles.header}>
          <Subtitle2>Copilot-Infra Admin</Subtitle2>
        </header>
        <div className={styles.container}>
          <nav className={styles.nav}>
            <Link to="/" className={styles.navLink}>Dashboard</Link>
            <Link to="/tasks" className={styles.navLink}>Tasks</Link>
            <Link to="/connections" className={styles.navLink}>Connections</Link>
          </nav>
          <main className={styles.content}>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/tasks" element={<Tasks />} />
              <Route path="/connections" element={<Connections />} />
            </Routes>
          </main>
        </div>
      </div>
    </BrowserRouter>
  );
};

export default App;
