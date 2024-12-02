FROM golang:1.21.5

RUN apt-get update
RUN apt-get install -y python3-pip python3.11-venv
RUN python3 -m venv py-venv

# Change default shell for subsequent RUN instructions
SHELL ["/bin/bash", "-c"]

# Now, because of the above SHELL directive, this will translate to: RUN /bin/bash -c "..."
RUN source /go/py-venv/bin/activate && pip3 install websockets

ENV DASHBOARD_CONFIG_DIR=/etc/distributed-notebook-cluster

COPY server-linux-amd64 ./server-linux-amd64

COPY configs/custom_event_sequences $DASHBOARD_CONFIG_DIR/custom_event_sequences

COPY configs/workload_templates/fcfs_test.json $DASHBOARD_CONFIG_DIR/workload_templates/fcfs_test.json
COPY configs/workload_templates/four-sessions-varied-training.json $DASHBOARD_CONFIG_DIR/workload_templates/four-sessions-varied-training.json
COPY configs/workload_templates/3-session-8-training-all-simultaneous.json $DASHBOARD_CONFIG_DIR/workload_templates/3-session-8-training-all-simultaneous.json
COPY configs/workload_templates/5-session-8-training-all-simultaneous.json $DASHBOARD_CONFIG_DIR/workload_templates/5-session-8-training-all-simultaneous.json
COPY configs/workload_templates/Template-8Session-39.53IAT-22.95Dur.json $DASHBOARD_CONFIG_DIR/workload_templates/Template-8Session-39.53IAT-22.95Dur.json

COPY configs/workload-templates-file-docker.yaml $DASHBOARD_CONFIG_DIR/workload-templates-file.yaml

COPY dist ./dist

EXPOSE 8000

CMD ["/bin/bash", "-c", "./server-linux-amd64 -yaml \"$DASHBOARD_CONFIG_DIR/dashboard.yaml\""]
