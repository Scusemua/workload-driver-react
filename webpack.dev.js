/* eslint-disable @typescript-eslint/no-var-requires */

const path = require('path');
const { merge } = require('webpack-merge');
const common = require('./webpack.common.js');
const { stylePaths } = require('./stylePaths');
const { debug } = require('console');
const HOST = process.env.HOST || 'localhost';
const PORT = process.env.PORT || '9001';

module.exports = merge(common('development'), {
  mode: 'development',
  devtool: 'eval-source-map',
  devServer: {
    proxy: {
      '/jupyter/*': {
        secure: false,
        logLevel: 'debug',
        pathRewrite: {
          '^/jupyter': '/',
        },
        router: {
          'http://localhost:8000': 'http://localhost:8888',
        },
        target: 'http://localhost:8888',
      },
      '/api/*': {
        context: ['/api'],
        host: 'localhost',
        port: PORT,
        scheme: 'http',
        target: 'http://localhost:8000',
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
  },
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
