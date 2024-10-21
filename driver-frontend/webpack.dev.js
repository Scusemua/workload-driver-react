/* eslint-disable @typescript-eslint/no-var-requires */

const path = require('path');
const { merge } = require('webpack-merge');
const common = require('./webpack.common.js');
const { stylePaths } = require('./stylePaths');
const { debug } = require('console');

const HOST = process.env.HOST || '0.0.0.0';
const PORT = Number.parseInt(process.env.PORT) || Number.parseInt('8001');

console.log('HOST: %s. PORT: %s.', HOST, PORT);

devServer = {
    proxy: [
        {
            context: ['/jupyter'],
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
        {
            context: ['/api', '/authenticate', '/refresh_token'],
            host: '127.0.0.1',
            port: PORT,
            scheme: 'http',
            target: 'http://127.0.0.1:8000',
        },
        {
            context: ['/kubernetes'],
            pathRewrite: {
                '^/kubernetes': '/',
            },
            host: '127.0.0.1',
            port: PORT,
            scheme: 'http',
            target: 'http://127.0.0.1:8889',
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

module.exports = (env) => {
    return merge(common('development'), {
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
};
