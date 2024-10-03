# Use the official Golang image as the base image
FROM golang:1.20-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the entire project to the working directory
COPY . .

# Build the Go application
RUN go build -o syzgydb ./syzgydb/main.go

# Expose the port that the application will run on
EXPOSE 8080

# Command to run the executable
CMD ["./syzgydb"]
