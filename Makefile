DOCKERUSER = "scusemua"

ifeq ($(OS),Windows_NT)     # is Windows_NT on XP, 2000, 7, Vista, 10...
    detected_OS := Windows
else
    detected_OS := $(shell uname)  # same as "uname -s"
endif

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

run-server-prebuilt:
	@echo Running pre-built server now.
	.\server --yaml ./config-files/config.yaml

# NOTE: MUST update version number here prior to running 'make release' and edit this file!
VERS=v0.0.1
PACKAGE=main
GIT_COMMIT=`git rev-parse --short HEAD`
VERS_DATE=`date -u +%Y-%m-%d\ %H:%M`
VERS_FILE=version.go

release:
ifeq ($(detected_OS),Windows)
	rm $(VERS_FILE)
else
	/bin/rm -f $(VERS_FILE)
endif
	@echo "// WARNING: auto-generated by Makefile release target -- run 'make release' to update" > $(VERS_FILE)
	@echo "" >> $(VERS_FILE)
	@echo "package $(PACKAGE)" >> $(VERS_FILE)
	@echo "" >> $(VERS_FILE)
	@echo "const (" >> $(VERS_FILE)
	@echo " Version = \"$(VERS)\"" >> $(VERS_FILE)
	@echo " GitCommit = \"$(GIT_COMMIT)\" // the commit JUST BEFORE the release" >> $(VERS_FILE)
	@echo " VersionDate = \"$(VERS_DATE)\" // UTC" >> $(VERS_FILE)
	@echo ")" >> $(VERS_FILE)
	@echo "" >> $(VERS_FILE)
	goimports -w $(VERS_FILE)
ifeq ($(detected_OS),Windows)
	cat $(VERS_FILE)
else
	/bin/cat $(VERS_FILE)
endif
	git commit -am "$(VERS) release"
	git tag -a $(VERS) -m "$(VERS) release"
	git push
	git push origin --tags
