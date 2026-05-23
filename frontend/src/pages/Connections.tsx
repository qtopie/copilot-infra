import React, { useState } from 'react';
import { 
  Title1, 
  Field, 
  Input, 
  Button, 
  makeStyles, 
  Dialog,
  DialogTrigger,
  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
} from '@fluentui/react-components';
import { AddRegular } from '@fluentui/react-icons';
import { api } from '../api/client';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '10px',
  },
});

export const Connections: React.FC = () => {
  const styles = useStyles();
  const [isOpen, setIsOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    url: '',
    ns: '',
    db: '',
    auth: '',
  });

  const handleSubmit = async () => {
    try {
      await api.registerConnection(formData);
      setIsOpen(false);
      setFormData({ name: '', url: '', ns: '', db: '', auth: '' });
      alert('Connection registered successfully');
    } catch (err: any) {
      alert('Failed to register connection: ' + err.message);
    }
  };

  return (
    <div className={styles.container}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <Title1>Connections</Title1>
        <Dialog open={isOpen} onOpenChange={(_, data) => setIsOpen(data.open)}>
          <DialogTrigger disableButtonEnhancement>
            <Button appearance="primary" icon={<AddRegular />}>Add Connection</Button>
          </DialogTrigger>
          <DialogSurface>
            <DialogBody>
              <DialogTitle>Register New Connection</DialogTitle>
              <DialogContent className={styles.form}>
                <Field label="Name" required>
                  <Input value={formData.name} onChange={(_, d) => setFormData({ ...formData, name: d.value })} />
                </Field>
                <Field label="URL" required>
                  <Input value={formData.url} onChange={(_, d) => setFormData({ ...formData, url: d.value })} />
                </Field>
                <Field label="Namespace">
                  <Input value={formData.ns} onChange={(_, d) => setFormData({ ...formData, ns: d.value })} />
                </Field>
                <Field label="Database">
                  <Input value={formData.db} onChange={(_, d) => setFormData({ ...formData, db: d.value })} />
                </Field>
                <Field label="Auth Token">
                  <Input type="password" value={formData.auth} onChange={(_, d) => setFormData({ ...formData, auth: d.value })} />
                </Field>
              </DialogContent>
              <DialogActions>
                <DialogTrigger disableButtonEnhancement>
                  <Button appearance="secondary">Cancel</Button>
                </DialogTrigger>
                <Button appearance="primary" onClick={handleSubmit}>Register</Button>
              </DialogActions>
            </DialogBody>
          </DialogSurface>
        </Dialog>
      </div>
      <p>Configure Dapr HTTP bindings for external services like SurrealDB.</p>
    </div>
  );
};
