DOCKERUSER = "scusemua"

build-grpc:
	@echo "Building gRPC now."
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/server/api/proto/gateway.proto

build-all: build-grpc build-server

# Alias for build-server. 
build-backend: build-server 

build-server: 
	@echo Building backend server now.
	go build -o server ./cmd/server/main.go 

# Alias for run-server. 
run-backend: run-server 

run-server-spoofed:
	@echo Running backend server now.
	go run ./cmd/server/main.go --yaml ./config-files/config-spoofed.yaml

run-server:
	@echo Running backend server now.
	go run ./cmd/server/main.go --yaml ./config-files/config.yaml