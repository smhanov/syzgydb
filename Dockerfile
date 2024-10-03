# Use the official Golang image as the base image
FROM golang:1.21

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the entire project to the working directory
COPY . .

# Set the default data folder environment variable
ENV DATA_FOLDER=/data
RUN cd cmd && go build -o ../syzgydb 

# Expose the port that the application will run on
EXPOSE 8080

# Command to run the executable
CMD ["./syzgydb", "--serve"]
