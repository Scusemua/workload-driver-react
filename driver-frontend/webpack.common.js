/* eslint-disable @typescript-eslint/no-var-requires */

const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const CopyPlugin = require('copy-webpack-plugin');
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin');
const Dotenv = require('dotenv-webpack');
const BG_IMAGES_DIRNAME = 'bgimages';
const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;

module.exports = (development_environment) => {
    console.log(`Development environment=${development_environment}`);

    return {
        module: {
            rules: [
                {
                    test: /\.(tsx|ts|jsx)?$/,
                    use: [
                        {
                            loader: 'ts-loader',
                            options: {
                                transpileOnly: true,
                                experimentalWatchApi: true,
                            },
                        },
                    ],
                },
                {
                    test: /\.(svg|ttf|eot|woff|woff2)$/,
                    type: 'asset/resource',
                    // only process modules with this loader
                    // if they live under a 'fonts' or 'pficon' directory
                    include: [
                        path.resolve(__dirname, 'node_modules/patternfly/dist/fonts'),
                        path.resolve(__dirname, 'node_modules/@patternfly/react-core/dist/styles/assets/fonts'),
                        path.resolve(__dirname, 'node_modules/@patternfly/react-core/dist/styles/assets/pficon'),
                        path.resolve(__dirname, 'node_modules/@patternfly/patternfly/assets/fonts'),
                        path.resolve(__dirname, 'node_modules/@patternfly/patternfly/assets/pficon'),
                    ],
                },
                {
                    test: /\.svg$/,
                    type: 'asset/inline',
                    include: (input) => input.indexOf('background-filter.svg') > 1,
                    use: [
                        {
                            options: {
                                limit: 5000,
                                outputPath: 'svgs',
                                name: '[name].[ext]',
                            },
                        },
                    ],
                },
                {
                    test: /\.svg$/,
                    // only process SVG modules with this loader if they live under a 'bgimages' directory
                    // this is primarily useful when applying a CSS background using an SVG
                    include: (input) => input.indexOf(BG_IMAGES_DIRNAME) > -1,
                    type: 'asset/inline',
                },
                {
                    test: /\.svg$/,
                    // only process SVG modules with this loader when they don't live under a 'bgimages',
                    // 'fonts', or 'pficon' directory, those are handled with other loaders
                    include: (input) =>
                        input.indexOf(BG_IMAGES_DIRNAME) === -1 &&
                        input.indexOf('fonts') === -1 &&
                        input.indexOf('background-filter') === -1 &&
                        input.indexOf('pficon') === -1,
                    use: {
                        loader: 'raw-loader',
                        options: {},
                    },
                },
                {
                    test: /\.(jpg|jpeg|png|gif|webp)$/i,
                    include: [
                        path.resolve(__dirname, 'src'),
                        path.resolve(__dirname, 'node_modules/patternfly'),
                        path.resolve(__dirname, 'node_modules/@patternfly/patternfly/assets/images'),
                        path.resolve(__dirname, 'node_modules/@patternfly/react-styles/css/assets/images'),
                        path.resolve(__dirname, 'node_modules/@patternfly/react-core/dist/styles/assets/images'),
                        path.resolve(
                            __dirname,
                            'node_modules/@patternfly/react-core/node_modules/@patternfly/react-styles/css/assets/images',
                        ),
                        path.resolve(
                            __dirname,
                            'node_modules/@patternfly/react-table/node_modules/@patternfly/react-styles/css/assets/images',
                        ),
                        path.resolve(
                            __dirname,
                            'node_modules/@patternfly/react-inline-edit-extension/node_modules/@patternfly/react-styles/css/assets/images',
                        ),
                    ],
                    type: 'asset/inline',
                    use: [
                        {
                            options: {
                                limit: 5000,
                                outputPath: 'images',
                                name: '[name].[ext]',
                            },
                        },
                    ],
                },
            ],
        },
        output: {
            filename: '[name].bundle.js',
            path: path.resolve(__dirname, 'dist'),
            publicPath: 'auto',
            clean: true, // Automatically clean the output directory before each build
        },
        plugins: [
            new HtmlWebpackPlugin({
                template: path.resolve(__dirname, 'src', 'index.html'),
                favicon: './src/favicon.png',
                base: {
                    href: development_environment === 'development' ? '/' : '__BASE_URL__',
                },
                baseURL: development_environment === 'development' ? '/' : '__BASE_URL__',
                publicPath: development_environment === 'development' ? '/' : '__BASE_URL__',
            }),
            new Dotenv({
                systemvars: true,
                silent: true,
                path: development_environment === 'development' ? '.development.env' : '.production.env',
            }),
            new CopyPlugin({
                patterns: [{ from: './src/favicon.png', to: 'images' }],
            }),
        ],
        resolve: {
            extensions: ['.js', '.ts', '.tsx', '.jsx'],
            plugins: [
                new TsconfigPathsPlugin({
                    configFile: path.resolve(__dirname, './tsconfig.json'),
                }),
            ],
            symlinks: false,
            cacheWithContext: false,
        },
        optimization: {
            sideEffects: true,
            splitChunks: {
                chunks: 'all',
            },
            usedExports: true,
        },
    };
};
