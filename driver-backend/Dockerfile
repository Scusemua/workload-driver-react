FROM golang:1.21.5

ENV PORT=9000

# Copy everything to the image.
#COPY vendor /go/pkg/distributed-notebook-dashboard-backend/vendor
#COPY cmd /go/pkg/distributed-notebook-dashboard-backend/cmd
#COPY pkg /go/pkg/distributed-notebook-dashboard-backend/pkg
#COPY internal /go/pkg/distributed-notebook-dashboard-backend/internal
#COPY configs /go/pkg/distributed-notebook-dashboard-backend/configs
#
#WORKDIR /go/pkg/distributed-notebook-dashboard-backend
#
#RUN make build-grpc
#RUN make build-server
#
#COPY configs /go/pkg/distributed-notebook-dashboard-backend/configs

COPY server-linux-amd64 ./server-linux-amd64
COPY configs/config-docker.yaml /etc/distributed-notebook-cluster/dashboard.yaml
COPY .env ./.env

EXPOSE 9000

CMD ["./server-linux-amd64", "--yaml", "/etc/distributed-notebook-cluster/dashboard.yaml"]
