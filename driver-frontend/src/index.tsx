const environment = process.env.NODE_ENV || 'development';
if (environment.trim().toLowerCase() === 'production') {
    __webpack_public_path__ = process.env.PUBLIC_PATH || '/';
}

import App from '@App/index';
import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';

if (process.env.NODE_ENV !== 'production') {
    const config = {
        rules: [
            {
                id: 'color-contrast',
                enabled: false,
            },
        ],
    };
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const axe = require('react-axe');
    axe(React, ReactDOM, 1000, config);
}

const root = ReactDOM.createRoot(document.getElementById('root') as Element);

root.render(
    <React.StrictMode>
        <BrowserRouter>
            <App />
        </BrowserRouter>
    </React.StrictMode>,
);
