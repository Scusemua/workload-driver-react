import { AppLayout } from '@App/AppLayout/AppLayout';
import { AppRoutes } from '@App/routes';
import { AuthProvider } from '@Providers/AuthProvider';
import * as React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import '@src/app/app.css';

const App: React.FunctionComponent = () => {
    return (
        <AuthProvider>
            <AppLayout>
              <AppRoutes />
            </AppLayout>
        </AuthProvider>
    );
};

export default App;
