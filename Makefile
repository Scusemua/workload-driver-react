FRONTEND_DIR="./driver-frontend"
BACKEND_DIR="./driver-backend"
DOCKERUSER=scusemua

# NOTE:
#
# If you're using Windows, then some of these commands may not work if you're using Windows PowerShell
# or Command Prompt. I recommend using a Unix shell, such as Git Bash, on Windows when building these targets.

# Copy the frontend 'dist' directory to the backend
copy-frontend:
	@echo "Removing $(BACKEND_DIR)/dist directory (if it exists)..."
	rm -rf $(BACKEND_DIR)/dist
	@echo "Copying frontend dist to backend..."
	cp -r $(FRONTEND_DIR)/dist $(BACKEND_DIR)/dist

# Build the frontend producing files that can be served statically by the backend
build-frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm run build

# Build the backend server for both Windows and Linux
build-backend-servers:
	@echo "Building backend servers for both Windows and Linux"
	make -C $(BACKEND_DIR) build-servers

# Build the backend server for Linux only
build-backend-linux:
	@echo "Building backend servers for Linux only"
	make -C $(BACKEND_DIR) build-server-linux

# Build the backend server for Windows only
build-backend-windows:
	@echo "Building backend servers for Windows only"
	make -C $(BACKEND_DIR) build-server

# Build the backend Docker image
build-backend-docker:
	@echo "Building Docker image for backend..."
	cd $(BACKEND_DIR) && docker build -t $(DOCKERUSER)/distributed-notebook-dashboard-backend .

# Push the latest backend docker image to Dockerhub.
push-backend-docker:
	@echo "Pushing Docker image for backend..."
	docker push $(DOCKERUSER)/distributed-notebook-dashboard-backend:latest

# Target to build the frontend and the backend server in parallel.
build-static-frontend-and-backend-servers:
	make -j2 build-frontend build-backend-servers

# Default target
all: build-static-frontend-and-backend-servers copy-frontend build-backend-docker push-backend-docker