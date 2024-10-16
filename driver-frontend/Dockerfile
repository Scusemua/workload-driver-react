FROM node:20.11.1-alpine

ENV PORT=9000

WORKDIR /distributed-notebook-dashboard-frontend/

# Copy source code folders.
COPY web ./web

# Copy resource folders.
COPY config-files ./
COPY custom_event_sequences ./
COPY "workload templates" ./

# Copy files.
COPY go.mod ./
COPY version.go ./

# Copy directories.
COPY dr-surge.js ./
COPY jest.config.js ./
COPY Makefile ./
COPY package.json ./
COPY package-lock.json ./
COPY setupJest.js ./
COPY stylePaths.js ./
COPY tsconfig.json ./
COPY version.go ./
COPY webpack.dev.js ./
COPY webpack.common.js ./
COPY webpack.prod.js ./

RUN go mod tidy && go mod verify

RUN make build-grpc
RUN make build-server

EXPOSE 9000

CMD ["npm", "run", "start:dev"]
