DOCKERUSER = "scusemua"

build-grpc:
	@echo "Building gRPC now."
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative server/api/proto/gateway.proto

build-all: build-grpc build-server

build-server: 
	@echo Building backend server now.
	go build -o server ./server/cmd/main.go 

run-server:
	@echo Running backend server now.
	go run ./server/cmd/main.go --in-cluster=false --spoof-cluster=true --server-port=8000