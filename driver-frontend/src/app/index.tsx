import { AppLayout } from '@App/AppLayout/AppLayout';
import { AppRoutes } from '@App/routes';
import { AuthProvider } from '@Providers/AuthProvider';
import { DarkModeProvider, NotificationProvider, SessionTabsDataProvider, WorkloadProvider } from '@src/Providers';
import * as React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import '@src/app/app.css';

const App: React.FunctionComponent = () => {
    return (
        <AuthProvider>
            <DarkModeProvider>
                <NotificationProvider>
                    <WorkloadProvider>
                        <SessionTabsDataProvider>
                            <AppLayout>
                                <AppRoutes />
                            </AppLayout>
                        </SessionTabsDataProvider>
                    </WorkloadProvider>
                </NotificationProvider>
            </DarkModeProvider>
        </AuthProvider>
    );
};

export default App;
