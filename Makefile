# Define the name of the Docker image
IMAGE_NAME = syzgydb
# Define the name of the Docker container
CONTAINER_NAME = syzgydb-container
# Define the port on which the server will run
PORT = 8080

# Declare phony targets to avoid conflicts with files of the same name
.PHONY: stop build run update push-hub

# Stop and remove the running container if it exists
stop:
	docker stop $(CONTAINER_NAME) || true
	docker rm $(CONTAINER_NAME) || true

# Build the Docker image
build:
	docker build -t $(IMAGE_NAME) .

# Run the Docker container
run:
	docker run -d --name $(CONTAINER_NAME) -p $(PORT):8080 -e SERVER_PASSWORD=$${SERVER_PASSWORD:-1234} -e STORAGE_FOLDER=/data -v $(PWD)/data:/data $(IMAGE_NAME)

# Update the Docker container by stopping, building, and running it again
update: stop build run

.PHONY: cmd

cmd: 
	cd cmd && go build -o ../syzgydb

# Push the Docker image to Docker Hub
push-hub: update
	docker tag $(IMAGE_NAME) smhanov/syzgydb:latest
	docker push smhanov/syzgydb:latest
