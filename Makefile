# Variables
IMAGE_NAME = syzgydb
DOCKER_USER = your_dockerhub_username
TAG = latest

# Default target
all: build

# Build the Docker image
build:
	docker build -t $(DOCKER_USER)/$(IMAGE_NAME):$(TAG) .

# Push the Docker image to Docker Hub
push: build
	docker push $(DOCKER_USER)/$(IMAGE_NAME):$(TAG)

# Clean up local Docker images
clean:
	docker rmi $(DOCKER_USER)/$(IMAGE_NAME):$(TAG)

# Run the Docker container
run:
	docker run -p 8080:8080 $(DOCKER_USER)/$(IMAGE_NAME):$(TAG)

# Login to Docker Hub
login:
	docker login

# Help message
help:
	@echo "Makefile commands:"
	@echo "  make build   - Build the Docker image"
	@echo "  make push    - Push the Docker image to Docker Hub"
	@echo "  make clean   - Remove the local Docker image"
	@echo "  make run     - Run the Docker container"
	@echo "  make login   - Login to Docker Hub"
	@echo "  make help    - Show this help message"
