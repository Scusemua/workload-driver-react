import React from 'react';
import ReactDOM from 'react-dom/client';
import App from '@app/index';
import { DarkModeProvider } from '@app/Providers/DarkModeProvider';
import { NotificationProvider } from '@app/Providers';
import { Toaster } from 'react-hot-toast';

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
                <Toaster
                    position="bottom-right"
                    containerStyle={{
                        zIndex: 9999,
                    }}
                    toastOptions={{
                        className: 'react-hot-toast',
                        style: {
                            zIndex: 9999,
                        },
                    }}
                />
                <App />
            </DarkModeProvider>
        </NotificationProvider>
    </React.StrictMode>,
);
