# Distributed Notebook Cluster: Workload Driver

This repository contains a "Workload Driver" for the Distributed Jupyter Notebook Cluster (which is not yet open-sourced).

The Workload Driver provides a web interface containing a dashboard for monitoring the Notebook Cluster. The Workload Driver will also allow users to execute large-scale workloads on the Jupyter Notebook cluster.

## Repository Overview 

The backend is contained within the `internal/` directory. The frontend is contained within the `web/` directory. The entrypoint for the backend is in the `cmd/server/` directory. 

## Quick-start

```bash
git clone https://github.com/Scusemua/workload-driver-react
cd workload-driver-react
npm install && npm run start:dev
```
## Development scripts
```sh
# Install development/build dependencies
npm install

# Start the development server
npm run start:dev

# Run a production build (outputs to "dist" dir)
npm run build

# Run the test suite
npm run test

# Run the test suite with coverage
npm run test:coverage

# Run the linter
npm run lint

# Run the code formatter
npm run format

# Launch a tool to inspect the bundle size
npm run bundle-profile:analyze

# Start the express server (run a production build first)
npm run start

# Start storybook component explorer
npm run storybook

# Build storybook component explorer as standalone app (outputs to "storybook-static" dir)
npm run build:storybook
```

## Configurations
* [TypeScript Config](./tsconfig.json)
* [Webpack Config](./webpack.common.js)
* [Jest Config](./jest.config.js)
* [Editor Config](./.editorconfig)

## Code quality tools
* For accessibility compliance, we use [react-axe](https://github.com/dequelabs/react-axe)
* To keep the bundle size in check, we use [webpack-bundle-analyzer](https://github.com/webpack-contrib/webpack-bundle-analyzer)
* To keep the code formatting in check, we use [prettier](https://github.com/prettier/prettier)
* To keep the code logic and test coverage in check, we use [jest](https://github.com/facebook/jest)
* To ensure code styles remain consistent, we use [eslint](https://eslint.org/)
* To provide a place to showcase custom components, we integrate with [storybook](https://storybook.js.org/)

## Multi environment configuration
This project uses [dotenv-webpack](https://www.npmjs.com/package/dotenv-webpack) for exposing environment variables to your code. Either export them at the system level like `export MY_ENV_VAR=http://dev.myendpoint.com && npm run start:dev` or simply drop a `.env` file in the root that contains your key-value pairs like below:

```sh
ENV_1=http://1.myendpoint.com
ENV_2=http://2.myendpoint.com
```

With that in place, you can use the values in your code like `console.log(process.env.ENV_1);`
