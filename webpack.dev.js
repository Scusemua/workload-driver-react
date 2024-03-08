/* eslint-disable @typescript-eslint/no-var-requires */

const path = require('path');
const { merge } = require('webpack-merge');
const common = require('./webpack.common.js');
const { stylePaths } = require('./stylePaths');
const { debug } = require('console');

const HOST = process.env.host || '127.0.0.1';
const PORT = process.env.port || '9001';
const SPOOF = process.env.spoof;

console.log('HOST: %s. PORT: %s.', HOST, PORT);

let devServer = undefined;
if (SPOOF) {
    console.log('Returning spoofed webpack dev configuration.');
    devServer = {
        proxy: {
            '/api/*': {
                context: ['/api'],
                host: '127.0.0.1',
                port: PORT,
                scheme: 'http',
                target: 'http://127.0.0.1:8000',
            },
        },
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
    };
} else {
    console.log('Returning non-spoofed webpack dev configuration.');
    devServer = {
        proxy: {
            '/jupyter/*': {
                secure: false,
                logLevel: 'debug',
                pathRewrite: {
                    '^/jupyter': '/',
                },
                router: {
                    'http://127.0.0.1:8000': 'http://127.0.0.1:8888',
                },
                target: 'http://127.0.0.1:8888',
            },
            '/api/*': {
                context: ['/api'],
                host: '127.0.0.1',
                port: PORT,
                scheme: 'http',
                target: 'http://127.0.0.1:8000',
            },
        },
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
    };
}

module.exports = merge(common('development'), {
    mode: 'development',
    devtool: 'eval-source-map',
    entry: path.resolve(__dirname, 'web') + '/index.tsx',
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
});
