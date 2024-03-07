import * as React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import { AppLayout } from '@app/AppLayout/AppLayout';
import { Dashboard } from '@app/Dashboard/Dashboard';
import '@app/app.css';

const App: React.FunctionComponent = () => (
    <AppLayout>
        <Dashboard
            /* Refresh every 2min  */ nodeRefreshInterval={120}
            /* Refresh every 10min */ workloadPresetRefreshInterval={600}
        />
    </AppLayout>
);

export default App;
