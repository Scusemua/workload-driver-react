FROM golang:1.21.5

ENV DASHBOARD_CONFIG_DIR=/etc/dashboard_backend_config_directory

COPY server-linux-amd64 ./server-linux-amd64
COPY configs/custom_event_sequences $DASHBOARD_CONFIG_DIR/custom_event_sequences
COPY configs/config-docker.yaml $DASHBOARD_CONFIG_DIR/dashboard.yaml
COPY .env ./.env
COPY dist ./dist

EXPOSE 8000

CMD ["/bin/bash", "-c", "./server-linux-amd64 -yaml \"$DASHBOARD_CONFIG_DIR/dashboard.yaml\""]
