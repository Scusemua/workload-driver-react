FROM golang:1.21.5

ENV DASHBOARD_CONFIG_DIR=/etc/distributed-notebook-cluster

ENV PUBLIC_PATH=/dashboard
ENV ASSET_PATH=/dashboard
ENV DEV_ENVIRONMENT=production

COPY server-linux-amd64 ./server-linux-amd64
COPY configs/custom_event_sequences $DASHBOARD_CONFIG_DIR/custom_event_sequences
COPY configs/config-docker.yaml $DASHBOARD_CONFIG_DIR/dashboard.yaml
COPY configs/workload-presets-file-docker.yaml $DASHBOARD_CONFIG_DIR/workload-presets-file.yaml
COPY configs/workload_templates $DASHBOARD_CONFIG_DIR/workload_templates
COPY .production.env ./.env
COPY dist ./dist

EXPOSE 8000

CMD ["/bin/bash", "-c", "./server-linux-amd64 -yaml \"$DASHBOARD_CONFIG_DIR/dashboard.yaml\""]
