FROM node:20.11.1-alpine

ENV PORT=8001

WORKDIR /distributed-notebook-dashboard-frontend/

COPY package.json ./
COPY package-lock.json ./
RUN npm install

# Copy source code folders.
COPY src ./src

# Copy directories.
COPY dr-surge.js ./
COPY jest.config.js ./
COPY setupJest.js ./
COPY stylePaths.js ./
COPY tsconfig.json ./
COPY webpack.dev.docker.js ./webpack.dev.js
COPY webpack.common.js ./
COPY webpack.prod.js ./
COPY .env ./

EXPOSE 8001

CMD ["npm", "run", "start:dev"]
