const environment = process.env.NODE_ENV || 'development';
if (environment.trim().toLowerCase() === 'production') {
  __webpack_public_path__ = process.env.PUBLIC_PATH || '/';
}

import App from '@App/index';
import { DarkModeProvider } from '@Providers/DarkModeProvider';
import { NotificationProvider } from '@Providers/NotificationProvider';
import React from 'react';
import ReactDOM from 'react-dom/client';

if (process.env.NODE_ENV !== 'production') {
    const config = {
        rules: [
            {
                id: 'color-contrast',
                enabled: false,
            },
        ],
    };
    /* eslint-disable-next-line @typescript-eslint/no-var-requires */
    const axe = require('react-axe');
    axe(React, ReactDOM, 1000, config);
}

const root = ReactDOM.createRoot(document.getElementById('root') as Element);

root.render(
    <React.StrictMode>
        <NotificationProvider>
            <DarkModeProvider>
                <App />
            </DarkModeProvider>
        </NotificationProvider>
    </React.StrictMode>,
);
