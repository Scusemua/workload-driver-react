FROM golang:1.21.5

ENV DASHBOARD_CONFIG_DIR=/etc/dashboard_backend_config_directory

COPY server-linux-amd64 ./server-linux-amd64
COPY configs/custom_event_sequences /etc/distributed-notebook-cluster/custom_event_sequences
COPY configs/config-docker.yaml /etc/distributed-notebook-cluster/dashboard.yaml
COPY .env ./.env

EXPOSE 8000

CMD ["./server-linux-amd64", "--yaml", "$DASHBOARD_CONFIG_DIR/dashboard.yaml"]
