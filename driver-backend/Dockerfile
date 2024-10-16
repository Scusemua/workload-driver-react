FROM golang:1.21.5

ENV PORT=9000

# Copy source code folders.
COPY cmd ./
COPY domain ./
COPY generator ./
COPY scripts ./

COPY vendor ./

# Copy resource folders.
COPY config-files ./
COPY custom_event_sequences ./
COPY "workload templates" ./

# Copy files.
COPY go.mod ./
COPY version.go ./

COPY ./server ./
COPY ./configs/config.yaml ./config.yaml

EXPOSE 9000

CMD ["server", "--yaml", "config.yaml"]
