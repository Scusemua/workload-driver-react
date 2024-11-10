/* eslint-disable @typescript-eslint/no-var-requires */

const path = require('path');
const { merge } = require('webpack-merge');
const common = require('./webpack.common.js').default;
const { stylePaths } = require('./stylePaths');
const { debug } = require('console');

const HOST = process.env.HOST || '0.0.0.0';
const PORT = Number.parseInt(process.env.PORT) || Number.parseInt('8001');

console.log('Host: %s. Port: %d.', HOST, PORT);

const devServer = {
    proxy: [
        {
            context: ['/jupyter'],
            secure: false,
            logLevel: 'debug',
            pathRewrite: {
                '^/jupyter': '/',
            },
            router: {
                'http://127.0.0.1:8000': 'http://jupyter:8888',
            },
            target: 'http://jupyter:8888',
            "timeout": 60000,
        },
        {
            context: ['/api', '/authenticate', '/refresh_token'],
            host: '127.0.0.1',
            port: PORT,
            scheme: 'http',
            target: 'http://dashboard_backend:8000',
            "timeout": 300000,
        },
        {
            context: ['/ws'],
            host: '127.0.0.1',
            port: PORT,
            ws: true,
            scheme: 'ws',
            target: 'ws://dashboard_backend:8000',
            "timeout": 300000,
        },
        {
            context: ['/kubernetes'],
            pathRewrite: {
                '^/kubernetes': '/',
            },
            host: '127.0.0.1',
            port: PORT,
            scheme: 'http',
            target: 'http://dashboard_backend:8889',
            "timeout": 300000,
        },
    ],
    host: HOST,
    port: PORT,
    historyApiFallback: true,
    open: true,
    static: {
        directory: path.resolve(__dirname, 'dist'),
    },
    client: {
        overlay: true,
    },
    headers: {
        'X-Frame-Options': 'SAMEORIGIN',
    },
};

module.exports = merge(common('development'), {
    mode: 'development',
    devtool: 'eval-source-map',
    entry: path.resolve(__dirname, 'src') + '/index.tsx',
    devServer: devServer,
    module: {
        rules: [
            {
                test: /\.css$/,
                include: [...stylePaths],
                use: ['style-loader', 'css-loader'],
            },
        ],
    },
    snapshot: {
        managedPaths: [],
    },
    // when symlinks.resolve is false, we need this to make sure dev server picks up the changes in the symlinked files and rebuilds
    watchOptions: {
        followSymlinks: true,
    },
    resolve: {
        // Uncomment the following line when working with local packages
        // More reading : https://webpack.js.org/configuration/resolve/#resolvesymlinks
        symlinks: false,
    },
});
